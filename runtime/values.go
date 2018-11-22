package runtime

import (
	"strconv"
	"strings"
)

// Value is a runtime value.
type Value interface{}

// Callable is the interface for callable values.
type Callable interface {
	Continuation(Cont) Cont
}

// ContWithArgs is a convenience function that returns a new
// continuation from a callable, some arguments and a next
// continuation.
func ContWithArgs(c Callable, args []Value, next Cont) Cont {
	cont := c.Continuation(next)
	Push(cont, args...)
	return cont
}

//
// Bool
//

// Bool is a runtime boolean value.
type Bool bool

// Int is a runtime integral numeric value.
type Int int64

//
// Float
//

// Float is a runtime floating point numeric value.
type Float float64

// ToInt turns a Float into an Int if possible.
func (f Float) ToInt() (Int, NumberType) {
	n := Int(f)
	if Float(n) == f {
		return n, IsInt
	}
	return 0, NaI
}

//
// String
//

// String is a runtime string value.
type String string

// ToInt turns a String into and Int if possible.
func (s String) ToInt() (Int, NumberType) {
	v, tp := s.ToNumber()
	switch tp {
	case IsInt:
		return v.(Int), IsInt
	case IsFloat:
		return v.(Float).ToInt()
	}
	return 0, NaN
}

// ToNumber turns a String into a numeric value (Int or Float) if
// possible.
func (s String) ToNumber() (Value, NumberType) {
	nstring := string(s)
	if strings.ContainsAny(nstring, ".eE") {
		f, err := strconv.ParseFloat(nstring, 64)
		if err != nil {
			return nil, NaN
		}
		return Float(f), IsFloat
	}
	n, err := strconv.ParseInt(nstring, 0, 64)
	if err != nil {
		return nil, NaN
	}
	return Int(n), IsInt
}

// NormPos returns a normalised position in the string
// i.e. -1 -> len(s)
//      -2 -> len(s) - 1
// etc
func (s String) NormPos(n Int) int {
	p := int(n)
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
	f      func(*Thread, *GoCont) (Cont, *Error)
	name   string
	nArgs  int
	hasEtc bool
}

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
func (f *GoFunction) Continuation(next Cont) Cont {
	return NewGoCont(f, next)
}

//
// LightUserData
//

// A LightUserData is some Go value of unspecified type wrapped to be used as a
// lua Value.
type LightUserData struct {
	Data interface{}
}
