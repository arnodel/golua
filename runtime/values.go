package runtime

import "github.com/arnodel/golua/ast"

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
	exp, err := ast.NumberFromString(string(s))
	if err == nil {
		switch x := exp.(type) {
		case ast.Int:
			return Int(x), IsInt
		case ast.Float:
			return Float(x).ToInt()
		}
	}
	return 0, NaN
}

type Callable interface {
	Continuation() Cont
}

type ToStringable interface {
	ToString() string
}

type Metatabler interface {
	Metatable() *Table
}

type Closure struct {
	*Code
	upvalues     []Value
	upvalueIndex int
}

func NewClosure(c *Code) *Closure {
	return &Closure{
		Code:     c,
		upvalues: make([]Value, c.UpvalueCount),
	}
}

func (c *Closure) AddUpvalue(v Value) {
	c.upvalues[c.upvalueIndex] = v
	c.upvalueIndex++
}

func (c *Closure) Continuation() Cont {
	return NewLuaCont(c)
}

type GoFunction struct {
	f      func(*Thread, *GoCont) (Cont, error)
	nArgs  int
	hasEtc bool
}

func (f *GoFunction) Continuation() Cont {
	return NewGoCont(f)
}

func ContWithArgs(c Callable, args []Value, next Cont) Cont {
	cont := c.Continuation()
	cont.Push(next)
	for _, arg := range args {
		cont.Push(arg)
	}
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
