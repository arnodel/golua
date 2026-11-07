package runtime

// A constant-key table access, "x.name" in the source, is one site in the
// code and tends to see tables of one layout: instances of a class, built by
// the same constructor with their fields set in the same order, or one module
// or class table.  In golua's hash table the slot of a key depends only on the
// key and the table's size, so at such a site the key is in the same slot of
// every table, and the site remembers that slot (Code.slotCache).
//
// The remembered slot is a guess, checked before use: the slot must exist in
// this table and hold the key with a non-nil value.  A hit is one comparison
// in place of a scan of a small table or a hash and a probe.  On a miss the
// full lookup runs and, if it finds the key, records its slot for next time.
// Absence is never remembered, so a key that is not in the table takes the
// full path, including __index, as before; and no invalidation is needed, as
// a stale guess fails the check.  A guess beyond 32 bits of slot index can't
// happen: tables are smaller than that.

// indexStringAt is IndexString for a site whose guess at the key's slot is
// *guess.
func indexStringAt(t *Thread, coll Value, k Value, guess *uint32) (Value, error) {
	t.RequireCPU(1)
	tbl, ok := coll.TryTable()
	if ok && tbl.hashTable != nil {
		h := tbl.hashTable
		mask := uintptr(1)<<h.base - 1
		if g := uintptr(*guess); g <= mask {
			if it := &h.slots[g]; it.key.equalsString(k) && !it.value.IsNil() {
				return it.value, nil
			}
		}
		if it, i := findSlot(h.slots, mask, k, stringKey); it != nil && !it.value.IsNil() {
			*guess = uint32(i)
			return it.value, nil
		}
	}
	return indexChain(t, coll, k, tbl, stringKey)
}

// setIndexStringAt is SetIndexString for a site whose guess at the key's
// slot is *guess.  Only a store of a non-nil value into a key the table
// already has takes the fast path: setting a key to nil removes it, and a
// new key may need the table to grow.
func setIndexStringAt(t *Thread, coll Value, k, v Value, guess *uint32) error {
	t.RequireCPU(1)
	tbl, isTable := coll.TryTable()
	if isTable && !v.IsNil() && tbl.hashTable != nil {
		h := tbl.hashTable
		mask := uintptr(1)<<h.base - 1
		if g := uintptr(*guess); g <= mask {
			if it := &h.slots[g]; it.key.equalsString(k) && !it.value.IsNil() {
				it.value = v
				return nil
			}
		}
		if it, i := findSlot(h.slots, mask, k, stringKey); it != nil && !it.value.IsNil() {
			*guess = uint32(i)
			it.value = v
			return nil
		}
		return setIndexChain(t, coll, k, v, tbl, stringKey)
	}
	if isTable && tbl.resetString(k, v) {
		return nil
	}
	return setIndexChain(t, coll, k, v, tbl, stringKey)
}
