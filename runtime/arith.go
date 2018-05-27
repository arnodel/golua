package runtime

import (
	"math"
)

func unm(t *Thread, x Value) (Value, *Error) {
	x, kx := ToNumber(x)
	switch kx {
	case IsInt:
		return -x.(Int), nil
	case IsFloat:
		return -x.(Float), nil
	}
	res, err, ok := metaun(t, "__unm", x)
	if ok {
		return res, err
	}
	return nil, NewErrorS("cannot neg")
}

func add(t *Thread, x Value, y Value) (Value, *Error) {
	x, kx := ToNumber(x)
	y, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			return x.(Int) + y.(Int), nil
		case IsFloat:
			return Float(x.(Int)) + y.(Float), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return x.(Float) + Float(y.(Int)), nil
		case IsFloat:
			return x.(Float) + y.(Float), nil
		}
	}
	res, err, ok := metabin(t, "__add", x, y)
	if ok {
		return res, err
	}
	return nil, NewErrorS("add expects addable values")
}

// func add(t *Thread, x Value, y Value) (Value, *Error) {
// 	switch xx := x.(type) {
// 	case Int:
// 		switch yy := y.(type) {
// 		case Int:
// 			return xx + yy, nil
// 		case Float:
// 			return Float(xx) + yy, nil
// 		case String:
// 			return xx, nil
// 		}
// 	case Float:
// 		switch yy := y.(type) {
// 		case Int:
// 			return xx + Float(yy), nil
// 		case Float:
// 			return xx + yy, nil
// 		case String:
// 			return xx, nil
// 		}
// 	case String:
// 		return xx, nil
// 	}
// 	res, err, ok := metabin(t, "__add", x, y)
// 	if ok {
// 		return res, err
// 	}
// 	return nil, NewErrorS("add expects addable values")
// }

func sub(t *Thread, x Value, y Value) (Value, *Error) {
	x, kx := ToNumber(x)
	y, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			return x.(Int) - y.(Int), nil
		case IsFloat:
			return Float(x.(Int)) - y.(Float), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return x.(Float) - Float(y.(Int)), nil
		case IsFloat:
			return x.(Float) - y.(Float), nil
		}
	}
	res, err, ok := metabin(t, "__sub", x, y)
	if ok {
		return res, err
	}
	return nil, NewErrorS("sub expects subtractable values")
}

func mul(t *Thread, x Value, y Value) (Value, *Error) {
	x, kx := ToNumber(x)
	y, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			return x.(Int) * y.(Int), nil
		case IsFloat:
			return Float(x.(Int)) * y.(Float), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return x.(Float) * Float(y.(Int)), nil
		case IsFloat:
			return x.(Float) * y.(Float), nil
		}
	}
	res, err, ok := metabin(t, "__mul", x, y)
	if ok {
		return res, err
	}
	return nil, NewErrorS("mul expects multipliable values")
}

func div(t *Thread, x Value, y Value) (Value, *Error) {
	x, kx := ToNumber(x)
	y, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			return Float(x.(Int)) / Float(y.(Int)), nil
		case IsFloat:
			return Float(x.(Int)) / y.(Float), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return x.(Float) / Float(y.(Int)), nil
		case IsFloat:
			return x.(Float) / y.(Float), nil
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
	x, kx := ToNumber(x)
	y, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			return floordivInt(x.(Int), y.(Int)), nil
		case IsFloat:
			return floordivFloat(Float(x.(Int)), y.(Float)), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return floordivFloat(x.(Float), Float(y.(Int))), nil
		case IsFloat:
			return floordivFloat(x.(Float), y.(Float)), nil
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
	x, kx := ToNumber(x)
	y, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			return modInt(x.(Int), y.(Int)), nil
		case IsFloat:
			return modFloat(Float(x.(Int)), y.(Float)), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return modFloat(x.(Float), Float(y.(Int))), nil
		case IsFloat:
			return modFloat(x.(Float), y.(Float)), nil
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
	x, kx := ToNumber(x)
	y, ky := ToNumber(y)
	switch kx {
	case IsInt:
		switch ky {
		case IsInt:
			return powFloat(Float(x.(Int)), Float(y.(Int))), nil
		case IsFloat:
			return powFloat(Float(x.(Int)), y.(Float)), nil
		}
	case IsFloat:
		switch ky {
		case IsInt:
			return powFloat(x.(Float), Float(y.(Int))), nil
		case IsFloat:
			return powFloat(x.(Float), y.(Float)), nil
		}
	}
	res, err, ok := metabin(t, "__pow", x, y)
	if ok {
		return res, err
	}
	return nil, NewErrorS("pow expects powidable values")
}
