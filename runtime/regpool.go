package runtime

import "math/bits"

// Slices of registers, arguments and cells are reused across continuations so
// that calling a function does not allocate.  A slicePool keeps released
// slices in size classes, powers of two from 8 to 1024, each a small LIFO, so
// that a request is served by the most recently released slice of the class
// with enough capacity.  Larger slices are allocated exactly and not pooled.
//
// A pooled slice is zero beyond its length: a slice is cleared up to its
// length when released, and nothing writes beyond a slice's length while it
// is in use, so this holds from allocation on.  A slice the pool hands out is
// therefore all zeros, as a fresh allocation would be.

const (
	minPoolClass = 3  // Smallest class holds slices of capacity 8
	maxPoolClass = 10 // Largest class holds slices of capacity 1024
	poolClasses  = maxPoolClass - minPoolClass + 1
	regPoolSize  = 16 // Default number of slices kept per class
)

// A slicePool reuses slices of T, by size class.
type slicePool[T any] struct {
	free [poolClasses][][]T
	max  int // Number of slices kept per class
}

type valuePool = slicePool[Value]
type cellPool = slicePool[Cell]

func mkValuePool(max uint) valuePool {
	return valuePool{max: int(max)}
}

func mkCellPool(max uint) cellPool {
	return cellPool{max: int(max)}
}

// poolClass returns the class of slices with capacity for n items and that
// capacity, or a negative class and n itself if n is too large to be pooled.
// The class is the smallest power of two not below n, from bits.Len.
func poolClass(n int) (class, size int) {
	if n <= 1<<minPoolClass {
		return 0, 1 << minPoolClass
	}
	class = bits.Len(uint(n-1)) - minPoolClass
	if class >= poolClasses {
		return -1, n
	}
	return class, 1 << (class + minPoolClass)
}

// get returns a zeroed slice of length n.
func (p *slicePool[T]) get(n int) []T {
	class, size := poolClass(n)
	if class >= 0 {
		if free := p.free[class]; len(free) > 0 {
			s := free[len(free)-1]
			free[len(free)-1] = nil
			p.free[class] = free[:len(free)-1]
			return s[:n]
		}
	}
	return make([]T, n, size)
}

// release clears s and keeps it for reuse, unless its capacity is not that
// of a class (it was grown, or is too large) or the class has enough already.
func (p *slicePool[T]) release(s []T) {
	clear(s)
	class, size := poolClass(cap(s))
	if class < 0 || cap(s) != size || len(p.free[class]) >= p.max {
		return
	}
	p.free[class] = append(p.free[class], s[:0])
}
