package runtime

import (
	"math/bits"
	"unsafe"
)

// Number of bits in an uintptr.
const uintptrLen = 8 * unsafe.Sizeof(uintptr(0))

//
// Implementation for Lua table.  It is made of two parts, a hash table and an
// array, the latter containing only values with positive integer keys.
//
// The Value type needs to satisfy the interface {
//     Hash() uintptr
//     IsNil() bool
//     Equals(Value) bool
// }

type mixedTable struct {
	*hashTable
	*array
}

// newMixedTableWithCapacity creates a mixedTable with preallocated capacity.
// nseq: capacity hint for array part (sequence elements)
// nrec: capacity hint for hash part (record/key-value pairs)
func newMixedTableWithCapacity(nseq, nrec int) *mixedTable {
	var arr *array
	if nseq > 0 {
		// Array can have arbitrary size. The grow() logic handles non-power-of-2
		// sizes by checking if items can actually be moved from hashtable to array.
		arr = &array{
			values: make([]Value, nseq),
			len:    0, // Table is empty, just preallocated
		}
	}

	var hash *hashTable
	if nrec > 0 {
		// Hash table size must be power of 2
		base := uint8(bits.Len(uint(nrec - 1)))
		sz := 1 << base
		hash = &hashTable{
			slots:    make([]hashTableSlot, sz),
			nextFree: uintptr(sz - 1),
			base:     base,
		}
	}

	return &mixedTable{
		array:     arr,
		hashTable: hash,
	}
}

// Return v such that k => v, else return nil.
func (t *mixedTable) get(k Value) Value {
	i, ok := ToIntNoString(k)
	if ok {
		if v, ok := t.array.get(i); ok {
			return v
		}
		k = IntValue(i)
	}
	return t.hashTable.find(k)
}

// Set k => v.
func (t *mixedTable) insert(k, v Value) {
	i, ok := ToIntNoString(k)
	if ok && t.array.setValue(i, v) {
		return
	}
	if t.hashTable.full() {
		t.grow()
		if ok && t.array.setValue(i, v) {
			return
		}
	}
	if ok {
		k = IntValue(i)
	}
	t.hashTable.set(k, v)
}

// Set k => v only if there is already v1 such that k => v1.  Returns true if
// that is the case.
func (t *mixedTable) reset(k, v Value) (wasSet bool) {
	i, ok := ToIntNoString(k)
	if ok {
		ok, wasSet = t.array.resetValue(i, v)
		if ok {
			return
		}
	}
	if ok {
		k = IntValue(i)
	}
	return t.hashTable.reset(k, v)
}

// Set k => nil, return true if there was v such that k => v.
func (t *mixedTable) remove(k Value) (wasSet bool) {
	i, ok := ToIntNoString(k)
	if ok {
		if ok, wasSet = t.array.remove(i); ok {
			return
		}
		k = IntValue(i)
	}

	return t.hashTable.removeKey(k)
}

// Return the "length" of the table, which is a positive integer such i => v but
// (i + 1) => nil.
func (t *mixedTable) len() uintptr {
	l := t.array.getLen()
	if l < t.array.size() {
		return l
	}
	for !t.hashTable.find(IntValue(int64(l + 1))).IsNil() {
		l++
	}
	return l
}

// Return the next key-value after k in the table if it exists (in which case ok
// is true).  If k is nil, return the first key-value in the table.
//
// Provided no new key is inserted between successive calls of next(), then the
// following code will iterate through all the key-value pairs in the table.
//
//	var k Value
//	for {
//	    k, v, ok = t.next(k)
//	    if !ok {
//	        break
//	    }
//	}
func (t *mixedTable) next(k Value) (next Value, v Value, ok bool) {
	var i int64
	var isInt bool
	if k.IsNil() {
		if t.array == nil {
			return t.hashTable.next(k)
		}
		// If there is an array, we pretend that k == 0
		isInt = true
	} else {
		i, isInt = ToIntNoString(k)
		if isInt && i <= 0 {
			isInt = false
		}
	}
	if isInt {
		j, v, ok := t.array.next(i)
		if ok {
			if j > 0 {
				return IntValue(j), v, true
			}
			// In this case we have run out of values in the array, so start the
			// hash table.
			return t.hashTable.next(NilValue)
		}
		k = IntValue(i)
	}
	return t.hashTable.next(k)
}

