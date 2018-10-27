package runtime

import "strings"

type NumberType uint16

const (
	IsFloat NumberType = 1 << iota
	IsInt
	NaN
	NaI // Not an Integer
	DivByZero
)

func ToNumber(x Value) (Value, NumberType) {
	switch xx := x.(type) {
	case Int:
		return x, IsInt
	case Float:
		return x, IsFloat
	case String:
		return String(strings.Trim(string(xx), " ")).ToNumber()
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

func ToFloat(v Value) (Float, bool) {
	switch x := v.(type) {
	case Int:
		return Float(x), true
	case Float:
		return x, true
	case String:
		vv, tp := x.ToNumber()
		switch tp {
		case IsInt:
			return Float(vv.(Int)), true
		case IsFloat:
			return vv.(Float), true
		}
	}
	return 0, false
}
