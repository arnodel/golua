package runtime

import (
	"os"
	"syscall"
)

// Cont is an interface for things that can be run in a Thread.  Implementations
// of Cont a typically an invocation of a Lua function or an invocation of a Go
// function.
type Cont interface {

	// Push "pushes" a Value to the continuation (arguments before a call, or
	// before resuming a continuation which has been suspended via "yield").  This is to
	// pass arguments to the continuation before calling RunInThread.
	Push(*Runtime, Value)

	// PushEtc "pushes" varargs to the continutation, this happens e.g. in Lua
	// code when running "f(...)".
	PushEtc(*Runtime, []Value)

	// RunInThread runs the continuation in the given thread, returning either
	// the next continuation to be run or an error.
	RunInThread(*Thread) (Cont, error)

	// Next() returns the continuation that follows after this one (could be
	// e.g. the caller).
	Next() Cont

	// Parent() returns the continuation that initiated this continuation.  It
	// can be the same as Next() but not necessarily.  This is useful for giving
	// meaningful tracebacks, not used by the runtime engine.
	Parent() Cont

	// DebugInfo() returns debug info for the continuation.  Used for building
	// error messages and tracebacks.
	DebugInfo() *DebugInfo
}

// Push is a convenience method that pushes a number of values to the
// continuation c.
func (r *Runtime) Push(c Cont, vals ...Value) {
	for _, v := range vals {
		c.Push(r, v)
	}
}

// Push1 is a convenience method that pushes v to the continuation c.
func (r *Runtime) Push1(c Cont, v Value) {
	c.Push(r, v)
}

// PushIoError is a convenience method that translates ioErr to a value if
// appropriated and pushes that value to c, else returns an error.  It is useful
// because a number of Go IO errors are considered regular return values by Lua.
func (r *Runtime) PushIoError(c Cont, ioErr error) error {
	// It is not specified in the Lua docs, but the test suite expects an
	// errno as the third value returned in case of an error.  I'm not sure
	// the conversion to syscall.Errno is future-proof though?

	var err error
	switch tErr := ioErr.(type) {
	case *os.PathError:
		err = tErr.Unwrap()
	case *os.LinkError:
		err = tErr.Unwrap()
	default:
		return ioErr
	}
	r.Push1(c, NilValue)
	r.Push1(c, StringValue(ioErr.Error()))
	if errno, ok := err.(syscall.Errno); ok {
		r.Push1(c, IntValue(int64(errno)))
	}
	return nil
}

// ProcessIoError is like PushIoError but its signature makes it convenient to
// use in a return statement from a GoFunc implementation.
func (r *Runtime) ProcessIoError(c Cont, ioErr error) (Cont, error) {
	if err := r.PushIoError(c, ioErr); err != nil {
		return nil, err
	}
	return c, nil
}
