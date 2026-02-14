//go:build go1.24

package runtime

import (
	"runtime"
	"sort"
	"sync"
	"weak"

	"github.com/arnodel/golua/runtime/internal/luagc"
)

func init() {
	luagc.RegisterPool("weak", func() luagc.Pool { return NewWeakPool() })
}

// WeakPool is an implementation of luagc.Pool that uses Go 1.24's weak.Pointer
// for proper weak references and runtime.AddCleanup for GC notifications.
// Unlike UnsafePool, this implementation uses official Go APIs and will not
// break if Go's GC implementation changes.
//
// This pool lives in the runtime package (rather than luagc) because it needs
// to type-switch on concrete types (*Table, *UserData) to create weak pointers
// and attach cleanups.
type WeakPool struct {
	mx              sync.Mutex
	entries         map[luagc.Key]*weakEntry
	pendingFinalize sortableWeakEntries
	pendingRelease  sortableWeakEntries
	lastMarkOrder   int
}

var _ luagc.Pool = &WeakPool{}

// NewWeakPool returns a new *WeakPool ready to be used.
func NewWeakPool() *WeakPool {
	return &WeakPool{entries: make(map[luagc.Key]*weakEntry)}
}

// weakEntry holds tracking state for a value.
type weakEntry struct {
	wp      any             // weak.Pointer[Table] or weak.Pointer[UserData]
	cleanup runtime.Cleanup // handle to cancel the cleanup
	clone   luagc.Value     // Clone of the value for finalization
	key     luagc.Key
	pool    *WeakPool

	markOrder int
	flags     weakEntryFlags
}

type weakEntryFlags uint8

const (
	weFinalized weakEntryFlags = 1 << iota // The Lua finalizer no longer needs to run
	weReleased                             // The value's resources no longer need to be released
)

func (e *weakEntry) hasFlag(flag weakEntryFlags) bool {
	return e.flags&flag != 0
}

func (e *weakEntry) setFlag(flag weakEntryFlags) {
	e.flags |= flag
}

func (e *weakEntry) clearFlag(flag weakEntryFlags) {
	e.flags &= ^flag
}

// liveValue returns the referenced value if it's still alive via weak pointer.
func (e *weakEntry) liveValue() luagc.Value {
	switch wp := e.wp.(type) {
	case weak.Pointer[Table]:
		if t := wp.Value(); t != nil {
			return t
		}
	case weak.Pointer[UserData]:
		if u := wp.Value(); u != nil {
			return u
		}
	}
	return nil
}

// Get returns a WeakRef for v if possible.
func (p *WeakPool) Get(v luagc.Value) luagc.WeakRef {
	p.mx.Lock()
	defer p.mx.Unlock()
	entry := p.getOrCreateEntry(v)
	return &weakPoolRef{entry: entry}
}

// Mark marks v for finalizing and/or releasing.
func (p *WeakPool) Mark(v luagc.Value, flags luagc.MarkFlags) {
	if flags == 0 {
		return
	}
	p.mx.Lock()
	defer p.mx.Unlock()
	p.lastMarkOrder++
	entry := p.getOrCreateEntry(v)
	entry.clone = v.Clone() // Update clone to capture latest state (e.g., metatable)
	entry.markOrder = p.lastMarkOrder
	if flags&luagc.Finalize == 0 {
		entry.setFlag(weFinalized)
	} else {
		entry.clearFlag(weFinalized)
	}
	if flags&luagc.Release == 0 {
		entry.setFlag(weReleased)
	} else {
		entry.clearFlag(weReleased)
	}
}

func (p *WeakPool) getOrCreateEntry(v luagc.Value) *weakEntry {
	key := v.Key()
	entry := p.entries[key]
	if entry == nil {
		entry = &weakEntry{
			key:  key,
			pool: p,
		}

		// Create weak pointer and attach cleanup based on concrete type.
		switch x := v.(type) {
		case *Table:
			entry.wp = weak.Make(x)
			entry.cleanup = runtime.AddCleanup(x, weakEntryCleanup, entry)
		case *UserData:
			entry.wp = weak.Make(x)
			entry.cleanup = runtime.AddCleanup(x, weakEntryCleanup, entry)
		}

		p.entries[key] = entry
	}
	return entry
}

