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
func ToNumber(v Value) (int64, float64, NumberType) {
	if n, ok := v.TryInt(); ok {
		return n, 0, IsInt
	}
	if f, ok := v.TryFloat(); ok {
		return 0, f, IsFloat
	}
	if s, ok := v.TryString(); ok {
		return stringToNumber(strings.Trim(s, " "))
	}
	return 0, 0, NaN
}

// ToNumberValue returns x as a Float or Int, and if it is a number.
func ToNumberValue(v Value) (Value, NumberType) {
	switch v.NumberType() {
	case IntType:
		return v, IsInt
	case FloatType:
		return v, IsFloat
	}
	if s, ok := v.TryString(); ok {
		n, f, tp := stringToNumber(strings.Trim(s, " "))
		switch tp {
		case IsInt:
			return IntValue(n), IsInt
		case IsFloat:
			return FloatValue(f), IsFloat
		}
	}
	return NilValue, NaN
}

// ToInt returns v as an Int and true if v is actually a valid integer.
func ToInt(v Value) (int64, bool) {
	if n, ok := v.TryInt(); ok {
		return n, true
	}
	if f, ok := v.TryFloat(); ok {
		n, tp := FloatToInt(f)
		return n, tp == IsInt
	}
	if s, ok := v.TryString(); ok {
		n, tp := stringToInt(s)
		return n, tp == IsInt
	}
	return 0, false
}

// ToIntNoString returns v as an Int and true if v is actually a valid integer.
func ToIntNoString(v Value) (int64, bool) {
	if n, ok := v.TryInt(); ok {
		return n, true
	}
	if f, ok := v.TryFloat(); ok {
		n, tp := FloatToInt(f)
		return n, tp == IsInt
	}
	return 0, false
}

// ToFloat returns v as a FLoat and true if v is a valid float.
func ToFloat(v Value) (float64, bool) {
	if n, ok := v.TryInt(); ok {
		return float64(n), true
	}
	if f, ok := v.TryFloat(); ok {
		return f, true
	}
	if s, ok := v.TryString(); ok {
		n, f, tp := stringToNumber(s)
		switch tp {
		case IsInt:
			return float64(n), true
		case IsFloat:
			return f, true
		}
	}
	return 0, false
}
