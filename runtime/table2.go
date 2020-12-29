package runtime

import "math/bits"

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
	items  []hashTableItem
	array  []Value
	length int
	base   uint8
}

func (t *hashTable) insert(k, v Value) {
	if t.length>>t.base != 0 {
		t.grow()
	}
	it := hashTableItem{
		key:   k,
		value: v,
	}
	if insert(t.items, t.base, (1<<t.base)-1, it) {
		t.length++
	}
}

func (t *hashTable) find(k Value) Value {
	return find(t.items, t.base, (1<<t.base)-1, k)
}

func (t *hashTable) delete(k Value) (v Value) {
	v = delete(t.items, t.base, (1<<t.base)-1, k)
	if !v.IsNil() {
		t.length--
	}
	return
}

func (t *hashTable) grow() {
	var (
		bySize    [64]uintptr
		otherKeys uintptr
	)

	// Count positive integer keys by order of magnitude
	for i, v := range t.array {
		if !v.IsNil() {
			bySize[bits.Len(uint(i))]++
		}
	}
	for _, it := range t.items {
		if it.value.IsNil() {
			continue
		}
		if n, ok := it.key.TryInt(); ok && n > 0 {
			bySize[bits.Len(uint(n-1))]++
		} else {
			otherKeys++
		}
	}

	// Find the size of the array
	var (
		acc    = bySize[0]
		arrlen uintptr
		l      uintptr
	)
	otherKeys += acc
	for i, c := range bySize[1:] {
		acc += c
		if acc>>uintptr(i) != 0 {
			l = uintptr(i) + 1
		}
	}
	if l > 0 {
		arrlen = 1 << l
	}

	var (
		base          = t.base + 1
		mask  uintptr = (1 << base) - 1
		items         = make([]hashTableItem, 1<<base)
	)

	// Populate the new
	for _, it := range t.items {
		if !it.value.IsNil() {
			insert(items, base, mask, it)
		}
	}
	t.base = base
	t.items = items
}

func (t *hashTable) get(k Value)

func insert(items []hashTableItem, base uint8, mask uintptr, it hashTableItem) (isNew bool) {
	var (
		i, d   = it.hashKey(base, mask)
		cit    = items[i] // current item
		c      uintptr    // cost of inserting item
		sc     = mask     // cost of swapping
		si, sj uintptr    // swap index
	)
	for cit.key != it.key && !it.nilKey() {
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
	isNew = cit.value.IsNil()
	if cit.key.IsNil() && c > sc {
		items[sj] = items[si]
		i = si
	}
	items[i] = it
	return
}

func find(items []hashTableItem, base uint8, mask uintptr, k Value) Value {
	i, d := hashTableItem{key: k}.hashKey(base, mask)
	it := items[i]
	for it.key != k && !it.value.IsNil() {
		i = (i + d) & mask
	}
	return it.value
}

func delete(items []hashTableItem, base uint8, mask uintptr, k Value) Value {
	i, d := hashTableItem{key: k}.hashKey(base, mask)
	it := items[i]
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
