package runtime

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
