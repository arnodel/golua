package runtime

import "errors"

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
		res := make([]Value, 1)
		if err := call(t, metaIdx, []Value{idx}, res); err != nil {
			return nil, err
		}
		return res[0], nil
	}
}

func setindex(t *Thread, coll Value, idx Value, val Value) error {
	if tbl, ok := coll.(*Table); ok {
		if _, ok := tbl.content[idx]; ok {
			tbl.content[idx] = val
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
		return call(t, metaNewIndex, []Value{coll, idx, val}, nil)
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
		return t.RunContinuation(Call(callable, args, NewTermination(results, nil)))
	}
	err, ok := metacall(t, f, "__call", append([]Value{f}, args...), results)
	if ok {
		return err
	}
	return errors.New("call expects a callable")
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
