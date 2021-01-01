package runtime

import "math/bits"

type mixedTable struct {
	hashTable
	array    []Value
	arraylen uintptr
}

func (t *mixedTable) insert(k, v Value) {
	i, ok := ToIntNoString(k)
	if ok && 1 <= i && i <= int64(len(t.array)) {
		t.array[i-1] = v
		t.arraylen++
		return
	}
	if t.hashTable.base == 0 || t.hashTable.len > 1<<(t.hashTable.base-1) {
		t.grow()
		if ok && 0 < i && i <= int64(len(t.array)) {
			t.array[i-1] = v
			t.arraylen++
			return
		}
	}
	t.hashTable.insert(k, v)
}

func (t *mixedTable) grow() {
	var idxCountByLen [64]uintptr
	var keyCount uintptr

	// Classify the keys in the hashtable
	for _, it := range t.hashTable.items {
		if it.value.IsNil() {
			continue
		}
		if i, ok := it.key.TryInt(); ok && i > 0 {
			idxCountByLen[bits.Len(uint(i-1))]++
		} else {
			keyCount++
		}
	}

	// If there are no possible index values, just grow the hash table
	if keyCount == t.hashTable.len {
		t.hashTable.grow()
		return
	}

	// Find out if we should grow the array
	for i, v := range t.array {
		if v.IsNil() {
			continue
		}
		idxCountByLen[bits.Len(uint(i))]++
	}

	var idxCount uintptr
	var base = -1
	var arrlen int
	for l, c := range idxCountByLen {
		idxCount += c
		if c != 0 && idxCount>>l != 0 {
			base = l
		}
	}
	if base >= 0 {
		arrlen = 1 << base
	}

	// If the array shouldn't grow, grow the hash table
	if arrlen <= len(t.array) {
		t.hashTable.grow()
		return
	}

	// Grow the array.  That should free capacity in the hashtable
	array := append(t.array, make([]Value, arrlen-len(t.array))...)
	for i, it := range t.hashTable.items {
		if it.value.IsNil() {
			continue
		}
		j, ok := it.key.TryInt()
		if ok && j > 0 && j <= int64(arrlen) {
			array[j-1] = it.value
			t.hashTable.items[i].value = NilValue
		}
	}
}

type hashTableItem struct {
	key, value Value
}

func (it hashTableItem) hashKey(base uint8, mask uintptr) (uintptr, uintptr) {
	h := it.key.Hash()
	return h & mask, (h>>base)&mask | 1
}

func (it hashTableItem) nilKey() bool {
	return it.key.IsNil()
}

type hashTable struct {
	items []hashTableItem
	free  uintptr
	base  uint8
}

func (t *hashTable) insert(k, v Value) {
	it := hashTableItem{
		key:   k,
		value: v,
	}
	if insert(t.items, t.base, (1<<t.base)-1, it) {
		t.free--
	}
}

func (t *hashTable) find(k Value) Value {
	return find(t.items, t.base, (1<<t.base)-1, k)
}

func (t *hashTable) delete(k Value) (v Value) {
	delete(t.items, t.base, (1<<t.base)-1, k)
	return
}

func (t *hashTable) grow() {
	var (
		base          = t.base + 1
		sz    uintptr = 1 << base
		mask  uintptr = sz - 1
		items         = make([]hashTableItem, sz)
	)

	// Populate the new
	t.free = sz - copy(items, t.items, base, mask)
	t.base = base
	t.items = items
}

func copy(items, from []hashTableItem, base uint8, mask uintptr) (count uintptr) {
	for _, it := range from {
		if !it.value.IsNil() {
			if insert(items, base, mask, it) {
				count++
			}
		}
	}
	return
}

func insert(items []hashTableItem, base uint8, mask uintptr, it hashTableItem) (isNew bool) {
	var (
		i, d   = it.hashKey(base, mask)
		cit    hashTableItem // current item
		c      uintptr       // cost of inserting item
		sc     = mask        // cost of swapping
		si, sj uintptr       // swap index
	)
	for {
		cit = items[i]
		isNew = cit.key.IsNil()
		if isNew || cit.key == it.key {
			break
		}
		if c >= 2 {
			if scc, sjc := shiftCost(items, base, mask, i); c+scc < sc {
				sc = c + scc
				si = i
				sj = sjc
			}
		}
		c++
		i = (i + d) & mask
		cit = items[i]
	}
	if isNew && c > sc {
		items[sj] = items[si]
		i = si
	}
	items[i] = it
	return
}

func find(items []hashTableItem, base uint8, mask uintptr, k Value) Value {
	var (
		i, d = hashTableItem{key: k}.hashKey(base, mask)
		it   = items[i]
	)
	for it.key != k && !it.value.IsNil() {
		i = (i + d) & mask
	}
	return it.value
}

func delete(items []hashTableItem, base uint8, mask uintptr, k Value) Value {
	var (
		i, d = hashTableItem{key: k}.hashKey(base, mask)
		it   = items[i]
	)
	for it.key != k && !it.value.IsNil() {
		i = (i + d) & mask
	}
	items[i].value = NilValue
	return it.value
}

func shiftCost(items []hashTableItem, base uint8, mask uintptr, i uintptr) (uintptr, uintptr) {
	var (
		_, d = items[i].hashKey(base, mask)
		c    uintptr
	)
	for {
		i = (i + d) & mask
		c++
		if items[i].nilKey() {
			return c, i
		}
	}
}
