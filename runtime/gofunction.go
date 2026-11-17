package runtime

import "unsafe"

type GoFunctionFunc func(*Thread, *GoCont) (Cont, error)

// A GoFunction is a callable value implemented by a native Go function.
type GoFunction struct {
	f           GoFunctionFunc
	safetyFlags ComplianceFlags
	name        string
	nArgs       int
	hasEtc      bool

	// noYieldAcross is true for a function that a yield never crosses:
	// see NoYieldAcross.
	noYieldAcross bool
}

// A GoFunctionOption says something more about a GoFunction when it is
// made.
type GoFunctionOption func(*GoFunction)

// NoYieldAcross declares that a yield never crosses the function: no Lua
// code that could yield runs while the function is under way.  A function
// cannot know what the Lua code it is given will do, so in practice it is
// one that calls no Lua at all: neither a function it is given, nor a
// metamethod of a value it handles.
//
// A coroutine that calls only such Go functions needs no goroutine (see
// notes/coroutines.md); one that calls any other Go function is given a
// goroutine first, in case the function calls Lua code that yields.  A
// function that calls Lua in spite of the declaration gets an error from the
// call when a coroutine that has no goroutine calls it.
func NoYieldAcross(f *GoFunction) {
	f.noYieldAcross = true
}

var _ Callable = (*GoFunction)(nil)

// NewGoFunction returns a new GoFunction without memory accounting.  It is
// typically called at init time before any quota context is active.  For dynamic
// creation within quota contexts, use the method (*Runtime).NewGoFunction.
func NewGoFunction(f GoFunctionFunc, name string, nArgs int, hasEtc bool, opts ...GoFunctionOption) *GoFunction {
	gof := &GoFunction{
		f:      f,
		name:   name,
		nArgs:  nArgs,
		hasEtc: hasEtc,
	}
	for _, opt := range opts {
		opt(gof)
	}
	return gof
}

// NewGoFunction is like the package-level NewGoFunction but accounts for memory
// via RequireSize.  Use this when creating GoFunctions dynamically at runtime
// (e.g. iterators) where quota contexts may be active.
func (r *Runtime) NewGoFunction(f GoFunctionFunc, name string, nArgs int, hasEtc bool, opts ...GoFunctionOption) *GoFunction {
	r.RequireSize(unsafe.Sizeof(GoFunction{}))
	return NewGoFunction(f, name, nArgs, hasEtc, opts...)
}

// Continuation implements Callable.Continuation.
func (f *GoFunction) Continuation(t *Thread, next Cont) Cont {
	return NewGoCont(t, f, next)
}

// SolemnlyDeclareCompliance adds compliance flags to f.  See quotas.md for
// details about compliance flags.
func (f *GoFunction) SolemnlyDeclareCompliance(flags ComplianceFlags) {
	if flags >= complyflagsLimit {
		// User is trying to register a safety flag that is not (yet) defined.
		// This is a sign this function is not called solemnly enough!
		panic("Invalid safety flags")
	}
	f.safetyFlags |= flags
}

// SolemnlyDeclareCompliance is a convenience function that adds the same set of
// compliance flags to a number of functions.  See quotas.md for details about
// compliance flags.
func SolemnlyDeclareCompliance(flags ComplianceFlags, fs ...*GoFunction) {
	for _, f := range fs {
		f.SolemnlyDeclareCompliance(flags)
	}
}
