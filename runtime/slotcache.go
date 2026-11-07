package runtime

// A constant-key table access, "x.name" in the source, is one site in the
// code and tends to see tables of one layout: instances of a class, built by
// the same constructor with their fields set in the same order, or one module
// or class table.  In golua's hash table the slot of a key depends only on the
// key and the table's size, so at such a site the key is in the same slot of
// every table, and the site remembers that slot (Code.slotCache).
//
// A remembered slot is a guess, checked before use: the slot must exist in
// the table and hold the key with a non-nil value.  A hit is one comparison
// in place of a scan of a small table or a hash and a probe.  On a miss the
// full lookup runs and, if it finds the key, records its slot for next time.
// Absence is never remembered, so a key that is not in the table takes the
// __index path as before; and no invalidation is needed, as a stale guess
// fails its check.
//
// A method call, "obj:m()", finds m not in the instance but in a class table
// through the metatable's __index, which costs two more lookups: "__index" in
// the metatable, then m in the class table.  Both see one table per site, so
// the site remembers their slots too, and that part of the __index chain is
// two comparisons.

// A siteGuess holds a site's remembered slots: of the key in the table
// accessed, of "__index" in that table's metatable, and of the key in the
// table __index names.
type siteGuess struct {
	own, meta, index uint32
}

// guessedString returns the value of k in h, nil if absent, checking the
// guessed slot first and recording the slot found.
func guessedString(h *hashTable, k Value, guess *uint32) Value {
	mask := uintptr(1)<<h.base - 1
	if g := uintptr(*guess); g <= mask {
		if it := &h.slots[g]; it.key.equalsString(k) && !it.value.IsNil() {
			return it.value
		}
	}
	if it, i := findSlot(h.slots, mask, k, stringKey); it != nil && !it.value.IsNil() {
		*guess = uint32(i)
		return it.value
	}
	return NilValue
}

// indexStringAt is IndexString for a site with guesses g.
func indexStringAt(t *Thread, coll Value, k Value, g *siteGuess) (Value, error) {
	t.RequireCPU(1)
	tbl, ok := coll.TryTable()
	if !ok {
		return indexChain(t, coll, k, nil, stringKey)
	}
	if tbl.hashTable != nil {
		if val := guessedString(tbl.hashTable, k, &g.own); !val.IsNil() {
			return val, nil
		}
	}
	// Absent: one level of __index through a table, with the site's guesses;
	// anything else is the general chain.  The depth passed on keeps the
	// chain's length limit where it was.
	if meta := tbl.meta; meta != nil && meta.hashTable != nil {
		if idx, ok := guessedString(meta.hashTable, metaIndex, &g.meta).TryTable(); ok {
			t.RequireCPU(1)
			if idx.hashTable != nil {
				if val := guessedString(idx.hashTable, k, &g.index); !val.IsNil() {
					return val, nil
				}
			}
			return indexChainFrom(t, TableValue(idx), k, idx, stringKey, 2)
		}
	}
	return indexChain(t, coll, k, tbl, stringKey)
}

// setIndexStringAt is SetIndexString for a site with guesses g.  Only a store
// of a non-nil value into a key the table already has takes the fast path:
// setting a key to nil removes it, and a new key may need the table to grow.
func setIndexStringAt(t *Thread, coll Value, k, v Value, g *siteGuess) error {
	t.RequireCPU(1)
	tbl, isTable := coll.TryTable()
	if isTable && !v.IsNil() && tbl.hashTable != nil {
		h := tbl.hashTable
		mask := uintptr(1)<<h.base - 1
		if gi := uintptr(g.own); gi <= mask {
			if it := &h.slots[gi]; it.key.equalsString(k) && !it.value.IsNil() {
				it.value = v
				return nil
			}
		}
		if it, i := findSlot(h.slots, mask, k, stringKey); it != nil && !it.value.IsNil() {
			g.own = uint32(i)
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
