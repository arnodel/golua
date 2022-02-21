package weakref

//
// Safe Pool implementation
//

// SafePool is an implementation of Pool that keeps alive all the values it is
// given.
type SafePool struct {
	markedFinalize []interface{}
	markedRelease  []interface{}
}

func NewSafePool() *SafePool {
	return &SafePool{}
}

var _ Pool = &SafePool{}

// Get returns a WeakRef with the given value.  That WeakRef will keep the value
// alive!
func (p *SafePool) Get(iface interface{}) WeakRef {
	return safeWeakRef{iface: iface}
}

// Mark adds iface to the list of marked values.
func (p *SafePool) Mark(iface interface{}, flags MarkFlags) {
	if flags&Finalize != 0 {
		p.markedFinalize = appendNoDup(p.markedFinalize, iface)
	}
	if flags&Release != 0 {
		p.markedRelease = appendNoDup(p.markedRelease, iface)
	}
}

// append y to the end of xs, removing a previous occurrence of y in xs if
// found.  It is assumed that xs doesn't have any duplicates.
func appendNoDup(xs []interface{}, y interface{}) []interface{} {
	for i, x := range xs {
		if x == y {
			copy(xs[i:], xs[i+1:])
			xs[len(xs)-1] = y
			return xs
		}
	}
	return append(xs, y)
}

// ExtractPendingFinalize returns nil because all marked values are kept alive by the
// pool.
func (p *SafePool) ExtractPendingFinalize() []interface{} {
	return nil
}

// ExtractPendingRelease returns nil because all marked values are kept alive by
// the pool.
func (p *SafePool) ExtractPendingRelease() []interface{} {
	return nil
}

// ExtractAllMarkedFinalize returns all values marked for finalizing in reverse
// order, clearing them.
func (p *SafePool) ExtractAllMarkedFinalize() []interface{} {
	marked := p.markedFinalize
	p.markedFinalize = nil
	reverse(marked)
	return marked
}

// ExtractAllMarkedRelease returns all values marked for release in reverse
// order, clearing them.
func (p *SafePool) ExtractAllMarkedRelease() []interface{} {
	marked := p.markedRelease
	p.markedRelease = nil
	reverse(marked)
	return marked
}

//
// WeakRef implementation for SafePool
//

// safeWeakRef keeps the value it reference alive.
type safeWeakRef struct {
	iface interface{}
}

var _ WeakRef = safeWeakRef{}

// Value returns the value r is referring to.
func (r safeWeakRef) Value() interface{} {
	return r.iface
}

//
// Helper functions
//

func reverse(s []interface{}) {
	for i, j := 0, len(s)-1; i < j; {
		s[i], s[j] = s[j], s[i]
		i++
		j--
	}
}
