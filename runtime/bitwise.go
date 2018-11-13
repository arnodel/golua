package runtime

var errNaI = NewErrorS("Float is not an integer")

func band(t *Thread, x, y Value) (Value, *Error) {
	ix, okx := ToInt(x)
	iy, oky := ToInt(y)
	if okx && oky {
		return ix & iy, nil
	}
	res, err, ok := metabin(t, "__band", x, y)
	if ok {
		return res, err
	}
	return nil, NewErrorS("band expects bandable values")
}

func bor(t *Thread, x, y Value) (Value, *Error) {
	ix, okx := ToInt(x)
	iy, oky := ToInt(y)
	if okx && oky {
		return ix | iy, nil
	}
	res, err, ok := metabin(t, "__bor", x, y)
	if ok {
		return res, err
	}
	return nil, NewErrorS("bor expects bordable values")
}

func bxor(t *Thread, x, y Value) (Value, *Error) {
	ix, okx := ToInt(x)
	iy, oky := ToInt(y)
	if okx && oky {
		return ix ^ iy, nil
	}
	res, err, ok := metabin(t, "__bxor", x, y)
	if ok {
		return res, err
	}
	return nil, NewErrorS("bxor expects bxordable values")
}

func shl(t *Thread, x, y Value) (Value, *Error) {
	ix, okx := ToInt(x)
	iy, oky := ToInt(y)
	if okx && oky {
		if iy < 0 {
			return ix >> uint64(-iy), nil
		}
		return ix << uint64(iy), nil
	}
	res, err, ok := metabin(t, "__shl", x, y)
	if ok {
		return res, err
	}
	return nil, NewErrorS("shl expects shldable values")
}

func shr(t *Thread, x, y Value) (Value, *Error) {
	ix, okx := ToInt(x)
	iy, oky := ToInt(y)
	if okx && oky {
		if iy < 0 {
			return ix << uint64(iy), nil
		}
		return ix >> uint64(iy), nil
	}
	res, err, ok := metabin(t, "__shr", x, y)
	if ok {
		return res, err
	}
	return nil, NewErrorS("shr expects shrdable values")
}

func bnot(t *Thread, x Value) (Value, *Error) {
	ix, okx := ToInt(x)
	if okx {
		return ^ix, nil
	}
	res, err, ok := metaun(t, "__bnot", x)
	if ok {
		return res, err
	}
	return nil, NewErrorS("bnot expects a bnotable value")
}
