package weakref

import (
	"runtime"
	"sort"
	"sync"
	"unsafe"
)

//
// Unsafe Pool implementation
//

// UnsafePool is an implementation of Pool that makes every effort to let
// values be GCed when they are only reachable via WeakRefs.  It relies on
// casting interface{} to unsafe pointers and back again, which would break if
// Go were to have a moving GC.
type UnsafePool struct {
	mx              sync.Mutex           // Used to synchronize access to weakrefs, pendingVals, pendingOrders.
	weakrefs        map[uintptr]*weakRef //
	pendingFinalize sortableVals         // Values pending Lua finalization
	pendingRelease  sortableVals
	lastMarkOrder   int // this is to sort values by reverse order of mark for finalize
}

var _ Pool = &UnsafePool{}

// NewUnsafePool returns a new *UnsafeWeakRefPool ready to be used.
func NewUnsafePool() *UnsafePool {
	return &UnsafePool{weakrefs: make(map[uintptr]*weakRef)}
}

// Get returns a *WeakRef for v if possible.
func (p *UnsafePool) Get(iface interface{}) WeakRef {
	p.mx.Lock()
	defer p.mx.Unlock()
	return p.get(iface)
}

// Returns a *WeakRef for iface, not thread safe, only call when you have the
// pool lock.
func (p *UnsafePool) get(iface interface{}) *weakRef {
	w := getwiface(iface)
	id := w.id()
	r := p.weakrefs[id]
	if r == nil {
		runtime.SetFinalizer(iface, p.goFinalizer)
		r = &weakRef{
			w:    getwiface(iface),
			pool: p,
		}
		p.weakrefs[id] = r
	}
	return r
}

// Mark marks v for finalizing, i.e. when v is garbage collected, its finalizer
// should be run.  It only takes effect if v can have a weak ref.
func (p *UnsafePool) Mark(iface interface{}, flags MarkFlags) {
	if flags == 0 {
		return
	}
	p.mx.Lock()
	defer p.mx.Unlock()
	p.lastMarkOrder++
	r := p.get(iface)
	r.markOrder = p.lastMarkOrder
	if flags&Finalize == 0 {
		r.setFlag(wrFinalized)
	}
	if flags&Release == 0 {
		r.setFlag(wrReleased)
	}
}

// ExtractDeadMarked returns the set of values which are being garbage collected
// and need their finalizer running, in the order that they should be run.  The
// caller of this function has the responsibility to run all the finalizers. The
// values returned are removed from the pool and their weak refs are
// invalidated.
func (p *UnsafePool) ExtractPendingFinalize() []interface{} {

	// It may be that since a value pending finalizing has been added to the
	// list, it was resurrected by a weak ref, so we need to go through the list
	// and filter these out first.
	p.mx.Lock()
	if p.pendingFinalize == nil {
		p.mx.Unlock()
		return nil
	}
	var pending sortableVals
	for _, rval := range p.pendingFinalize {
		if rval.ref.hasFlag(wrResurrected) {
			rval.ref.clearFlag(wrResurrected)
		} else {
			rval.ref.setFlag(wrFinalized)
			pending = append(pending, rval)
		}
	}
	p.pendingFinalize = nil
	p.mx.Unlock()

	// Lua wants to run finalizers in reverse order
	sort.Sort(pending)
	return pending.vals()
}

func (p *UnsafePool) ExtractPendingRelease() []interface{} {
	p.mx.Lock()
	pending := p.pendingRelease
	if pending == nil {
		p.mx.Unlock()
		return nil
	}
	p.pendingRelease = nil
	p.mx.Unlock()

	sort.Sort(pending)
	return pending.vals()
}

// ExtractAllMarkedFinalized returns all the values that have been marked for
// finalizing, even if their go finalizer hasn't run yet.  This is useful e.g.
// when closing a runtime, to run all pending finalizers.
func (p *UnsafePool) ExtractAllMarkedFinalize() []interface{} {
	p.mx.Lock()

	// Disregard the pendingFinalize list as all values are still present in the
	// weakrefs map.
	p.pendingFinalize = nil
	var marked sortableVals
	for _, r := range p.weakrefs {
		if r.flags&wrFinalized == 0 {
			iface := r.w.iface()
			marked = append(marked, refVal{
				val: iface,
				ref: r,
			})
			r.setFlag(wrFinalized)
			// We don't want the finalizer to be triggered anymore, but more
			// important the finalizer is holding a reference to the pool
			// (although that may not affect its reachability?)
			runtime.SetFinalizer(iface, nil)
		}
	}
	p.mx.Unlock()

	// Sort in reverse order
	sort.Sort(marked)
	return marked.vals()
}

