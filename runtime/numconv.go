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
	switch v.Type() {
	case IntType:
		return v.AsInt(), 0, IsInt
	case FloatType:
		return 0, v.AsFloat(), IsFloat
	case StringType:
		return stringToNumber(strings.Trim(v.AsString(), " "))
	}
	return 0, 0, NaN
}

// ToNumberValue returns x as a Float or Int, and if it is a number.
func ToNumberValue(v Value) (Value, NumberType) {
	switch v.Type() {
	case IntType:
		return v, IsInt
	case FloatType:
		return v, IsFloat
	case StringType:
		n, f, tp := stringToNumber(strings.Trim(v.AsString(), " "))
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
	switch v.Type() {
	case IntType:
		return v.AsInt(), true
	case FloatType:
		n, tp := floatToInt(v.AsFloat())
		return n, tp == IsInt
	case StringType:
		n, tp := stringToInt(v.AsString())
		return n, tp == IsInt
	}
	return 0, false
}

// ToFloat returns v as a FLoat and true if v is a valid float.
func ToFloat(v Value) (float64, bool) {
	switch v.Type() {
	case IntType:
		return float64(v.AsInt()), true
	case FloatType:
		return v.AsFloat(), true
	case StringType:
		n, f, tp := stringToNumber(v.AsString())
		switch tp {
		case IsInt:
			return float64(n), true
		case IsFloat:
			return f, true
		}
	}
	return 0, false
}
