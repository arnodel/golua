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
// through the metatable's __index, which costs two more lookups: "__index" in
// the metatable, then m in the class table.  Both see one table per site, so
// the hint holds their slots too, and that part of the __index chain is two
// comparisons.
//
// The hint is passed by pointer rather than returned: returning it and
// storing it on every access measured 3% slower on the field-heavy
// benchmarks, a third of what the hint gains there.  Its three slots are
// packed in 32 bits, 4 bytes per opcode of every function: the tables seen
// at the three positions were measured over the benchmarks, with the
// globals table the biggest at the first.

// A lookupHint is what a site remembers between two accesses: where the key
// was in the table accessed (12 bits: instances, modules, and the globals
// table, which grows with the program), where "__index" was in that table's
// metatable (8 bits: a one-field metatable, or the class itself), and where
// the key was in the table __index names (12 bits: a class or a module).  A
// field of all ones means no hint, which a table bigger than the field
// leaves behind.
type lookupHint uint32

// The three fields of a lookupHint: their offset and width.
const (
	ownHint   = 0
	metaHint  = 12
	indexHint = 20
	hintBits  = 0x0c080c // widths, 12 8 12, read by width
)

func hintWidth(field uint) uint { return uint(hintBits>>field) & 0xff }

// hintedString returns the value of k in h, nil if absent, trying the slot
// in the given field of the hint first.  The hit is inline, the miss is a
// call: this is the hottest path of the field-heavy programs.
func hintedString(h *hashTable, k Value, hint *lookupHint, field uint) Value {
	none := uint32(1)<<hintWidth(field) - 1
	if g := uintptr(uint32(*hint>>field) & none); g < uintptr(none) && g < uintptr(1)<<h.base {
		if it := &h.slots[g]; it.key.equalsString(k) && !it.value.IsNil() {
			return it.value
		}
	}
	if it := hintMiss(h, k, hint, field, none); it != nil {
		return it.value
	}
	return NilValue
}

// hintMiss is the full lookup of k in h after the hint failed, recording the
// slot found in the hint's field; nil if k is absent.
func hintMiss(h *hashTable, k Value, hint *lookupHint, field uint, none uint32) *hashTableSlot {
	it, i := findSlot(h.slots, uintptr(1)<<h.base-1, k, stringKey)
	if it == nil || it.value.IsNil() {
		return nil
	}
	slot := uint32(i)
	if slot >= none {
		slot = none
	}
	*hint = *hint&^(lookupHint(none)<<field) | lookupHint(slot)<<field
	return it
}

// indexStringAt is IndexString with a site's hint.
func indexStringAt(t *Thread, coll Value, k Value, hint *lookupHint) (Value, error) {
	t.RequireCPU(1)
	tbl, ok := coll.TryTable()
	if !ok {
		return indexChain(t, coll, k, nil, stringKey)
	}
	if tbl.hashTable != nil {
		if val := hintedString(tbl.hashTable, k, hint, ownHint); !val.IsNil() {
			return val, nil
		}
	}
	// Absent: one level of __index through a table, with the hint's slots;
	// anything else is the general chain.  The depth passed on keeps the
	// chain's length limit where it was.
	if meta := tbl.meta; meta != nil && meta.hashTable != nil {
		idxVal := hintedString(meta.hashTable, metaIndex, hint, metaHint)
		if idx, ok := idxVal.TryTable(); ok {
			t.RequireCPU(1)
			if idx.hashTable != nil {
				if val := hintedString(idx.hashTable, k, hint, indexHint); !val.IsNil() {
					return val, nil
				}
			}
			return indexChainFrom(t, idxVal, k, idx, stringKey, 2)
		}
	}
	return indexChain(t, coll, k, tbl, stringKey)
}

// setIndexStringAt is SetIndexString with a site's hint.  Only a store of a
// non-nil value into a key the table already has takes the fast path:
// setting a key to nil removes it, and a new key may need the table to grow.
func setIndexStringAt(t *Thread, coll Value, k, v Value, hint *lookupHint) error {
	t.RequireCPU(1)
	tbl, isTable := coll.TryTable()
	if isTable && !v.IsNil() && tbl.hashTable != nil {
		h := tbl.hashTable
		none := uint32(1)<<hintWidth(ownHint) - 1
		if g := uintptr(uint32(*hint>>ownHint) & none); g < uintptr(none) && g < uintptr(1)<<h.base {
			if it := &h.slots[g]; it.key.equalsString(k) && !it.value.IsNil() {
				it.value = v
				return nil
			}
		}
		if it := hintMiss(h, k, hint, ownHint, none); it != nil {
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
