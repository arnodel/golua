package runtime

import (
	"math"
)

// Unm returns (z, true) where z is the value representing -x if x is a number,
// else (NilValue, false).
func Unm(x Value) (Value, bool) {
	switch x.iface.(type) {
	case int64:
		return IntValue(-x.AsInt()), true
	case float64:
		return FloatValue(-x.AsFloat()), true
	default:
		return NilValue, false
	}
}

// Add returns (z, true) where z is the value representing x+y if x and y are
// numbers, else (NilValue, false).
func Add(x, y Value) (Value, bool) {
	switch x.iface.(type) {
	case int64:
		switch y.iface.(type) {
		case int64:
			return IntValue(x.AsInt() + y.AsInt()), true
		case float64:
			return FloatValue(float64(x.AsInt()) + y.AsFloat()), true
		}
	case float64:
		switch y.iface.(type) {
		case int64:
			return FloatValue(x.AsFloat() + float64(y.AsInt())), true
		case float64:
			return FloatValue(x.AsFloat() + y.AsFloat()), true
		}
	}
	return NilValue, false
}

// Sub returns (z, true) where z is the value representing x-y if x and y are
// numbers, else (NilValue, false).
func Sub(x, y Value) (Value, bool) {
	switch x.iface.(type) {
	case int64:
		switch y.iface.(type) {
		case int64:
			return IntValue(x.AsInt() - y.AsInt()), true
		case float64:
			return FloatValue(float64(x.AsInt()) - y.AsFloat()), true
		}
	case float64:
		switch y.iface.(type) {
		case int64:
			return FloatValue(x.AsFloat() - float64(y.AsInt())), true
		case float64:
			return FloatValue(x.AsFloat() - y.AsFloat()), true
		}
	}
	return NilValue, false
}

func Mul(x, y Value) (Value, bool) {
	switch x.iface.(type) {
	case int64:
		switch y.iface.(type) {
		case int64:
			return IntValue(x.AsInt() * y.AsInt()), true
		case float64:
			return FloatValue(float64(x.AsInt()) * y.AsFloat()), true
		}
	case float64:
		switch y.iface.(type) {
		case int64:
			return FloatValue(x.AsFloat() * float64(y.AsInt())), true
		case float64:
			return FloatValue(x.AsFloat() * y.AsFloat()), true
		}
	}
	return NilValue, false
}

func Div(x, y Value) (Value, bool) {
	switch x.iface.(type) {
	case int64:
		switch y.iface.(type) {
		case int64:
			return FloatValue(float64(x.AsInt()) / float64(y.AsInt())), true
		case float64:
			return FloatValue(float64(x.AsInt()) / y.AsFloat()), true
		}
	case float64:
		switch y.iface.(type) {
		case int64:
			return FloatValue(x.AsFloat() / float64(y.AsInt())), true
		case float64:
			return FloatValue(x.AsFloat() / y.AsFloat()), true
		}
	}
	return NilValue, false
}

func floordivInt(x, y int64) int64 {
	r := x % y
	q := x / y
	if r != 0 && (r < 0) != (y < 0) {
		q--
	}
	return q
}

func floordivFloat(x, y float64) float64 {
	return math.Floor(x / y)
}

func Idiv(x Value, y Value) (Value, bool, *Error) {
	switch x.iface.(type) {
	case int64:
		switch y.iface.(type) {
		case int64:
			ny := y.AsInt()
			if ny == 0 {
				return NilValue, true, NewErrorS("attempt to divide by zero")
			}
			return IntValue(floordivInt(x.AsInt(), ny)), true, nil
		case float64:
			return FloatValue(floordivFloat(float64(x.AsInt()), y.AsFloat())), true, nil
		}
	case float64:
		switch y.iface.(type) {
		case int64:
			return FloatValue(floordivFloat(x.AsFloat(), float64(y.AsInt()))), true, nil
		case float64:
			return FloatValue(floordivFloat(x.AsFloat(), y.AsFloat())), true, nil
		}
	}
	return NilValue, false, nil
}

func modInt(x, y int64) int64 {
	r := x % y
	if r != 0 && (r < 0) != (y < 0) {
		r += y
	}
	return r
}

func modFloat(x, y float64) float64 {
	r := math.Mod(x, y)
	if r != 0 && (r < 0) != (y < 0) {
		r += y
	}
	return r
}

// Mod returns x % y.
func Mod(x Value, y Value) (Value, bool, *Error) {
	switch x.iface.(type) {
	case int64:
		switch y.iface.(type) {
		case int64:
			ny := y.AsInt()
			if ny == 0 {
				return NilValue, true, NewErrorS("attempt to perform 'n%0'")
			}
			return IntValue(modInt(x.AsInt(), ny)), true, nil
		case float64:
			return FloatValue(modFloat(float64(x.AsInt()), y.AsFloat())), true, nil
		}
	case float64:
		switch y.iface.(type) {
		case int64:
			return FloatValue(modFloat(x.AsFloat(), float64(y.AsInt()))), true, nil
		case float64:
			return FloatValue(modFloat(x.AsFloat(), y.AsFloat())), true, nil
		}
	}
	return NilValue, false, nil
}

func powFloat(x, y float64) float64 {
	return math.Pow(x, y)
}

func Pow(x, y Value) (Value, bool) {
	var fx, fy float64
	switch x.iface.(type) {
	case int64:
		fx = float64(x.AsInt())
	case float64:
		fx = x.AsFloat()
	default:
		return NilValue, false
	}
	switch y.iface.(type) {
	case int64:
		fy = float64(y.AsInt())
	case float64:
		fy = y.AsFloat()
	default:
		return NilValue, false
	}
	return FloatValue(powFloat(fx, fy)), true
}

func BinaryArithFallback(t *Thread, op string, x, y Value) (Value, *Error) {
	res, err, ok := metabin(t, op, x, y)
	if ok {
		return res, err
	}
	return NilValue, BinaryArithmeticError(op[2:], x, y)
}

func BinaryArithmeticError(op string, x, y Value) *Error {
	var wrongVal Value
	switch {
	case numberType(y) != NaN:
		wrongVal = x
	case numberType(x) != NaN:
		wrongVal = y
	default:
		return NewErrorF("attempt to %s a '%s' with a '%s'", op, x.CustomTypeName(), y.CustomTypeName())
	}
	return NewErrorF("attempt to perform arithmetic on a %s value", wrongVal.CustomTypeName())
}

func UnaryArithFallback(t *Thread, op string, x Value) (Value, *Error) {
	res, err, ok := metaun(t, op, x)
	if ok {
		return res, err
	}
	return NilValue, UnaryArithmeticError(op[2:], x)
}

func UnaryArithmeticError(op string, x Value) *Error {
	return NewErrorF("attempt to %s a '%s'", op, x.CustomTypeName())
}
