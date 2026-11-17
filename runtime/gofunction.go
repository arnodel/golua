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

	// noYieldAcross is true for a function that no yield is allowed across:
	// see NoYieldAcross.
	noYieldAcross bool
}

// A GoFunctionOption says something more about a GoFunction when it is
// made.
type GoFunctionOption func(*GoFunction)

// NoYieldAcross declares that no yield is allowed across the function: if it
// calls Lua code, a function it is given or a metamethod of a value it
// handles, that code cannot yield, and gets an error if it tries ("attempt
// to yield across a Go-call boundary").  It can still resume other
// coroutines, which yield back to it.
//
// In return a coroutine can call the function without having a goroutine of
// its own (see lightthread.go).  A Go function that is not declared can have
// a yield cross it: a coroutine that calls one is given a goroutine first,
// and is slower from then on.  So declare a function that calls no Lua, or
// that does not need what it calls to be able to yield; the standard library
// is declared, but for pcall, xpcall and dofile.
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
