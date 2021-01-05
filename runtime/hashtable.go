package runtime

import (
	"math/bits"
	"unsafe"
)

const uintptrLen = 8 * unsafe.Sizeof(uintptr(0))

//
// Implementation for Lua table.  It is made of two parts, a hash table and an
// array, the latter containing only values with positive integer keys.
//

type mixedTable struct {
	*hashTable
	*array
}

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

func (t *mixedTable) grow() {
	var idxCountByLen [uintptrLen]uintptr

	// Classify the keys in the hashtable
	idxCount := t.hashTable.classifyIndices(&idxCountByLen)

	// If there are no possible index values, just grow the hash table
	if idxCount == 0 {
		t.hashTable = t.hashTable.grow()
		return
	}

	// Find out if we should grow the array
	t.array.classifyIndices(&idxCountByLen)
	arrSize := calculateArraySize(&idxCountByLen)

	// If the array shouldn't grow, grow the hash table
	if arrSize <= t.array.size() {
		t.hashTable = t.hashTable.grow()
		return
	}

	// Grow the array.  That should free capacity in the hashtable
	array := t.array.grow(arrSize)
	for i := range t.hashTable.items {
		it := &t.hashTable.items[i]
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

const smallHashTableSize = 8

const (
	hasNextFlag uintptr = 1
	chainedFlag uintptr = 2
	nextFlags           = hasNextFlag | chainedFlag
)

type hashTableItem struct {
	key, value Value
	next       uintptr
}

func (it hashTableItem) hasNext() bool {
	return it.next&hasNextFlag != 0
}

func (it hashTableItem) nextIndex() uintptr {
	return it.next >> 2
}

func (it hashTableItem) isChained() bool {
	return it.next&chainedFlag != 0
}

func (it hashTableItem) isEmpty() bool {
	return it.key.IsNil()
}

func (it *hashTableItem) setNext(next uintptr, flags uintptr) {
	it.next = next<<2 | flags
}

func (it *hashTableItem) nextFlags() uintptr {
	return it.next & nextFlags
}

const noNextFree uintptr = 1<<uintptrLen - 1

type hashTable struct {
	items    []hashTableItem
	nextFree uintptr
	base     uint8
}

func (t *hashTable) set(k, v Value) {
	if setKeyValue(t.items, (1<<t.base)-1, k, v, t.nextFree) {
		t.nextFree = updateNextFree(t.items, t.nextFree)
	}
}

func (t *hashTable) reset(k, v Value) bool {
	if t == nil {
		return false
	}
	return resetKeyValue(t.items, (1<<t.base)-1, k, v)
}

func (t *hashTable) find(k Value) Value {
	if t == nil {
		return NilValue
	}
	it, _ := findItem(t.items, (1<<t.base)-1, k)
	if it == nil {
		return NilValue
	}
	return it.value
}

func (t *hashTable) removeKey(k Value) (wasSet bool) {
	if t == nil {
		return false
	}
	return removeKey(t.items, (1<<t.base)-1, k)
}

func (t *hashTable) full() bool {
	return t == nil || t.nextFree == noNextFree
}

func (t *hashTable) grow() *hashTable {
	if t == nil {
		return &hashTable{
			items: make([]hashTableItem, 1),
		}
	}
	var (
		base          = t.base + 1
		sz    uintptr = 1 << base
		mask  uintptr = sz - 1
		items         = make([]hashTableItem, sz)
	)

	// Populate the new
	t.nextFree = copyItems(items, t.items, mask, mask)
	t.base = base
	t.items = items
	return t
}

func (t *hashTable) cleanup() {
	if t == nil {
		return
	}
	mask := uintptr(len(t.items) - 1)
	items := make([]hashTableItem, len(t.items))
	t.nextFree = copyItems(items, t.items, mask, mask)
	t.items = items
}

func (t *hashTable) next(k Value) (next Value, v Value, ok bool) {
	if t == nil {
		return
	}

	// Find the starting point
	var i uintptr
	if !k.IsNil() {
		var it *hashTableItem
		it, i = findItem(t.items, (1<<t.base)-1, k)
		if it == nil {
			return
		}
		i++
	}

	// Iterate to the next item
	var nextIt hashTableItem
	for {
		if int(i) >= len(t.items) {
			return NilValue, NilValue, true
		}
		nextIt = t.items[i]
		if !nextIt.value.IsNil() {
			return nextIt.key, nextIt.value, true
		}
		i++
	}
}

func (t *hashTable) classifyIndices(idxCountByLen *[uintptrLen]uintptr) (idxCount uintptr) {
	if t == nil {
		return
	}
	for _, it := range t.items {
		if it.value.IsNil() {
			continue
		}
		if i, ok := it.key.TryInt(); ok && i > 0 {
			idxCountByLen[bits.Len(uint(i-1))]++
			idxCount++
		}
	}
	return
}
func copyItems(items, from []hashTableItem, mask uintptr, nextFree uintptr) uintptr {
	for _, it := range from {
		if !it.value.IsNil() {
			if insertNewKeyValue(items, mask, it.key, it.value, nextFree) {
				nextFree = updateNextFree(items, nextFree)
			}
		}
	}
	return nextFree
}

func setKeyValue(items []hashTableItem, mask uintptr, k, v Value, nextFree uintptr) bool {
	if it, _ := findItem(items, mask, k); it != nil {
		it.value = v
		return false
	}
	return insertNewKeyValue(items, mask, k, v, nextFree)
}

func resetKeyValue(items []hashTableItem, mask uintptr, k, v Value) (wasSet bool) {
	it, _ := findItem(items, mask, k)
	wasSet = it != nil && !it.value.IsNil()
	if wasSet {
		it.value = v
	}
	return
}

func insertNewKeyValue(items []hashTableItem, mask uintptr, k, v Value, nextFree uintptr) bool {
	it := hashTableItem{key: k, value: v}

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
		cit.next |= 2
		items[nextFree] = cit
		it.setNext(nextFree, hasNextFlag)
		items[i] = it
		return true
	}
}

func updateNextFree(items []hashTableItem, nextFree uintptr) uintptr {
	for nextFree != noNextFree && !items[nextFree].isEmpty() {
		nextFree--
	}
	return nextFree
}

func findItem(items []hashTableItem, mask uintptr, k Value) (it *hashTableItem, i uintptr) {
	// For a small table, it's cheaper not to calculate the hash
	if mask < smallHashTableSize {
		for j := int(mask); j >= 0; j-- {
			it = &items[j]
			if it.key.Equals(k) {
				i = uintptr(j)
				return
			}
		}
		return nil, 0
	}
	i = k.Hash() & mask
	it = &items[i]
	for !it.key.Equals(k) {
		if !it.hasNext() {
			return nil, 0
		}
		i = it.nextIndex()
		it = &items[i]
	}
	return
}

func removeKey(items []hashTableItem, mask uintptr, k Value) (wasSet bool) {
	if it, _ := findItem(items, mask, k); it != nil {
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

func (a *array) classifyIndices(idxCountByLen *[uintptrLen]uintptr) {
	if a == nil {
		return
	}
	for i, v := range a.values[:a.len] {
		if !v.IsNil() {
			idxCountByLen[bits.Len(uint(i))]++
		}
	}
}

func calculateArraySize(idxCountByLen *[uintptrLen]uintptr) uintptr {
	var base = -1
	var idxCount uintptr
	for l, c := range idxCountByLen {
		idxCount += c
		if c != 0 && (l == 0 || idxCount >= 1<<(l-1)) {
			base = l
		}
	}
	if base >= 0 {
		return 1 << base
	}
	return 0
}
