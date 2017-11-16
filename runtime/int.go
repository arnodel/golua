package runtime

import "errors"

type Bool bool
type Int int64
type Float float64
type NilType struct{}
type String string

type Callable interface {
	Call([]Value, []Value) error
}

type ToStringable interface {
	ToString() string
}

type Table struct {
	content map[Value]Value
	meta    *Table
}

type NumberType uint16

const (
	IsFloat NumberType = iota
	IsInt
	NaN
	DivByZero
)

// type NumberPairType uint16

// const (
// 	FF NumberPairType = IsFloat | IsFloat<<8
// 	IF NumberPairType = IsInt | IsInt<<8
// 	FI NumberPairType = IsFloat | IsInt<<8
// 	II NumberPairType = IsInt | IsInt<<8
// )

func (n NumberType) Error() string {
	switch n {
	case NaN:
		return "Not a Number"
	default:
		return "OK"
	}
}

func TruthValue(x Value) bool {
	switch x.(type) {
	case Bool:
		return x.(bool)
	case NilType:
		return false
	default:
		return true
	}
}

func ToNumber(x Value) (Value, NumberType) {
	switch x.(type) {
	case Int:
		return x, IsInt
	case Float:
		return x, IsFloat
	default:
		return nil, NaN
	}
}

func ToInt(x Value) (Int, bool) {
	return 0, true
}

func call(f Value, args []Value, results []Value) error {
	callable, ok := f.(Callable)
	if ok {
		return callable.Call(args, results)
	}
	err, ok := metacall(f, "__call", append([]Value{f}, args...), results)
	if ok {
		return err
	}
	return errors.New("call expects a callable")
}

func metacall(obj Value, method string, args []Value, results []Value) (error, bool) {
	f, err := gettable(obj, String(method))
	if err != nil {
		return nil, false
	}
	return call(f, args, results), true
}

func gettable(t Value, k Value) (Value, error) {
	res := make([]Value, 1)
	err, ok := metacall(t, "__index", []Value{k}, res)
	if ok {
		return res, err
	}
	tbl, ok := t.(Table)
	if ok {
		v, ok := tbl.content[k]
		if !ok {
			return NilType{}, nil
		}
		return v, nil
	}
	return nil, errors.New("gettable expects an indexable")
}

func metabin(f string, x Value, y Value) (Value, error, bool) {
	res := make([]Value, 1)
	xy := []Value{x, y}
	err, ok := metacall(x, f, xy, res)
	if !ok {
		err, ok = metacall(y, f, xy, res)
	}
	if ok {
		return res[0], err, true
	}
	return nil, nil, false
}

func add(x Value, y Value) (Value, error) {
	x, kx := ToNumber(x)
	y, ky := ToNumber(y)
	switch kx | ky<<8 {
	case IsInt | IsInt<<8:
		return x.(Int) + y.(Int), nil
	case IsInt | IsFloat<<8:
		return Float(x.(Int)) + y.(Float), nil
	case IsFloat | IsInt<<8:
		return x.(Float) + Float(y.(Int)), nil
	case IsFloat | IsFloat<<8:
		return x.(Float) + x.(Float), nil
	default:
		res, err, ok := metabin("__add", x, y)
		if ok {
			return res, err
		}
		return nil, errors.New("add expects addable values")
	}
}

func sub(x Value, y Value) (Value, error) {
	x, kx := ToNumber(x)
	y, ky := ToNumber(y)
	switch kx | ky<<8 {
	case IsInt | IsInt<<8:
		return x.(Int) - y.(Int), nil
	case IsInt | IsFloat<<8:
		return Float(x.(Int)) - y.(Float), nil
	case IsFloat | IsInt<<8:
		return x.(Float) - Float(y.(Int)), nil
	case IsFloat | IsFloat<<8:
		return x.(Float) - x.(Float), nil
	default:
		res, err, ok := metabin("__sub", x, y)
		if ok {
			return res, err
		}
		return nil, errors.New("add expects addable values")
	}
}

func band(x Value, y Value) (Value, error) {
	ix, okx := ToInt(x)
	iy, oky := ToInt(y)
	if okx && oky {
		return ix & iy, nil
	}
	res, err, ok := metabin("__band", x, y)
	if ok {
		return res, err
	}
	return nil, errors.New("band expects bitwise andable values")
}

func concat(x Value, y Value) (Value, error) {
	var sx, sy ToStringable
	sx, ok := x.(ToStringable)
	if ok {
		sy, ok = y.(ToStringable)
	}
	if ok {
		return sx.ToString() + sy.ToString(), nil
	}
	res, err, ok := metabin("__concat", x, y)
	if ok {
		return res, err
	}
	return nil, errors.New("concat expects concatable values")
}