// Grow the table - either the hash table part or the array part.
//
// The array part grows if there are enough integer keys to justify it.
// When arrSize=0, uses absolute bucketing; when arrSize>0, uses relative
// bucketing. This allows non-power-of-2 array sizes to work correctly.
//
// If the array part doesn't grow, then the hash table part grows by a factor of
// 2.
//
// After growing the array it is guaranteed that there is at least one free slot
// in the hash table part.
func (t *mixedTable) grow() {
	var idxCountByBucket [uintptrLen]uintptr
	arrSize := t.array.size() // 0 if nil

	// Classify the keys in the hashtable
	idxCount := t.hashTable.classifyIndices(&idxCountByBucket, arrSize)

	// If there are no possible index values, just grow the hash table
	if idxCount == 0 {
		t.hashTable = t.hashTable.grow()
		return
	}

	// Calculate optimal new array size
	arrayItems := t.array.itemCount()
	newSize := calculateNewArraySize(&idxCountByBucket, arrSize, arrayItems)

	if newSize == 0 {
		t.hashTable = t.hashTable.grow()
		return
	}

	array := t.array.grow(newSize)
	for i := range t.hashTable.slots {
		it := &t.hashTable.slots[i]
		if it.value.IsNil() {
			continue
		}
		j, ok := it.key.TryInt()
		if ok && array.setValue(j, it.value) {
			it.value = NilValue
		}
	}
	t.array = array
	t.hashTable.cleanup()
}

//
// Hash table implementation
//

// A hashTable contains an array of slots, `S_1, ... S_N` which contain
// key-value pairs.  Each slot can also point at (at most) another slot, which
// we represent as `S_i -> S_j`.
//
// By following the arrow we can form sequences of slots which we call "chains".
//
// There is a function that maps each key `k` to a given slot `S(k)` (in
// practice we use hash(k) % N).  Let's call this slot the "primary slot" of
// `k`.
//
// As we populate the hash table, we keep the following invariants:
//
// (I1) All chains are finite (no cycles).
//
// (I2) All items in a chain have the same primary slot.
//
// (I3) The first item in a chain is in its primary slot.
//
// Having those invariants mean that it is easy to find a key `k` in the table:
// just start at its primary slot and follow the chain until you find a slot
// containing `k` or reach the end of the chain.
//
// To insert a new key-value pair k => v into the table, consider the primary
// slot `S` of `k`.  There are 3 possibilities.
//
// (1) Slot `S` is free.  This is simple: just put (k, v) in this slot.
//
// (2) Slot `S` already contains an item `J` for which it is the primary slot.
// Move it to the next free slot `F` and put the new key-value pair in `S`, so
// that the chain `S -> S'...` becomes `S -> F -> S'...`
//
// (3) Slot `S` contains an item `J` not in its primary position.  Because of
// (I2) there is a chain `...S' -> S -> S”...`.  We move `J` to the next free
// slot `F`, adjusting the chain as `...S' -> F -> S”...`.  That frees slot
// `S`, which means we can put the new item in it.
//
// It is easy to check that in the 3 cases all invariants (I1), (I2) and (I3)
// are preserved.
type hashTable struct {
	slots    []hashTableSlot
	nextFree uintptr
	base     uint8
}

type hashTableSlot struct {
	key, value Value
	next       uintptr // Where to look next for colliding keys (and flags)
}

const (
	hasNextFlag uintptr = 1 // flags that another item is chained after this one
	chainedFlag uintptr = 2 // flags that this item is chained (thus not in primary position)
	nextFlags           = hasNextFlag | chainedFlag
)

