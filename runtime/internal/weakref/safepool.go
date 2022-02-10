package weakref

//
// Safe Pool implementation
//

// SafePool is an implementation of Pool that keeps alive all the values it is
// given.
type SafePool struct {
	marked []interface{}
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
func (p *SafePool) Mark(iface interface{}) {
	p.marked = append(p.marked, iface)
}

// ExtractDeadMarked returns nil because all marked values are kept alive by the
// pool.
func (p *SafePool) ExtractDeadMarked() []interface{} {
	return nil
}

// ExtractAllMarked returns all marked values in reverse order, clearing them.
func (p *SafePool) ExtractAllMarked() []interface{} {
	marked := p.marked
	p.marked = nil
	reverse(marked)
	return runPrefinalizers(marked)
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
