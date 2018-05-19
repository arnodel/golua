package runtime

import "github.com/arnodel/golua/ast"

type NumberType uint16

const (
	IsFloat NumberType = 1 << iota
	IsInt
	NaN
	NaI // Not an Integer
	DivByZero
)

func ToNumber(x Value) (Value, NumberType) {
	switch x.(type) {
	case Int:
		return x, IsInt
	case Float:
		return x, IsFloat
	case String:
		exp, err := ast.NumberFromString(string(x.(String)))
		if err == nil {
			switch n := exp.(type) {
			case ast.Int:
				return Int(n.Val()), IsInt
			case ast.Float:
				return Float(n.Val()), IsFloat
			}
		}
	}
	return x, NaN
}

func ToInt(v Value) (Int, NumberType) {
	switch x := v.(type) {
	case Int:
		return x, IsInt
	case Float:
		return x.ToInt()
	case String:
		return x.ToInt()
	}
	return 0, NaN
}
