// Package weakref implements weak refs and weak ref pools to be used by the
// Golua runtime.
//
// Two interfaces: WeakRef and Pool are defined and the packages provides
// two implementations of WeakRefPool.
//
// SafeWeakRefPool is a simple implementation whose strategy is to keep all
// values alive as long as they have live WeakRefs.
//
// UnsafeWeakRefPool make every effort to let values be GCed when they are only
// reachable via WeakRefs.  It relies on casting interface{} to unsafe pointers
// and back again, which would break if Go were to have a moving GC.
package weakref

// A WeakRef is a weak reference to a value. Its Value() method returns the
// value if it is not dead (meaning that the value is still reachable),
// otherwise it returns nil.
//
// Note that it is valid for a WeakRef to keep its value alive while it is
// alive, although not very efficient.
type WeakRef interface {
	Value() interface{}
}

// A Pool maintains a set of weak references to values.  Its methods are
// not required to be thread-safe insofar as they should not get called
// concurrently.
type Pool interface {

	// Get returns a WeakRef for the given value v.  Calling Get several times
	// with the same value should return the same WeakRef.
	Get(v interface{}) WeakRef

	// Mark indicates that the pool should keep a copy of v when it dies.  This
	// allows "finalizing" v.
	Mark(v interface{})

	// ExtractDeadMarked returns all marked values which are now dead and
	// haven't been returned yet, so that some finalizing code can be run with
	// them.  The returned values are ordered in reverse order of marking (i.e.
	// if v1 was marked before v2, then v2 comes before v1 in the returned
	// list).  Further calls should not return the same values again.
	ExtractDeadMarked() []interface{}

	// ExtractAllMarked returns all marked values (live or dead), following the
	// same order as ExtractDeadMarked.  All marked values are cleared in the
	// pool so that they will no longer be returned by this method or
	// ExtractDeadMarked.
	ExtractAllMarked() []interface{}
}

// NewPool returns a new WeakRefPool with an appropriate implementation.
func NewPool() Pool {
	return NewUnsafePool()
}
