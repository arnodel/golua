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
}

var _ Callable = (*GoFunction)(nil)

// NewGoFunction returns a new GoFunction without memory accounting.  It is
// typically called at init time before any quota context is active.  For dynamic
// creation within quota contexts, use the method (*Runtime).NewGoFunction.
func NewGoFunction(f GoFunctionFunc, name string, nArgs int, hasEtc bool) *GoFunction {
	return &GoFunction{
		f:      f,
		name:   name,
		nArgs:  nArgs,
		hasEtc: hasEtc,
	}
}

// NewGoFunction is like the package-level NewGoFunction but accounts for memory
// via RequireSize.  Use this when creating GoFunctions dynamically at runtime
// (e.g. iterators) where quota contexts may be active.
func (r *Runtime) NewGoFunction(f GoFunctionFunc, name string, nArgs int, hasEtc bool) *GoFunction {
	r.RequireSize(unsafe.Sizeof(GoFunction{}))
	return NewGoFunction(f, name, nArgs, hasEtc)
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
