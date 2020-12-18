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
	return NilValue, NewErrorS("cannot neg")
}

func add(t *Thread, x Value, y Value) (Value, *Error) {
	nx, fx, kx := ToNumber(x)
	ny, fy, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			return IntValue(nx + ny), nil
		case IsFloat:
			return FloatValue(float64(nx) + fy), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return FloatValue(fx + float64(ny)), nil
		case IsFloat:
			return FloatValue(fx + fy), nil
		}
	}
	res, err, ok := metabin(t, "__add", x, y)
	if ok {
		return res, err
	}
	return NilValue, NewErrorS("add expects addable values")
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
	return NilValue, NewErrorS("sub expects subtractable values")
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
	return NilValue, NewErrorS("mul expects multipliable values")
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
	return NilValue, NewErrorS("div expects dividable values")
}

func floordivInt(x, y int64) int64 {
	r := x % y
	q := x / y
	if (r < 0) != (y < 0) {
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
	return NilValue, NewErrorS("idiv expects idividable values")
}

func modInt(x, y int64) int64 {
	r := x % y
	if (r < 0) != (y < 0) {
		r += y
	}
	return r
}

func modFloat(x, y float64) float64 {
	r := math.Mod(x, y)
	if (r < 0) != (y < 0) {
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
	return NilValue, NewErrorS("mod expects modable values")
}

func powFloat(x, y float64) float64 {
	return math.Pow(x, y)
}

func pow(t *Thread, x Value, y Value) (Value, *Error) {
	nx, fx, kx := ToNumber(x)
	ny, fy, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			return FloatValue(powFloat(float64(nx), float64(ny))), nil
		case IsFloat:
			return FloatValue(powFloat(float64(nx), fy)), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return FloatValue(powFloat(fx, float64(ny))), nil
		case IsFloat:
			return FloatValue(powFloat(fx, fy)), nil
		}
	}
	res, err, ok := metabin(t, "__pow", x, y)
	if ok {
		return res, err
	}
	return NilValue, NewErrorS("pow expects powidable values")
}