const noNextFree uintptr = 1<<uintptrLen - 1

// Small hash tables are treated differently (we bypass hashing the keys).
const smallHashTableSize = 8

func (it hashTableSlot) hasNext() bool {
	return it.next&hasNextFlag != 0
}

func (it hashTableSlot) nextIndex() uintptr {
	return it.next >> 2
}

func (it hashTableSlot) isChained() bool {
	return it.next&chainedFlag != 0
}

func (it hashTableSlot) isEmpty() bool {
	return it.key.IsNil()
}

func (it *hashTableSlot) setNext(next uintptr, flags uintptr) {
	it.next = next<<2 | flags
}

func (it *hashTableSlot) nextFlags() uintptr {
	return it.next & nextFlags
}

func (t *hashTable) set(k, v Value) {
	if setKeyValue(t.slots, (1<<t.base)-1, k, v, t.nextFree) {
		t.nextFree = updateNextFree(t.slots, t.nextFree)
	}
}

func (t *hashTable) reset(k, v Value) bool {
	if t == nil {
		return false
	}
	return resetKeyValue(t.slots, (1<<t.base)-1, k, v)
}

func (t *hashTable) find(k Value) Value {
	if t == nil {
		return NilValue
	}
	it, _ := findSlot(t.slots, (1<<t.base)-1, k)
	if it == nil {
		return NilValue
	}
	return it.value
}

func (t *hashTable) removeKey(k Value) (wasSet bool) {
	if t == nil {
		return false
	}
	return removeKey(t.slots, (1<<t.base)-1, k)
}

func (t *hashTable) full() bool {
	return t == nil || t.nextFree == noNextFree
}

func (t *hashTable) grow() *hashTable {
	if t == nil {
		return &hashTable{
			slots: make([]hashTableSlot, 1),
		}
	}
	var (
		base          = t.base + 1
		sz    uintptr = 1 << base
		mask  uintptr = sz - 1
		items         = make([]hashTableSlot, sz)
	)

	// Populate the new
	t.nextFree = copyItems(items, t.slots, mask, mask)
	t.base = base
	t.slots = items
	return t
}

func (t *hashTable) cleanup() {
	if t == nil {
		return
	}
	mask := uintptr(len(t.slots) - 1)
	items := make([]hashTableSlot, len(t.slots))
	t.nextFree = copyItems(items, t.slots, mask, mask)
	t.slots = items
}

func (t *hashTable) next(k Value) (next Value, v Value, ok bool) {
	if t == nil {
		return NilValue, NilValue, k.IsNil()
	}

	// Find the starting point
	var i uintptr
	if !k.IsNil() {
		var it *hashTableSlot
		it, i = findSlot(t.slots, (1<<t.base)-1, k)
		if it == nil {
			return
		}
		i++
	}

	// Iterate to the next item
	var nextIt hashTableSlot
	for {
		if int(i) >= len(t.slots) {
			return NilValue, NilValue, true
		}
		nextIt = t.slots[i]
		if !nextIt.value.IsNil() {
			return nextIt.key, nextIt.value, true
		}
		i++
	}
}

// Maximum index considered for array storage. Indices beyond this are always
// stored in the hashtable. This prevents overflow in capacity calculations
// and avoids considering impractically large arrays (2^40 = ~1 trillion slots).
const maxArrayIndex = 1 << 40

