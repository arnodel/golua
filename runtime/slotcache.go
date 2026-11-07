package runtime

// A constant-key table access, "x.name" in the source, is one site in the
// code and tends to see tables of one layout: instances of a class, built by
// the same constructor with their fields set in the same order, or one module
// or class table.  In golua's hash table the slot of a key depends only on the
// key and the table's size, so at such a site the key is in the same slot of
// every table.  The site keeps a hint, the slot that answered last time
// (Code.slotCache), and passes it to the access, which returns the hint to
// keep for next time.
//
// To the site the hint is opaque.  Inside, it is a guess checked before use:
// the slot must exist in the table and hold the key with a non-nil value.  A
// hit is one comparison in place of a scan of a small table or a hash and a
// probe.  On a miss the full lookup runs and, if it finds the key, its slot
// becomes the hint.  Absence is never remembered, so a key that is not in
// the table takes the __index path as before; and no invalidation is needed,
// as a stale hint fails its check.

// A lookupHint is what a site remembers between two accesses.
type lookupHint struct {
	slot uint32 // where the key was in the table last time
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
	if ok && tbl.hashTable != nil {
		var val Value
		if val, hint.slot = hintedString(tbl.hashTable, k, hint.slot); !val.IsNil() {
			return val, hint, nil
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
