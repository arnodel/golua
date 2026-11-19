package runtime

// A constant-key table access, "x.name" in the source, is one site in the
// code and tends to see tables of one layout: instances of a class, built by
// the same constructor with their fields set in the same order, or one module
// or class table.  In golua's hash table the slot of a key depends only on the
// key and the table's size, so at such a site the key is in the same slot of
// every table.  The site keeps a hint (Code.slotCache) that it hands to the
// access, which reads it and, when the lookup learns something, updates it.
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
// through the metatable's __index, and the class table sees one table per
// site, so the hint holds the slot of m in it too.  The same goes for s:m() on a string: the
// string metatable's __index is the string library, and the site remembers
// where m is in it.
//
// A key that is only known when the code runs, "t[k]", uses the site's hint
// in the same way when it is a string: a site that reads a field whose name
// it was given, or looks words up in a dictionary, often sees the same key
// again.  When the key changes from one access to the next the hint misses,
// which costs one comparison before the lookup that would have been made
// anyway.
//
// The hint is passed by pointer, and its slots are plain 16-bit fields:
// returning the hint and storing it on every access measured 3% slower on
// the field-heavy benchmarks, and so did packing the slots into fewer bits.
// A slot index beyond 16 bits is truncated, which the check catches: such a
// table is looked up the ordinary way.

// A lookupHint is what a site remembers between two accesses: where the key
// was in the table accessed, where "__index" was in that table's metatable,
// and where the key was in the table __index names.
type lookupHint struct {
	slot, metaSlot, indexSlot uint16
}

// hintedString returns the value of k in h, nil if absent, trying the
// hinted slot first.  The hit is inline, the miss is a call: this is the
// hottest path of the field-heavy programs.
func hintedString(h *hashTable, k Value, hint *uint16) Value {
	if g := uintptr(*hint); g < uintptr(1)<<h.base {
		if it := &h.slots[g]; it.key.equalsString(k) && !it.value.IsNil() {
			return it.value
		}
	}
	if it := hintMiss(h, k, hint); it != nil {
		return it.value
	}
	return NilValue
}

// hintMiss is the full lookup of k in h after the hint failed, recording the
// slot found in the hint; nil if k is absent.
func hintMiss(h *hashTable, k Value, hint *uint16) *hashTableSlot {
	it, i := findSlot(h.slots, uintptr(1)<<h.base-1, k, stringKey)
	if it == nil || it.value.IsNil() {
		return nil
	}
	*hint = uint16(i)
	return it
}

// indexStringAt is IndexString with a site's hint.
func indexStringAt(t *Thread, coll Value, k Value, hint *lookupHint) (Value, error) {
	t.RequireCPU(1)
	tbl, isTable := coll.TryTable()
	if isTable && tbl.hashTable != nil {
		if val := hintedString(tbl.hashTable, k, &hint.slot); !val.IsNil() {
			return val, nil
		}
	}
	// Absent, or not a table: one level of __index through a table, with
	// the hint; anything else is the general chain.  The depth passed on
	// keeps the chain's length limit where it was.  A table's metatable is
	// one table per site, like the class, so __index is hinted in it too.
	var idxVal Value
	if isTable && tbl.meta != nil && tbl.meta.hashTable != nil {
		idxVal = hintedString(tbl.meta.hashTable, metaNames[metaIndex], &hint.metaSlot)
	} else {
		idxVal = t.metaGet(coll, metaIndex)
	}
	if idx, ok := idxVal.TryTable(); ok {
		t.RequireCPU(1)
		if idx.hashTable != nil {
			if val := hintedString(idx.hashTable, k, &hint.indexSlot); !val.IsNil() {
				return val, nil
			}
		}
		return indexChainFrom(t, TableValue(idx), k, idx, stringKey, 2)
	}
	return indexChain(t, coll, k, tbl, stringKey)
}

// keySlot returns the slot holding key k in h, nil if k has none, trying the
// hinted slot first.  A removed key keeps its slot with a nil value, so the
// slot may be that of an absent key.  The hit is inline, the miss a call.
func keySlot(h *hashTable, k Value, hint *uint16) *hashTableSlot {
	if g := uintptr(*hint); g < uintptr(1)<<h.base {
		if it := &h.slots[g]; it.key.equalsString(k) {
			return it
		}
	}
	return keySlotMiss(h, k, hint)
}

func keySlotMiss(h *hashTable, k Value, hint *uint16) *hashTableSlot {
	it, i := findSlot(h.slots, uintptr(1)<<h.base-1, k, stringKey)
	if it != nil {
		*hint = uint16(i)
	}
	return it
}

// setIndexStringAt is SetIndexString with a site's hint.  A store into a key
// that has a slot, present or removed, writes the slot: a removed key keeps
// its slot, so storing into it again is an insert that needs no room, and
// clearing a present key is what removing it does.  Only an absent key
// consults __newindex; a key with no slot is inserted the ordinary way.
func setIndexStringAt(t *Thread, coll Value, k, v Value, hint *lookupHint) error {
	t.RequireCPU(1)
	tbl, isTable := coll.TryTable()
	if isTable && tbl.hashTable != nil {
		if it := keySlot(tbl.hashTable, k, &hint.slot); it != nil {
			if it.value.IsNil() && tbl.meta != nil && !tbl.meta.metaField(metaNewindex).IsNil() {
				return setIndexChain(t, coll, k, v, tbl, stringKey)
			}
			if it.value.IsNil() {
				tbl.hashTable.metaAbsent = 0 // the key is added back
			}
			it.value = v
			return nil
		}
	}
	return setIndexChain(t, coll, k, v, tbl, stringKey)
}
