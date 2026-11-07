package runtime

// A constant-key table access, "x.name" in the source, is one site in the
// code and tends to see tables of one layout: instances of a class, built by
// the same constructor with their fields set in the same order, or one module
// or class table.  In golua's hash table the slot of a key depends only on the
// key and the table's size, so at such a site the key is in the same slot of
// every table.  The site keeps a hint, the slots that answered last time
// (Code.slotCache), and passes it to the access, which returns the hint to
// keep for next time.
//
// To the site the hint is opaque.  Inside, each slot is a guess checked
// before use: it must exist in the table and hold the key with a non-nil
// value.  A hit is one comparison in place of a scan of a small table or a
// hash and a probe.  On a miss the full lookup runs and, if it finds the key,
// its slot goes into the hint.  Absence is never remembered, so a key that is
// not in the table takes the __index path as before; and no invalidation is
// needed, as a stale hint fails its check.
//
// A method call, "obj:m()", finds m not in the instance but in a class table
// through the metatable's __index, which costs two more lookups: "__index" in
// the metatable, then m in the class table.  Both see one table per site, so
// the hint holds their slots too, and that part of the __index chain is two
// comparisons.

// A lookupHint is what a site remembers between two accesses: where the key
// was in the table accessed, where "__index" was in that table's metatable,
// and where the key was in the table __index names.
type lookupHint struct {
	slot, metaSlot, indexSlot uint32
}

// hintedString returns the value of k in h, nil if absent, trying the hinted
// slot first, and the hint for next time.
func hintedString(h *hashTable, k Value, hint uint32) (Value, uint32) {
	mask := uintptr(1)<<h.base - 1
	if g := uintptr(hint); g <= mask {
		if it := &h.slots[g]; it.key.equalsString(k) && !it.value.IsNil() {
			return it.value, hint
		}
	}
	if it, i := findSlot(h.slots, mask, k, stringKey); it != nil && !it.value.IsNil() {
		return it.value, uint32(i)
	}
	return NilValue, hint
}

// indexStringAt is IndexString with a hint, which it returns updated.
func indexStringAt(t *Thread, coll Value, k Value, hint lookupHint) (Value, lookupHint, error) {
	t.RequireCPU(1)
	tbl, ok := coll.TryTable()
	if !ok {
		val, err := indexChain(t, coll, k, nil, stringKey)
		return val, hint, err
	}
	var val Value
	if tbl.hashTable != nil {
		if val, hint.slot = hintedString(tbl.hashTable, k, hint.slot); !val.IsNil() {
			return val, hint, nil
		}
	}
	// Absent: one level of __index through a table, with the hint's slots;
	// anything else is the general chain.  The depth passed on keeps the
	// chain's length limit where it was.
	if meta := tbl.meta; meta != nil && meta.hashTable != nil {
		var idxVal Value
		idxVal, hint.metaSlot = hintedString(meta.hashTable, metaIndex, hint.metaSlot)
		if idx, ok := idxVal.TryTable(); ok {
			t.RequireCPU(1)
			if idx.hashTable != nil {
				if val, hint.indexSlot = hintedString(idx.hashTable, k, hint.indexSlot); !val.IsNil() {
					return val, hint, nil
				}
			}
			val, err := indexChainFrom(t, idxVal, k, idx, stringKey, 2)
			return val, hint, err
		}
	}
	val, err := indexChain(t, coll, k, tbl, stringKey)
	return val, hint, err
}

// setIndexStringAt is SetIndexString with a hint, which it returns updated.
// Only a store of a non-nil value into a key the table already has takes the
// fast path: setting a key to nil removes it, and a new key may need the
// table to grow.
func setIndexStringAt(t *Thread, coll Value, k, v Value, hint lookupHint) (lookupHint, error) {
	t.RequireCPU(1)
	tbl, isTable := coll.TryTable()
	if isTable && !v.IsNil() && tbl.hashTable != nil {
		h := tbl.hashTable
		mask := uintptr(1)<<h.base - 1
		if g := uintptr(hint.slot); g <= mask {
			if it := &h.slots[g]; it.key.equalsString(k) && !it.value.IsNil() {
				it.value = v
				return hint, nil
			}
		}
		if it, i := findSlot(h.slots, mask, k, stringKey); it != nil && !it.value.IsNil() {
			hint.slot = uint32(i)
			it.value = v
			return hint, nil
		}
		return hint, setIndexChain(t, coll, k, v, tbl, stringKey)
	}
	if isTable && tbl.resetString(k, v) {
		return hint, nil
	}
	return hint, setIndexChain(t, coll, k, v, tbl, stringKey)
}
