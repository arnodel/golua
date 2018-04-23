package runtime

import "errors"

func floatToInt(x Float) (Int, bool) {
	n := Int(x)
	return n, Float(n) == x
}

func ToInt(v Value) (Int, NumberType) {
	switch x := v.(type) {
	case Int:
		return x, IsInt
	case Float:
		n, ok := floatToInt(x)
		if !ok {
			return 0, NaI
		}
		return n, IsInt
	case String:
		v, k := ToNumber(x)
		if k&(IsInt|IsFloat) != 0 {
			return ToInt(v)
		}
		return 0, k
	}
	return 0, NaN
}

var errNaI = errors.New("Float is not an integer")

func band(t *Thread, x, y Value) (Value, error) {
	ix, kx := ToInt(x)
	iy, ky := ToInt(y)
	k := kx | ky
	switch {
	case k == IsInt:
		return ix & iy, nil
	case k&NaI != 0:
		return nil, errNaI
	}
	res, err, ok := metabin(t, "__band", x, y)
	if ok {
		return res, err
	}
	return nil, errors.New("band expects bandable values")
}

func bor(t *Thread, x, y Value) (Value, error) {
	ix, kx := ToInt(x)
	iy, ky := ToInt(y)
	k := kx | ky
	switch {
	case k == IsInt:
		return ix | iy, nil
	case k&NaI != 0:
		return nil, errNaI
	}
	res, err, ok := metabin(t, "__or", x, y)
	if ok {
		return res, err
	}
	return nil, errors.New("bor expects bordable values")
}

func bxor(t *Thread, x, y Value) (Value, error) {
	ix, kx := ToInt(x)
	iy, ky := ToInt(y)
	k := kx | ky
	switch {
	case k == IsInt:
		return ix ^ iy, nil
	case k&NaI != 0:
		return nil, errNaI
	}
	res, err, ok := metabin(t, "__xor", x, y)
	if ok {
		return res, err
	}
	return nil, errors.New("bxor expects bxordable values")
}

func shl(t *Thread, x, y Value) (Value, error) {
	ix, kx := ToInt(x)
	iy, ky := ToInt(y)
	k := kx | ky
	switch {
	case k == IsInt:
		if iy < 0 {
			return ix >> uint64(-iy), nil
		}
		return ix << uint64(iy), nil
	case k&NaI != 0:
		return nil, errNaI
	}
	res, err, ok := metabin(t, "__shl", x, y)
	if ok {
		return res, err
	}
	return nil, errors.New("shl expects shldable values")
}

func shr(t *Thread, x, y Value) (Value, error) {
	ix, kx := ToInt(x)
	iy, ky := ToInt(y)
	k := kx | ky
	switch {
	case k == IsInt:
		if iy < 0 {
			return ix << uint64(iy), nil
		}
		return ix >> uint64(iy), nil
	case k&NaI != 0:
		return nil, errNaI
	}
	res, err, ok := metabin(t, "__shr", x, y)
	if ok {
		return res, err
	}
	return nil, errors.New("shr expects shrdable values")
}

func bnot(t *Thread, x Value) (Value, error) {
	ix, kx := ToInt(x)
	if kx == IsInt {
		return ^ix, nil
	}
	res, err, ok := metaun(t, "__bnot", x)
	if ok {
		return res, err
	}
	return nil, errors.New("bnot expects a bnotable value")
}