// classifyIndices buckets positive integer keys from the hashtable.
//
// When arrSize=0 (no array exists), uses absolute bucketing:
//   - Bucket 0: index 1
//   - Bucket 1: index 2
//   - Bucket b: indices [2^(b-1)+1, 2^b]
//
// When arrSize>0, uses relative bucketing (indices > arrSize only):
//   - Bucket 0: indices [arrSize+1, 2*arrSize]
//   - Bucket 1: indices [2*arrSize+1, 4*arrSize]
//   - Bucket b: indices [arrSize*2^b+1, arrSize*2^(b+1)]
//
// Indices beyond maxArrayIndex are ignored (kept in hashtable).
func (t *hashTable) classifyIndices(idxCountByBucket *[uintptrLen]uintptr, arrSize uintptr) (idxCount uintptr) {
	if t == nil {
		return
	}
	for _, it := range t.slots {
		if it.value.IsNil() {
			continue
		}
		if i, ok := it.key.TryInt(); ok && i > 0 && i <= maxArrayIndex {
			var bucket int
			if arrSize == 0 {
				bucket = bits.Len(uint(i - 1))
			} else {
				bucket = bits.Len(uint((i-1)/int64(arrSize))) - 1
			}
			idxCountByBucket[bucket]++
			idxCount++
		}
	}
	return
}
func copyItems(items, from []hashTableSlot, mask uintptr, nextFree uintptr) uintptr {
	for _, it := range from {
		if !it.value.IsNil() {
			if insertNewKeyValue(items, mask, it.key, it.value, nextFree) {
				nextFree = updateNextFree(items, nextFree)
			}
		}
	}
	return nextFree
}

func setKeyValue(items []hashTableSlot, mask uintptr, k, v Value, nextFree uintptr) bool {
	if it, _ := findSlot(items, mask, k); it != nil {
		it.value = v
		return false
	}
	return insertNewKeyValue(items, mask, k, v, nextFree)
}

func resetKeyValue(items []hashTableSlot, mask uintptr, k, v Value) (wasSet bool) {
	it, _ := findSlot(items, mask, k)
	wasSet = it != nil && !it.value.IsNil()
	if wasSet {
		it.value = v
	}
	return
}

func insertNewKeyValue(items []hashTableSlot, mask uintptr, k, v Value, nextFree uintptr) bool {
	it := hashTableSlot{key: k, value: v}

	// Just fill a small table, it's faster than calculating hashes.
	if mask < smallHashTableSize {
		items[nextFree] = it
		return true
	}
	var (
		i   = k.Hash() & mask // primary position for the new item
		cit = items[i]        // item currently at primary position
	)
	switch {
	case cit.isEmpty():
		// The simple case.
		items[i] = it
		return i == nextFree
	case cit.isChained():
		// Move new item into primary position, move colliding item into free position.
		pidx := cit.key.Hash() & mask
		pit := &items[pidx]
		for nidx := pit.nextIndex(); nidx != i; nidx = pit.nextIndex() {
			pidx = nidx
			pit = &items[pidx]
		}
		items[nextFree] = cit
		items[i] = it
		pit.setNext(nextFree, pit.nextFlags()|hasNextFlag)
		return true
	default:
		// Colliding item is in primary position, put new item into free position.
		cit.next |= chainedFlag
		items[nextFree] = cit
		it.setNext(nextFree, hasNextFlag)
		items[i] = it
		return true
	}
}

func updateNextFree(slots []hashTableSlot, nextFree uintptr) uintptr {
	for nextFree != noNextFree && !slots[nextFree].isEmpty() {
		nextFree--
	}
	return nextFree
}

func findSlot(slots []hashTableSlot, mask uintptr, k Value) (it *hashTableSlot, i uintptr) {
	// For a small table, it's cheaper not to calculate the hash
	if mask < smallHashTableSize {
		for j := int(mask); j >= 0; j-- {
			it = &slots[j]
			if it.key.Equals(k) {
				i = uintptr(j)
				return
			}
		}
		return nil, 0
	}
	i = k.Hash() & mask
	it = &slots[i]
	if it.isChained() {
		return nil, 0
	}
	for !it.key.Equals(k) {
		if !it.hasNext() {
			return nil, 0
		}
		i = it.nextIndex()
		it = &slots[i]
	}
	return
}

func removeKey(slots []hashTableSlot, mask uintptr, k Value) (wasSet bool) {
	if it, _ := findSlot(slots, mask, k); it != nil {
		wasSet = !it.value.IsNil()
		it.value = NilValue
	}
	return
}

