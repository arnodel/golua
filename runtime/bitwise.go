package runtime

func band(t *Thread, x, y Value) (Value, *Error) {
	ix, okx := ToInt(x)
	iy, oky := ToInt(y)
	if okx && oky {
		return IntValue(ix & iy), nil
	}
	res, err, ok := metabin(t, "__band", x, y)
	if ok {
		return res, err
	}
	return NilValue, binaryBitwiseError("and", x, y, okx, oky)
}

func bor(t *Thread, x, y Value) (Value, *Error) {
	ix, okx := ToInt(x)
	iy, oky := ToInt(y)
	if okx && oky {
		return IntValue(ix | iy), nil
	}
	res, err, ok := metabin(t, "__bor", x, y)
	if ok {
		return res, err
	}
	return NilValue, binaryBitwiseError("or", x, y, okx, oky)
}

func bxor(t *Thread, x, y Value) (Value, *Error) {
	ix, okx := ToInt(x)
	iy, oky := ToInt(y)
	if okx && oky {
		return IntValue(ix ^ iy), nil
	}
	res, err, ok := metabin(t, "__bxor", x, y)
	if ok {
		return res, err
	}
	return NilValue, binaryBitwiseError("xor", x, y, okx, oky)
}

func shl(t *Thread, x, y Value) (Value, *Error) {
	ix, okx := ToInt(x)
	iy, oky := ToInt(y)
	if okx && oky {
		if iy < 0 {
			return IntValue(ix >> uint64(-iy)), nil
		}
		return IntValue(ix << uint64(iy)), nil
	}
	res, err, ok := metabin(t, "__shl", x, y)
	if ok {
		return res, err
	}
	return NilValue, binaryBitwiseError("shl", x, y, okx, oky)
}

func shr(t *Thread, x, y Value) (Value, *Error) {
	ix, okx := ToInt(x)
	iy, oky := ToInt(y)
	if okx && oky {
		if iy < 0 {
			return IntValue(ix << uint64(iy)), nil
		}
		return IntValue(ix >> uint64(iy)), nil
	}
	res, err, ok := metabin(t, "__shr", x, y)
	if ok {
		return res, err
	}
	return NilValue, binaryBitwiseError("shr", x, y, okx, oky)
}

func bnot(t *Thread, x Value) (Value, *Error) {
	ix, okx := ToInt(x)
	if okx {
		return IntValue(^ix), nil
	}
	res, err, ok := metaun(t, "__bnot", x)
	if ok {
		return res, err
	}
	return NilValue, binaryBitwiseError("not", x, x, false, true)
}

func binaryBitwiseError(op string, x, y Value, okx, oky bool) *Error {
	var wrongVal Value
	switch {
	case oky:
		wrongVal = x
	case okx:
		wrongVal = y
	case x.Type() != FloatType:
		wrongVal = x
	case y.Type() != FloatType:
		wrongVal = y
	default:
		// Both x, and y are floats
		wrongVal = x
	}
	if wrongVal.Type() == FloatType {
		return NewErrorF("number has no integer representation")
	}
	return NewErrorF("attempt to perform bitwise %s on a %s value", op, wrongVal.TypeName())
}
