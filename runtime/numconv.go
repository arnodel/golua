package runtime

import (
	"strconv"
	"strings"
)

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

func numberType(v Value) NumberType {
	switch v.iface.(type) {
	case int64:
		return IsInt
	case float64:
		return IsFloat
	default:
		return NaN
	}
}

// ToNumber returns x as a Float or Int, and the type (IsFloat, IsInt or NaN).
func ToNumber(v Value) (int64, float64, NumberType) {
	switch v.iface.(type) {
	case int64:
		return v.AsInt(), 0, IsInt
	case float64:
		return 0, v.AsFloat(), IsFloat
	case string:
		s := v.AsString()
		return StringToNumber(strings.TrimSpace(s))
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
		n, f, tp := StringToNumber(strings.TrimSpace(s))
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
	switch v.iface.(type) {
	case int64:
		return v.AsInt(), true
	case float64:
		n, tp := FloatToInt(v.AsFloat())
		return n, tp == IsInt
	}
	return 0, false
}

// ToFloat returns v as a FLoat and true if v is a valid float.
func ToFloat(v Value) (float64, bool) {
	switch v.iface.(type) {
	case int64:
		return float64(v.AsInt()), true
	case float64:
		return v.AsFloat(), true
	case string:
		n, f, tp := StringToNumber(v.AsString())
		switch tp {
		case IsInt:
			return float64(n), true
		case IsFloat:
			return f, true
		}
	}
	return 0, false
}

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
	n, f, tp := StringToNumber(s)
	switch tp {
	case IsInt:
		return n, IsInt
	case IsFloat:
		return FloatToInt(f)
	}
	return 0, NaN
}

func StringToNumber(s string) (n int64, f float64, tp NumberType) {
	s = strings.TrimSpace(s)
	var err error
	if len(s) == 0 {
		tp = NaN
		return
	}
	var i0 = 0
	// If the string starts with -?0[xX] then it may be an hex number
	if s[0] == '+' {
		s = s[1:]
	} else if s[0] == '-' || s[0] == '+' {
		i0++
	}
	var isHex = len(s) >= 2+i0 && s[i0] == '0' && (s[i0+1] == 'x' || s[i0+1] == 'X')
	var isFloat = isHex && strings.ContainsAny(s, ".pP") || !isHex && strings.ContainsAny(s, ".eE")
	if isFloat {
		// This is to make strconv.ParseFloat happy
		if isHex && !strings.ContainsAny(s, "pP") {
			s = s + "p0"
		}
		f, err = strconv.ParseFloat(s, 64)
		if err != nil && f == 0 {
			tp = NaN
			return
		}
		tp = IsFloat
		return
	}

	// If s is an hex number, it is parsed as a uint of 64 bits
	if isHex {
		us := s[2+i0:]
		if len(us) > 16 {
			us = us[len(us)-16:]
		}
		un, err := strconv.ParseUint(us, 16, 64)
		if err != nil {
			tp = NaN
			return
		}
		n = int64(un)
		if s[0] == '-' {
			n = -n
		}
		tp = IsInt
		return
	}
	n, err = strconv.ParseInt(s, 10, 64)
	if err != nil {
		if err.(*strconv.NumError).Err == strconv.ErrRange {
			// Try a float instead
			f, err = strconv.ParseFloat(s, 64)
			if err == nil || f != 0 {
				tp = IsFloat
				return
			}
		}
		tp = NaN
		return
	}
	tp = IsInt
	return
}
