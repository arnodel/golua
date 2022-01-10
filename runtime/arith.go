package runtime

import (
	"math"
)

func unm(t *Thread, x Value) (Value, *Error) {
	nx, fx, kx := ToNumber(x)
	switch kx {
	case IsInt:
		return IntValue(-nx), nil
	case IsFloat:
		return FloatValue(-fx), nil
	}
	res, err, ok := metaun(t, "__unm", x)
	if ok {
		return res, err
	}
	return NilValue, NewErrorF("attempt to unm a '%s'", x.CustomTypeName())
}

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
func Mod(t *Thread, x Value, y Value) (Value, *Error) {
	nx, fx, kx := ToNumber(x)
	ny, fy, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			if ny == 0 {
				return NilValue, NewErrorS("attempt to perform 'n%0'")
			}
			return IntValue(modInt(nx, ny)), nil
		case IsFloat:
			return FloatValue(modFloat(float64(nx), fy)), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return FloatValue(modFloat(fx, float64(ny))), nil
		case IsFloat:
			return FloatValue(modFloat(fx, fy)), nil
		}
	}
	res, err, ok := metabin(t, "__mod", x, y)
	if ok {
		return res, err
	}
	return NilValue, BinaryArithmeticError("mod", x, y)
}

func powFloat(x, y float64) float64 {
	return math.Pow(x, y)
}

func pow(t *Thread, x Value, y Value) (Value, *Error) {
	fx, okx := ToFloat(x)
	fy, oky := ToFloat(y)
	if okx && oky {
		return FloatValue(powFloat(fx, fy)), nil
	}
	res, err, ok := metabin(t, "__pow", x, y)
	if ok {
		return res, err
	}
	return NilValue, BinaryArithmeticError("pow", x, y)
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
