// Package weakref implements weak refs and weak ref pools to be used by the
// Golua runtime.
//
// Two interfaces WeakRef and Pool are defined and the packages provides three
// implementations of WeakRefPool.  The Golua runtime has a Pool instance that
// it uses to help with finalizing of Lua values and making sure finalizers do
// not run after the runtime has finished.
//
// SafeWeakRefPool is a simple implementation whose strategy is to keep all
// values alive as long as they have live WeakRefs.
//
// UnsafeWeakRefPool makes every effort to let values be GCed when they are only
// reachable via WeakRefs.  It relies on casting interface{} to unsafe pointers
// and back again, which would break if Go were to have a moving GC.
//
// ClonePool also lets values be GCed when they are unreachable outside of the
// pool and does so on any compliant Go implementation.  However it does not
// support WeakRefs (i.e. Get(v) always returns nil).
package weakref

// Value is the interface that must be implemented by values managed by a Pool.
type Value interface {

	// Key returns a Key instance that is unique to a Value and its clones (i.e.
	// v.Key() == v.Clone().Key(), but two independent values should have
	// distinct keys)
	Key() Key

	// Clone() returns a copy of the value that behaves like the original value
	// (in particular it shares its state).
	Clone() Value
}

// Key is the type for value keys (see the Value interface).
type Key interface{}

// A WeakRef is a weak reference to a value. Its Value() method returns the
// value if it is not dead (meaning that the value is still reachable),
// otherwise it returns nil.
//
// Note that it is valid for a WeakRef to keep its value alive while it is
// alive, although not very efficient.
type WeakRef interface {
	Value() Value
}

// A Pool maintains a set of weak references to values.  Its methods are not
// required to be thread-safe insofar as they should not get called
// concurrently.
//
// Each Golua Runtime has a Pool instance to help it manage weak references and
// finalizers.
type Pool interface {

	// Get returns a WeakRef for the given value v.  Calling Get several times
	// with the same value should return the same WeakRef.
	//
	// An implementation of Pool may return nil if it is unable to make a
	// WeakRef for the passed-in value.
	Get(v Value) WeakRef

	// Mark indicates that the pool should keep a copy of v when it becomes
	// unreachable.  It can then notify the Golua runtime via the
	// ExtractPendingFinalize() and ExtractPendingRelease() methods (depending
	// on the flags passed in).
	//
	// The associated Golua Runtime marks all values which have a __gc
	// metamethod with the Finalize flag, and all values which implement the
	// ResourceReleaser interface with the Release flag (this is contextual
	// info, this package is unaware of this).
	//
	// Note: "Mark" is the terminology used in the Lua docs, it is unrelated to
	// "Mark and Sweep".
	Mark(v Value, flags MarkFlags)

	// ExtractPendingFinalize returns all marked values which are no longer reachable
	// and haven't been returned yet, so that some finalizing code can be run
	// with them.  The returned values are ordered in reverse order of marking
	// (i.e. if v1 was marked before v2, then v2 comes before v1 in the returned
	// list). Further calls should not return the same values again.
	//
	// This is called periodically by the Golua Runtime to run Lua finalizers on
	// GCed values.
	ExtractPendingFinalize() []Value

	// ExtractPendingRelease returns all marked values which are no longer
	// reachable, no longer need to be finalized and haven't been returned yet,
	// so that their associated resources can be released.  The returned values
	// are ordered in reverse order of marking (i.e. if v1 was marked before v2,
	// then v2 comes before v1 in the returned list). Further calls should not
	// return the same values again.
	//
	// This is called periodically by the Golua Runtime to release resources
	// associated with GCed values.
	ExtractPendingRelease() []Value

	// ExtractAllMarkedFinalize returns all values marked for Lua finalizing,
	// following the same order as ExtractPendingFinalize.  All marked values
	// are cleared in the pool so that they will no longer be returned by this
	// method or ExtractPendingFinalize.
	//
	// Typically this method will be called when the associated Golua Runtime is
	// being closed so that all outstanding Lua finalizers can be called (even
	// if their values might get GCed later).
	ExtractAllMarkedFinalize() []Value

	// ExtractAllMarkedRelease returns all values marked for releasing,
	// following the same order as ExtractPendingRelease.  All values marked for
	// Finalize or Release are cleared in the pool so that they will no longer
	// be returned by any Extract* method.  This means this method should be
	// called as the last action before discarding the pool.
	//
	// Typically this method will be called when the associated Golua Runtime is
	// being closed so that all outstanding Lua finalizers can be called (even
	// if their values might get GCed later).
	ExtractAllMarkedRelease() []Value
}

// MarkFlags are passsed to the Pool.Mark method to signal to the pool how to
// deal with the value when it becomes unreachable.
type MarkFlags uint8

const (
	Finalize MarkFlags = 1 << iota // Mark a value for finalizing
	Release                        // Mark a value for releasing
)
