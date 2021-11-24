package runtime

import (
	"strconv"
	"strings"
)

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

// FloatToInt turns a float64 into an int64 if possible.
func FloatToInt(f float64) (int64, NumberType) {
	n := int64(f)
	if float64(n) == f {
		return n, IsInt
	}
	return 0, NaI
}

//
// String
//

// stringToInt turns a string into and int64 if possible.
func stringToInt(s string) (int64, NumberType) {
	n, f, tp := stringToNumber(s)
	switch tp {
	case IsInt:
		return n, IsInt
	case IsFloat:
		return FloatToInt(f)
	}
	return 0, NaN
}

func stringToNumber(s string) (n int64, f float64, tp NumberType) {
	var err error
	if strings.ContainsAny(s, ".eE") {
		f, err = strconv.ParseFloat(s, 64)
		if err != nil {
			tp = NaN
			return
		}
		tp = IsFloat
		return
	}
	n, err = strconv.ParseInt(s, 0, 64)
	if err != nil {
		tp = NaN
		return
	}
	tp = IsInt
	return
}

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
	safetyFlags RuntimeSafetyFlags
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

func (f *GoFunction) SolemnlyDeclareSafetyFlags(flags RuntimeSafetyFlags) {
	if flags >= rcs_limit {
		// User is trying to register a safety flag that is not (yet) defined.
		// This is a sign this function is not called solemnly enough!
		panic("Invalid safety flags")
	}
	f.safetyFlags |= flags
}

func SolemnlyDeclareSafetyFlags(flags RuntimeSafetyFlags, fs ...*GoFunction) {
	for _, f := range fs {
		f.SolemnlyDeclareSafetyFlags(flags)
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
