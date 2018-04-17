package runtime

import (
	"errors"

	"github.com/arnodel/golua/ast"
)

type NumberType uint16

const (
	IsFloat NumberType = iota
	IsInt
	NaN
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

func neg(t *Thread, x Value) (Value, error) {
	x, kx := ToNumber(x)
	switch kx {
	case IsInt:
		return -x.(Int), nil
	case IsFloat:
		return -x.(Float), nil
	}
	res, err, ok := metaun(t, "__unm", x)
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

func metabin(t *Thread, f string, x Value, y Value) (Value, error, bool) {
	res := make([]Value, 1)
	xy := []Value{x, y}
	err, ok := metacall(t, x, f, xy, res)
	if !ok {
		err, ok = metacall(t, y, f, xy, res)
	}
	if ok {
		return res[0], err, true
	}
	return nil, nil, false
}

func metaun(t *Thread, f string, x Value) (Value, error, bool) {
	res := make([]Value, 1)
	err, ok := metacall(t, x, f, []Value{x}, res)
	if ok {
		return res[0], err, true
	}
	return nil, nil, false
}

func metacall(t *Thread, obj Value, method string, args []Value, results []Value) (error, bool) {
	meta := getmetatable(obj)
	if meta != nil {
		if f := rawget(meta, String(method)); f != nil {
			return call(t, f, args, results), true
		}
	}
	return nil, false
}

func getmetatable(v Value) *Table {
	mv, ok := v.(Metatabler)
	if !ok {
		return nil
	}
	meta := mv.Metatable()
	metam := rawget(meta, "__metatable")
	if metam != nil {
		// Here we assume that a metatable must be a table...
		return metam.(*Table)
	}
	return meta
}

func rawget(t *Table, k Value) Value {
	if t == nil {
		return nil
	}
	return t.content[k]
}

func call(t *Thread, f Value, args []Value, results []Value) error {
	callable, ok := f.(Callable)
	if ok {
		return callable.Call(t, args, results)
	}
	err, ok := metacall(t, f, "__call", append([]Value{f}, args...), results)
	if ok {
		return err
	}
	return errors.New("call expects a callable")
}
