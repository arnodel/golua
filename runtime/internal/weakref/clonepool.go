package weakref

import (
	"sort"
	"sync"
)

//
// Clone-based Pool implementation
//

// ClonePool is an implementation of Pool that makes every effort to let values
// be GCed when they are only reachable via WeakRefs.  Unlike UnsafePool, it
// doesn't rely on any undocumented properties of the Go runtime so it is safe
// to use in any compliant Go implementation.  The downsid of this
// implementation is that it cannot provide any WeakRefs.
type ClonePool struct {
	pendingClones map[Key]pendingClone
	lastMarkOrder int
	mx            sync.Mutex

	pendingFinalize sortablePendingClones
	pendingRelease  sortablePendingClones
}

// NewClonePool returns a new *ClonePool ready to be used.
func NewClonePool() *ClonePool {
	return &ClonePool{
		pendingClones: make(map[Key]pendingClone),
	}
}

var _ Pool = (*ClonePool)(nil)

// Get always returns nil because ClonePool doesn't support weak references.
func (p *ClonePool) Get(v Value) WeakRef {
	return nil
}

// Mark marks v for finalizing, i.e. when v is garbage collected, its finalizer
// should be run.
func (p *ClonePool) Mark(v Value, flags MarkFlags) {
	k := v.Key()
	p.mx.Lock()
	defer p.mx.Unlock()
	c, ok := p.pendingClones[k]
	if flags == 0 {
		if ok {
			setFinalizer(v, nil)
			delete(p.pendingClones, k)
		}
		return
	}
	if !ok {
		setFinalizer(v, p.goFinalizer)
	}
	c.value = v.Clone()
	p.lastMarkOrder++
	c.markOrder = p.lastMarkOrder
	if flags&Finalize == 0 {
		c.setFlag(wrFinalized)
	} else {
		c.clearFlag(wrFinalized)
	}
	if flags&Release == 0 {
		c.setFlag(wrReleased)
	} else {
		c.clearFlag(wrReleased)
	}
	p.pendingClones[k] = c
}

// ExtractPendingFinalize returns the set of values which are being garbage
// collected and need their finalizer running, in the order that they should be
// run.  The caller of this function has the responsibility to run all the
// finalizers. The values returned are removed from the pool and their weak refs
// are invalidated.
func (p *ClonePool) ExtractPendingFinalize() []Value {
	p.mx.Lock()
	pending := p.pendingFinalize
	if pending == nil {
		p.mx.Unlock()
		return nil
	}
	p.pendingFinalize = nil
	p.mx.Unlock()

	for _, c := range pending {
		// The finalizer code might resurrect the value, or there may still be
		// release code to run, so we need the clone to have a finalizer.
		setFinalizer(c.value, p.goFinalizer)
	}

	// Lua wants to run finalizers in reverse order
	sort.Sort(pending)
	return pending.vals()
}

func (p *ClonePool) ExtractPendingRelease() []Value {
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
func (p *ClonePool) ExtractAllMarkedFinalize() []Value {
	p.mx.Lock()

	// Disregard the pendingFinalize list as all values are still present in the
	// weakrefs map.
	p.pendingFinalize = nil
	var marked sortablePendingClones
	for k, c := range p.pendingClones {
		if !c.hasFlag(wrFinalized) {
			c.setFlag(wrFinalized)
			p.pendingClones[k] = c
			marked = append(marked, c)
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
func (p *ClonePool) ExtractAllMarkedRelease() []Value {
	p.mx.Lock()

	// Start from values whose go finalizer has already run and are awaiting
	// release, then add all values in the weakrefs map not yet released.
	marked := p.pendingRelease
	for _, c := range p.pendingClones {
		if !c.hasFlag(wrReleased) {
			marked = append(marked, c)
		}
	}
	p.pendingRelease = nil
	p.pendingClones = nil
	p.mx.Unlock()

	// Sort in reverse order
	sort.Sort(marked)
	return marked.vals()
}

func (p *ClonePool) goFinalizer(v Value) {
	k := v.Key()
	p.mx.Lock()
	defer p.mx.Unlock()
	c, ok := p.pendingClones[k]
	if !ok {
		// We are too late - ExtractAllMarkedRelease() has been run
		return
	}

	if !c.hasFlag(wrFinalized) {
		p.pendingFinalize = append(p.pendingFinalize, c)
		c.setFlag(wrFinalized)
		p.pendingClones[k] = c
		return
	}

	if !c.hasFlag(wrReleased) {
		p.pendingRelease = append(p.pendingRelease, c)
	}

	delete(p.pendingClones, v.Key())
}

type pendingClone struct {
	value     Value
	markOrder int
	flags     wrStatusFlags
}

func (c *pendingClone) hasFlag(flag wrStatusFlags) bool {
	return c.flags&flag != 0
}

func (c *pendingClone) setFlag(flag wrStatusFlags) {
	c.flags |= flag
}

func (c *pendingClone) clearFlag(flag wrStatusFlags) {
	c.flags &= ^flag
}

type sortablePendingClones []pendingClone

var _ sort.Interface = sortableVals(nil)

func (vs sortablePendingClones) Len() int {
	return len(vs)
}

func (vs sortablePendingClones) Less(i, j int) bool {
	return vs[i].markOrder > vs[j].markOrder
}

func (vs sortablePendingClones) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}

// Extract the values.
func (vs sortablePendingClones) vals() []Value {
	vals := make([]Value, len(vs))
	for i, v := range vs {
		vals[i] = v.value
	}
	return vals
}
