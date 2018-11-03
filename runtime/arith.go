package runtime

import (
	"math"
)

func unm(t *Thread, x Value) (Value, *Error) {
	nx, kx := ToNumber(x)
	switch kx {
	case IsInt:
		return -nx.(Int), nil
	case IsFloat:
		return -nx.(Float), nil
	}
	res, err, ok := metaun(t, "__unm", x)
	if ok {
		return res, err
	}
	return nil, NewErrorS("cannot neg")
}

func add(t *Thread, x Value, y Value) (Value, *Error) {
	nx, kx := ToNumber(x)
	ny, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			return nx.(Int) + ny.(Int), nil
		case IsFloat:
			return Float(nx.(Int)) + ny.(Float), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return nx.(Float) + Float(ny.(Int)), nil
		case IsFloat:
			return nx.(Float) + ny.(Float), nil
		}
	}
	res, err, ok := metabin(t, "__add", x, y)
	if ok {
		return res, err
	}
	return nil, NewErrorS("add expects addable values")
}

func sub(t *Thread, x Value, y Value) (Value, *Error) {
	nx, kx := ToNumber(x)
	ny, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			return nx.(Int) - ny.(Int), nil
		case IsFloat:
			return Float(nx.(Int)) - ny.(Float), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return nx.(Float) - Float(ny.(Int)), nil
		case IsFloat:
			return nx.(Float) - ny.(Float), nil
		}
	}
	res, err, ok := metabin(t, "__sub", x, y)
	if ok {
		return res, err
	}
	return nil, NewErrorS("sub expects subtractable values")
}

func mul(t *Thread, x Value, y Value) (Value, *Error) {
	nx, kx := ToNumber(x)
	ny, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			return nx.(Int) * ny.(Int), nil
		case IsFloat:
			return Float(nx.(Int)) * ny.(Float), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return nx.(Float) * Float(ny.(Int)), nil
		case IsFloat:
			return nx.(Float) * ny.(Float), nil
		}
	}
	res, err, ok := metabin(t, "__mul", x, y)
	if ok {
		return res, err
	}
	return nil, NewErrorS("mul expects multipliable values")
}

func div(t *Thread, x Value, y Value) (Value, *Error) {
	nx, kx := ToNumber(x)
	y, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			return Float(nx.(Int)) / Float(y.(Int)), nil
		case IsFloat:
			return Float(nx.(Int)) / y.(Float), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return nx.(Float) / Float(y.(Int)), nil
		case IsFloat:
			return nx.(Float) / y.(Float), nil
		}
	}
	res, err, ok := metabin(t, "__div", x, y)
	if ok {
		return res, err
	}
	return nil, NewErrorS("div expects dividable values")
}

func floordivInt(x, y Int) Int {
	r := x % y
	q := x / y
	if (r < 0) != (y < 0) {
		q--
	}
	return q
}

func floordivFloat(x, y Float) Float {
	return Float(math.Floor(float64(x / y)))
}

func idiv(t *Thread, x Value, y Value) (Value, *Error) {
	nx, kx := ToNumber(x)
	ny, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			return floordivInt(nx.(Int), ny.(Int)), nil
		case IsFloat:
			return floordivFloat(Float(nx.(Int)), ny.(Float)), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return floordivFloat(nx.(Float), Float(ny.(Int))), nil
		case IsFloat:
			return floordivFloat(nx.(Float), ny.(Float)), nil
		}
	}
	res, err, ok := metabin(t, "__idiv", x, y)
	if ok {
		return res, err
	}
	return nil, NewErrorS("idiv expects idividable values")
}

func modInt(x, y Int) Int {
	r := x % y
	if (r < 0) != (y < 0) {
		r += y
	}
	return r
}

func modFloat(x, y Float) Float {
	r := Float(math.Mod(float64(x), float64(y)))

	if (r < 0) != (y < 0) {
		r += y
	}
	return r
}

func Mod(t *Thread, x Value, y Value) (Value, *Error) {
	nx, kx := ToNumber(x)
	ny, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			return modInt(nx.(Int), ny.(Int)), nil
		case IsFloat:
			return modFloat(Float(nx.(Int)), ny.(Float)), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return modFloat(nx.(Float), Float(ny.(Int))), nil
		case IsFloat:
			return modFloat(nx.(Float), ny.(Float)), nil
		}
	}
	res, err, ok := metabin(t, "__mod", x, y)
	if ok {
		return res, err
	}
	return nil, NewErrorS("mod expects modable values")
}

func powFloat(x, y Float) Float {
	return Float(math.Pow(float64(x), float64(y)))
}

func pow(t *Thread, x Value, y Value) (Value, *Error) {
	nx, kx := ToNumber(x)
	ny, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			return powFloat(Float(nx.(Int)), Float(ny.(Int))), nil
		case IsFloat:
			return powFloat(Float(nx.(Int)), ny.(Float)), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return powFloat(nx.(Float), Float(ny.(Int))), nil
		case IsFloat:
			return powFloat(nx.(Float), ny.(Float)), nil
		}
	}
	res, err, ok := metabin(t, "__pow", x, y)
	if ok {
		return res, err
	}
	return nil, NewErrorS("pow expects powidable values")
}