//
// array implemetation
//

type array struct {
	values []Value
	len    uintptr
}

func (a *array) get(i int64) (v Value, ok bool) {
	ok = a != nil && 1 <= i && i <= int64(len(a.values))
	if ok {
		v = a.values[i-1]
	}
	return
}

func (a *array) setValue(i int64, v Value) (ok bool) {
	ok = a != nil && 1 <= i && i <= int64(len(a.values))
	if ok {
		a.values[i-1] = v
		if a.len < uintptr(i) {
			a.len = uintptr(i)
		}
	}
	return
}

func (a *array) resetValue(i int64, v Value) (ok bool, wasSet bool) {
	ok = a != nil && 1 <= i && i <= int64(len(a.values))
	if ok {
		wasSet = !a.values[i-1].IsNil()
		if wasSet {
			a.values[i-1] = v
		}
	}
	return
}

func (a *array) remove(i int64) (ok bool, wasSet bool) {
	ok = a != nil && 1 <= i && i <= int64(len(a.values))
	if !ok {
		return
	}
	wasSet = int64(a.len) >= i && !a.values[i-1].IsNil()
	if !wasSet {
		return
	}
	a.values[i-1] = NilValue
	l := uintptr(i)
	if a.len == l {
		for l >= 1 && a.values[l-1].IsNil() {
			l--
		}
		a.len = l
	}
	return
}

func (a *array) size() uintptr {
	if a == nil {
		return 0
	}
	return uintptr(len(a.values))
}

func (a *array) getLen() uintptr {
	if a == nil {
		return 0
	}
	return a.len
}

func (a *array) next(i int64) (next int64, v Value, ok bool) {
	ok = a != nil && 0 <= i && i <= int64(a.len)
	if !ok {
		return
	}
	for {
		if i == int64(a.len) {
			return
		}
		v = a.values[i]
		i++
		if !v.IsNil() {
			next = i
			return
		}
	}
}

func (a *array) grow(sz uintptr) *array {
	values := make([]Value, sz)
	if a == nil {
		return &array{values: values}
	}
	copy(values, a.values)
	a.values = values
	return a
}

// itemCount returns the number of non-nil items in the array.
func (a *array) itemCount() uintptr {
	if a == nil {
		return 0
	}
	var count uintptr
	for _, v := range a.values[:a.len] {
		if !v.IsNil() {
			count++
		}
	}
	return count
}

// calculateNewArraySize determines the optimal array size based on index distribution.
// Returns 0 if no array should be created/grown, otherwise returns the new size.
//
// arrayItems is the count of non-nil items currently in the array.
// idxCountByBucket contains only hashtable items (indices outside current array).
//
// When arrSize=0 (absolute mode):
//   - Bucket b covers indices up to 2^b, so capacity = 2^b
//   - Threshold for 50% density = 2^(b-1), or 1 for b=0
//
// When arrSize>0 (relative mode):
//   - Bucket b covers indices up to arrSize * 2^(b+1), so capacity = arrSize * 2^(b+1)
//   - Threshold for 50% density = arrSize * 2^b
func calculateNewArraySize(idxCountByBucket *[uintptrLen]uintptr, arrSize, arrayItems uintptr) uintptr {
	var newSize uintptr
	idxCount := arrayItems

	for b, c := range idxCountByBucket {
		idxCount += c
		if c == 0 {
			continue
		}

		var threshold, capacity uintptr
		if arrSize == 0 {
			// Absolute: capacity = 2^b, threshold = 2^(b-1) (or 1 for b=0)
			capacity = 1 << b
			if b == 0 {
				threshold = 1
			} else {
				threshold = 1 << (b - 1)
			}
		} else {
			// Relative: capacity = arrSize * 2^(b+1), threshold = arrSize * 2^b
			capacity = arrSize << (b + 1)
			threshold = arrSize << b
		}

		// Only grow if density threshold is met
		if idxCount >= threshold {
			newSize = capacity
		}
	}
	return newSize
}
