package runtime

import (
	"errors"
	"strconv"
)

func getindex(t *Thread, coll Value, idx Value) (Value, error) {
	if tbl, ok := coll.(*Table); ok {
		if val := rawget(tbl, idx); val != nil {
			return val, nil
		}
	}
	metaIdx := rawget(getmetatable(coll), "__index")
	if metaIdx == nil {
		return nil, nil
	}
	switch metaIdx.(type) {
	case *Table:
		return getindex(t, metaIdx, idx)
	default:
		res := NewTerminationWith(1, false)
		if err := Call(t, metaIdx, []Value{idx}, res); err != nil {
			return nil, err
		}
		return res.Get(0), nil
	}
}

func setindex(t *Thread, coll Value, idx Value, val Value) error {
	if tbl, ok := coll.(*Table); ok {
		if tbl.Get(idx) != nil {
			tbl.Set(idx, val)
			return nil
		}
	}
	metaNewIndex := rawget(getmetatable(coll), "__newindex")
	if metaNewIndex == nil {
		return nil
	}
	switch metaNewIndex.(type) {
	case *Table:
		return setindex(t, metaNewIndex, idx, val)
	default:
		return Call(t, metaNewIndex, []Value{coll, idx, val}, nil)
	}
}

func truth(v Value) bool {
	if v == nil {
		return false
	}
	switch x := v.(type) {
	case NilType:
		return false
	case Bool:
		return bool(x)
	default:
		return true
	}
}

func Metacall(t *Thread, obj Value, method string, args []Value, next Continuation) (error, bool) {
	meta := getmetatable(obj)
	if meta != nil {
		if f := rawget(meta, String(method)); f != nil {
			return Call(t, f, args, next), true
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
	return t.Get(k)
}

func Call(t *Thread, f Value, args []Value, next Continuation) error {
	callable, ok := f.(Callable)
	if ok {
		return t.RunContinuation(ContWithArgs(callable, args, next))
	}
	err, ok := Metacall(t, f, "__call", append([]Value{f}, args...), next)
	if ok {
		return err
	}
	return errors.New("call expects a callable")
}

func metabin(t *Thread, f string, x Value, y Value) (Value, error, bool) {
	xy := []Value{x, y}
	res := NewTerminationWith(1, false)
	err, ok := Metacall(t, x, f, xy, res)
	if !ok {
		err, ok = Metacall(t, y, f, xy, res)
	}
	if ok {
		return res.Get(0), err, true
	}
	return nil, nil, false
}

func metaun(t *Thread, f string, x Value) (Value, error, bool) {
	res := NewTerminationWith(1, false)
	err, ok := Metacall(t, x, f, []Value{x}, res)
	if ok {
		return res.Get(0), err, true
	}
	return nil, nil, false
}

func AsString(x Value) (String, bool) {
	switch xx := x.(type) {
	case String:
		return xx, true
	case Int:
		return String(strconv.Itoa(int(xx))), true
	case Float:
		return String(strconv.FormatFloat(float64(xx), 'g', -1, 64)), true
	case Bool:
		if xx {
			return String("true"), true
		}
		return String("false"), true
	}
	return String(""), false
}

func concat(t *Thread, x, y Value) (Value, error) {
	if sx, ok := AsString(x); ok {
		if sy, ok := AsString(y); ok {
			return sx + sy, nil
		}
	}
	res, err, ok := metabin(t, "__concat", x, y)
	if ok {
		return res, err
	}
	return nil, errors.New("concat expects concatable values")
}

func length(t *Thread, v Value) (Value, error) {
	if s, ok := v.(String); ok {
		return Int(len(s)), nil
	}
	res := NewTerminationWith(1, false)
	err, ok := Metacall(t, v, "__len", []Value{v}, res)
	if ok {
		return res.Get(0), err
	}
	if tbl, ok := v.(*Table); ok {
		return tbl.Len(), nil
	}
	return nil, errors.New("Cannot compute len")
}

func Type(v Value) String {
	if v == nil {
		return String("nil")
	}
	switch v.(type) {
	case String:
		return String("string")
	case Int, Float:
		return String("number")
	case *Table:
		return String("table")
	case NilType:
		return String("nil")
	case Bool:
		return String("bool")
	case *Closure:
		return String("function")
	}
	return String("unknown")
}

func SetEnvFunc(t *Table, name string, f func(*Thread, []Value, Continuation) error) {
	t.Set(String(name), GoFunction(f))
}
