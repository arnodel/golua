// Package luagc defines the types shared between the Golua runtime and its
// weak ref pool, WeakPool in the runtime package: the Value interface that
// pooled values implement, the WeakRef interface, and the MarkFlags passed to
// WeakPool.Mark.  The runtime uses the pool to help with finalizing Lua values
// and making sure finalizers do not run after the runtime has finished.
package luagc

// Value is the interface that must be implemented by values managed by the
// pool.
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

// MarkFlags are passed to the pool's Mark method to signal to the pool how to
// deal with the value when it becomes unreachable.
type MarkFlags uint8

const (
	Finalize MarkFlags = 1 << iota // Mark a value for finalizing
	Release                        // Mark a value for releasing
)
