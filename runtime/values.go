package runtime

// Callable is the interface for callable values.
type Callable interface {
	Continuation(*Runtime, Cont) Cont
}

// ContWithArgs is a convenience function that returns a new
// continuation from a callable, some arguments and a next
// continuation.
func (r *Runtime) ContWithArgs(c Callable, args []Value, next Cont) Cont {
	cont := c.Continuation(r, next)
	r.Push(cont, args...)
	return cont
}

//
// Float
//

// StringNormPos returns a normalised position in the string
// i.e. -1 -> len(s)
//      -2 -> len(s) - 1
// etc
func StringNormPos(s string, p int) int {
	if p < 0 {
		p = len(s) + 1 + p
	}
	return p
}

//
// GoFunction
//

// A GoFunction is a callable value implemented by a native Go function.
type GoFunction struct {
	f           func(*Thread, *GoCont) (Cont, *Error)
	safetyFlags ComplianceFlags
	name        string
	nArgs       int
	hasEtc      bool
}

var _ Callable = (*GoFunction)(nil)

// NewGoFunction returns a new GoFunction.
func NewGoFunction(f func(*Thread, *GoCont) (Cont, *Error), name string, nArgs int, hasEtc bool) *GoFunction {
	return &GoFunction{
		f:      f,
		name:   name,
		nArgs:  nArgs,
		hasEtc: hasEtc,
	}
}

// Continuation implements Callable.Continuation.
func (f *GoFunction) Continuation(r *Runtime, next Cont) Cont {
	return NewGoCont(r, f, next)
}

func (f *GoFunction) SolemnlyDeclareCompliance(flags ComplianceFlags) {
	if flags >= complyflagsLimit {
		// User is trying to register a safety flag that is not (yet) defined.
		// This is a sign this function is not called solemnly enough!
		panic("Invalid safety flags")
	}
	f.safetyFlags |= flags
}

func SolemnlyDeclareCompliance(flags ComplianceFlags, fs ...*GoFunction) {
	for _, f := range fs {
		f.SolemnlyDeclareCompliance(flags)
	}
}

//
// LightUserData
//

// A LightUserData is some Go value of unspecified type wrapped to be used as a
// lua Value.
type LightUserData struct {
	Data interface{}
}