// ExtractAllMarkedRelease returns all the values that have been marked for
// release, even if their go finalizer hasn't run yet.  This is useful e.g. when
// closing a runtime, to release all resources.
func (p *UnsafePool) ExtractAllMarkedRelease() []interface{} {
	p.mx.Lock()

	// Start from values whose go finalizer has already run and are awaiting
	// release, then add all values in the weakrefs map not yet released.
	marked := p.pendingRelease
	for _, r := range p.weakrefs {
		if r.flags&wrReleased == 0 {
			iface := r.w.iface()
			marked = append(marked, refVal{
				val: iface,
				ref: r,
			})
			r.flags |= wrReleased
			// We don't want the finalizer to be triggered anymore, but more
			// important the finalizer is holding a reference to the pool
			// (although that may not affect its reachability?)
			runtime.SetFinalizer(iface, nil)
		}
	}
	p.pendingRelease = nil
	p.mx.Unlock()

	// Sort in reverse order
	sort.Sort(marked)
	return marked.vals()
}

// This is the finalizer that Go runs on values added to the pool when they
// become unreachable.
func (p *UnsafePool) goFinalizer(iface interface{}) {
	p.mx.Lock()
	defer p.mx.Unlock()
	id := getwiface(iface).id()
	r := p.weakrefs[id]
	if r == nil {
		return
	}

	// A resurrected value has its go finalizer reinstated.
	if r.flags&wrResurrected != 0 {
		r.flags &= ^wrResurrected
		runtime.SetFinalizer(iface, p.goFinalizer)
		return
	}

	rval := refVal{
		val: iface,
		ref: r,
	}

	// A not yet finalized value is added to the pendingFinalize list.  As it
	// may get resurrected in the finalizer, we reinstate its go finalizer.
	// When it is extracted to be processed, its finalized flag will be set.
	if r.flags&wrFinalized == 0 {
		p.pendingFinalize = append(p.pendingFinalize, rval)
		runtime.SetFinalizer(iface, p.goFinalizer)
		return
	}

	// A not yet released value is added to the pendingRelease list.  This is a
	// point of no return, this value is now dead to the Lua runtime.
	if r.flags&wrReleased == 0 {
		r.flags = wrDead
		p.pendingRelease = append(p.pendingRelease, rval)
	}

	// It is now safe to remove this value from the weakref pool.
	delete(p.weakrefs, id)
}

//
// WeakRef implementation for UnsafePool
//

type weakRef struct {
	w         wiface // encodes the value the weak ref refers to
	markOrder int    // positive if the value was marked with WeakRefPool.Mark()
	flags     wrStatusFlags

	// Needed to sync with the Go finalizers which run in their own goroutine.
	pool *UnsafePool
}

var _ WeakRef = &weakRef{}

// Value returns the value this weak ref refers to if it is still alive, else
// returns NilValue.
func (r *weakRef) Value() interface{} {
	r.pool.mx.Lock()
	defer r.pool.mx.Unlock()
	if r.hasFlag(wrDead) {
		return nil
	}
	r.setFlag(wrResurrected)
	return r.w.iface()
}

func (r *weakRef) hasFlag(flag wrStatusFlags) bool {
	return r.flags&flag != 0
}

func (r *weakRef) setFlag(flag wrStatusFlags) {
	r.flags |= flag
}

func (r *weakRef) clearFlag(flag wrStatusFlags) {
	r.flags &= ^flag
}

//
// Statuses of a weak ref
//

type wrStatusFlags uint8

// A weakRef has 4 status flags: "dead" , "resurrected", "finalized",
// "released".
const (
	wrDead        wrStatusFlags = 1 << iota // The value is dead to the Lua runtime
	wrResurrected                           // The weakRef's value has been obtained
	wrFinalized                             // The Lua finalizer no longer needs to run
	wrReleased                              // The value's resources no longer need to be released (in this case it should be dead)
)

//
// Non-retaining reference to an interface value
//

// wiface is an unsafe copy of an interface.  It remembers the type and data of
// a Go interface value, but does not keep it alive.
type wiface [2]uintptr

func getwiface(iface interface{}) wiface {
	return *(*[2]uintptr)(unsafe.Pointer(&iface))
}

func (w wiface) id() uintptr {
	// This is the address containing the interface data.
	return w[1]
}

func (w wiface) iface() interface{} {
	return *(*interface{})(unsafe.Pointer(&w))
}

//
// Values need to be sorted by reverse mark order.  The data structures below help with that.
//
type refVal struct {
	val interface{}
	ref *weakRef
}

type sortableVals []refVal

var _ sort.Interface = sortableVals(nil)

func (vs sortableVals) Len() int {
	return len(vs)
}

func (vs sortableVals) Less(i, j int) bool {
	return vs[i].ref.markOrder > vs[j].ref.markOrder
}

func (vs sortableVals) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}

// Extract the values.
func (vs sortableVals) vals() []interface{} {
	vals := make([]interface{}, len(vs))
	for i, v := range vs {
		vals[i] = v.val
	}
	return vals
}
