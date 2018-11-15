package runtime

import "strings"

// NumberType represents a type of number
type NumberType uint16

const (
	// IsFloat is the type of Floats
	IsFloat NumberType = 1 << iota
	// IsInt is the type of Ints
	IsInt
	// NaN is the type of values which are not numbers
	NaN
	// NaI is a type for values which a not Ints
	NaI
)

// ToNumber returns x as a Float or Int, and the type (IsFloat, IsInt or NaN).
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

// ToInt returns v as an Int and true if v is actually a valid integer.
func ToInt(v Value) (Int, bool) {
	switch x := v.(type) {
	case Int:
		return x, true
	case Float:
		n, tp := x.ToInt()
		return n, tp == IsInt
	case String:
		n, tp := x.ToInt()
		return n, tp == IsInt
	}
	return 0, false
}

// ToFloat returns v as a FLoat and true if v is a valid float.
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
