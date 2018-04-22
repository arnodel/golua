package runtime

import (
	"errors"
	"math"

	"github.com/arnodel/golua/ast"
)

type NumberType uint16

const (
	IsFloat NumberType = 1 << iota
	IsInt
	NaN
	NaI // Not an Integer
	DivByZero
)

func ToNumber(x Value) (Value, NumberType) {
	switch x.(type) {
	case Int:
		return x, IsInt
	case Float:
		return x, IsFloat
	case String:
		exp, err := ast.NumberFromString(string(x.(String)))
		if err == nil {
			switch n := exp.(type) {
			case ast.Int:
				return Int(n), IsInt
			case ast.Float:
				return Float(n), IsFloat
			}
		}
	}
	return nil, NaN
}

func unm(t *Thread, x Value) (Value, error) {
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
	return nil, errors.New("cannot neg")
}

func add(t *Thread, x Value, y Value) (Value, error) {
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
	return nil, errors.New("add expects addable values")
}

func sub(t *Thread, x Value, y Value) (Value, error) {
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
	return nil, errors.New("sub expects subtractable values")
}

func mul(t *Thread, x Value, y Value) (Value, error) {
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
	res, err, ok := metabin(t, "__sub", x, y)
	if ok {
		return res, err
	}
	return nil, errors.New("mul expects multipliable values")
}

func div(t *Thread, x Value, y Value) (Value, error) {
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
	return nil, errors.New("div expects dividable values")
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

func idiv(t *Thread, x Value, y Value) (Value, error) {
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
	res, err, ok := metabin(t, "__div", x, y)
	if ok {
		return res, err
	}
	return nil, errors.New("idiv expects idividable values")
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

func mod(t *Thread, x Value, y Value) (Value, error) {
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
	return nil, errors.New("mod expects modable values")
}

func powFloat(x, y Float) Float {
	return Float(math.Pow(float64(x), float64(y)))
}

func pow(t *Thread, x Value, y Value) (Value, error) {
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
	return nil, errors.New("pow expects powidable values")
}