// weakEntryCleanup is called by Go's GC when a tracked value becomes
// unreachable. It adds the value's clone to the appropriate pending lists.
func weakEntryCleanup(entry *weakEntry) {
	p := entry.pool
	p.mx.Lock()
	defer p.mx.Unlock()

	ev := entryVal{v: entry.clone, e: entry}
	if !entry.hasFlag(weFinalized) {
		p.pendingFinalize = append(p.pendingFinalize, ev)
	}
	if !entry.hasFlag(weReleased) {
		p.pendingRelease = append(p.pendingRelease, ev)
	}
	delete(p.entries, entry.key)
}

// ExtractPendingFinalize returns values needing Lua finalization.
func (p *WeakPool) ExtractPendingFinalize() []luagc.Value {
	p.mx.Lock()
	pending := p.pendingFinalize
	if pending == nil {
		p.mx.Unlock()
		return nil
	}
	p.pendingFinalize = nil
	for _, ev := range pending {
		ev.e.setFlag(weFinalized)
	}
	p.mx.Unlock()

	sort.Sort(pending)
	return pending.vals()
}

// ExtractPendingRelease returns values needing resource release.
// Only returns values whose finalization is already done (weFinalized is set).
func (p *WeakPool) ExtractPendingRelease() []luagc.Value {
	p.mx.Lock()
	if p.pendingRelease == nil {
		p.mx.Unlock()
		return nil
	}
	var ready, remaining sortableWeakEntries
	for _, ev := range p.pendingRelease {
		if ev.e.hasFlag(weFinalized) {
			ev.e.setFlag(weReleased)
			ready = append(ready, ev)
		} else {
			remaining = append(remaining, ev)
		}
	}
	p.pendingRelease = remaining
	p.mx.Unlock()

	if ready == nil {
		return nil
	}
	sort.Sort(ready)
	return ready.vals()
}

// ExtractAllMarkedFinalize returns all values marked for finalizing.
func (p *WeakPool) ExtractAllMarkedFinalize() []luagc.Value {
	p.mx.Lock()
	p.pendingFinalize = nil
	var marked sortableWeakEntries
	for _, entry := range p.entries {
		if !entry.hasFlag(weFinalized) {
			marked = append(marked, entryVal{
				v: entry.clone,
				e: entry,
			})
			entry.setFlag(weFinalized)
			entry.cleanup.Stop()
		}
	}
	p.mx.Unlock()

	sort.Sort(marked)
	return marked.vals()
}

// ExtractAllMarkedRelease returns all values marked for releasing.
func (p *WeakPool) ExtractAllMarkedRelease() []luagc.Value {
	p.mx.Lock()
	marked := p.pendingRelease
	for _, entry := range p.entries {
		if !entry.hasFlag(weReleased) {
			marked = append(marked, entryVal{
				v: entry.clone,
				e: entry,
			})
			entry.setFlag(weReleased)
			entry.cleanup.Stop()
		}
	}
	p.pendingRelease = nil
	p.entries = nil
	p.mx.Unlock()

	sort.Sort(marked)
	return marked.vals()
}

//
// WeakRef implementation for WeakPool
//

type weakPoolRef struct {
	entry *weakEntry
}

var _ luagc.WeakRef = &weakPoolRef{}

// Value returns the value if still alive, otherwise nil.
func (r *weakPoolRef) Value() luagc.Value {
	return r.entry.liveValue()
}

//
// Sorting helpers
//

type entryVal struct {
	v luagc.Value
	e *weakEntry
}

type sortableWeakEntries []entryVal

var _ sort.Interface = sortableWeakEntries(nil)

func (vs sortableWeakEntries) Len() int {
	return len(vs)
}

func (vs sortableWeakEntries) Less(i, j int) bool {
	return vs[i].e.markOrder > vs[j].e.markOrder
}

func (vs sortableWeakEntries) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}

func (vs sortableWeakEntries) vals() []luagc.Value {
	vals := make([]luagc.Value, len(vs))
	for i, v := range vs {
		vals[i] = v.v
	}
	return vals
}
