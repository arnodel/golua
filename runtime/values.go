package runtime

import (
	"strconv"
	"strings"
)

// Value is a runtime value.
type Value interface{}

type Bool bool
type Int int64
type Float float64
type NilType struct{}
type String string

func (f Float) ToInt() (Int, NumberType) {
	n := Int(f)
	if Float(n) == f {
		return n, IsInt
	}
	return 0, NaI
}

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

type Callable interface {
	Continuation(Cont) Cont
}

type ToStringable interface {
	ToString() string
}

type Metatabler interface {
	Metatable() *Table
}

type GoFunction struct {
	f      func(*Thread, *GoCont) (Cont, *Error)
	nArgs  int
	hasEtc bool
}

func NewGoFunction(f func(*Thread, *GoCont) (Cont, *Error), nArgs int, hasEtc bool) *GoFunction {
	return &GoFunction{
		f:      f,
		nArgs:  nArgs,
		hasEtc: hasEtc,
	}
}

func (f *GoFunction) Continuation(next Cont) Cont {
	return NewGoCont(f, next)
}

func ContWithArgs(c Callable, args []Value, next Cont) Cont {
	cont := c.Continuation(next)
	Push(cont, args...)
	return cont
}

type ValueError struct {
	value Value
}

func (err ValueError) Error() string {
	s, _ := AsString(err.value)
	return string(s)
}

func ValueFromError(err error) Value {
	if v, ok := err.(ValueError); ok {
		return v.value
	}
	return String(err.Error())
}

func ErrorFromValue(v Value) error {
	return ValueError{value: v}
}
