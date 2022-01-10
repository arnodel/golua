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

func add(t *Thread, x Value, y Value) (Value, *Error) {
	res, ok := Add(x, y)
	if ok {
		return res, nil
	}
	res, err, ok := metabin(t, "__add", x, y)
	if ok {
		return res, err
	}
	return NilValue, BinaryArithmeticError("add", x, y, numberType(x), numberType(y))
}

func sub(t *Thread, x Value, y Value) (Value, *Error) {
	nx, fx, kx := ToNumber(x)
	ny, fy, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			return IntValue(nx - ny), nil
		case IsFloat:
			return FloatValue(float64(nx) - fy), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return FloatValue(fx - float64(ny)), nil
		case IsFloat:
			return FloatValue(fx - fy), nil
		}
	}
	res, err, ok := metabin(t, "__sub", x, y)
	if ok {
		return res, err
	}
	return NilValue, BinaryArithmeticError("sub", x, y, kx, ky)
}

func mul(t *Thread, x Value, y Value) (Value, *Error) {
	nx, fx, kx := ToNumber(x)
	ny, fy, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			return IntValue(nx * ny), nil
		case IsFloat:
			return FloatValue(float64(nx) * fy), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return FloatValue(fx * float64(ny)), nil
		case IsFloat:
			return FloatValue(fx * fy), nil
		}
	}
	res, err, ok := metabin(t, "__mul", x, y)
	if ok {
		return res, err
	}
	return NilValue, BinaryArithmeticError("mul", x, y, kx, ky)
}

func div(t *Thread, x Value, y Value) (Value, *Error) {
	nx, fx, kx := ToNumber(x)
	ny, fy, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			return FloatValue(float64(nx) / float64(ny)), nil
		case IsFloat:
			return FloatValue(float64(nx) / fy), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return FloatValue(fx / float64(ny)), nil
		case IsFloat:
			return FloatValue(fx / fy), nil
		}
	}
	res, err, ok := metabin(t, "__div", x, y)
	if ok {
		return res, err
	}
	return NilValue, BinaryArithmeticError("div", x, y, kx, ky)
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

func idiv(t *Thread, x Value, y Value) (Value, *Error) {
	nx, fx, kx := ToNumber(x)
	ny, fy, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			if ny == 0 {
				return NilValue, NewErrorS("attempt to divide by zero")
			}
			return IntValue(floordivInt(nx, ny)), nil
		case IsFloat:
			return FloatValue(floordivFloat(float64(nx), fy)), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return FloatValue(floordivFloat(fx, float64(ny))), nil
		case IsFloat:
			return FloatValue(floordivFloat(fx, fy)), nil
		}
	}
	res, err, ok := metabin(t, "__idiv", x, y)
	if ok {
		return res, err
	}
	return NilValue, BinaryArithmeticError("idiv", x, y, kx, ky)
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
	return NilValue, BinaryArithmeticError("mod", x, y, kx, ky)
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
	return NilValue, BinaryArithmeticError("pow", x, y, numberType(x), numberType(y))
}

func BinaryArithmeticError(op string, x, y Value, kx, ky NumberType) *Error {
	var wrongVal Value
	switch {
	case ky != NaN:
		wrongVal = x
	case kx != NaN:
		wrongVal = y
	default:
		return NewErrorF("attempt to %s a '%s' with a '%s'", op, x.CustomTypeName(), y.CustomTypeName())
	}
	return NewErrorF("attempt to perform arithmetic on a %s value", wrongVal.CustomTypeName())
}
