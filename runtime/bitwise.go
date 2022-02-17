package runtime

import "fmt"

func band(t *Thread, x, y Value) (Value, error) {
	ix, okx := ToIntNoString(x)
	iy, oky := ToIntNoString(y)
	if okx && oky {
		return IntValue(ix & iy), nil
	}
	res, err, ok := metabin(t, "__band", x, y)
	if ok {
		return res, err
	}
	return NilValue, binaryBitwiseError("and", x, y, okx, oky)
}

func bor(t *Thread, x, y Value) (Value, error) {
	ix, okx := ToIntNoString(x)
	iy, oky := ToIntNoString(y)
	if okx && oky {
		return IntValue(ix | iy), nil
	}
	res, err, ok := metabin(t, "__bor", x, y)
	if ok {
		return res, err
	}
	return NilValue, binaryBitwiseError("or", x, y, okx, oky)
}

func bxor(t *Thread, x, y Value) (Value, error) {
	ix, okx := ToIntNoString(x)
	iy, oky := ToIntNoString(y)
	if okx && oky {
		return IntValue(ix ^ iy), nil
	}
	res, err, ok := metabin(t, "__bxor", x, y)
	if ok {
		return res, err
	}
	return NilValue, binaryBitwiseError("xor", x, y, okx, oky)
}

func shl(t *Thread, x, y Value) (Value, error) {
	ix, okx := ToIntNoString(x)
	iy, oky := ToIntNoString(y)

	// We turn the value into an uint64 before shifting so that it's a logical
	// shift, not arithmetic.
	if okx && oky {
		if iy < 0 {
			return IntValue(int64(uint64(ix) >> uint64(-iy))), nil
		}
		return IntValue(int64(uint64(ix) << uint64(iy))), nil
	}
	res, err, ok := metabin(t, "__shl", x, y)
	if ok {
		return res, err
	}
	return NilValue, binaryBitwiseError("shl", x, y, okx, oky)
}

func shr(t *Thread, x, y Value) (Value, error) {
	ix, okx := ToIntNoString(x)
	iy, oky := ToIntNoString(y)

	// We turn the value into an uint64 before shifting so that it's a logical
	// shift, not arithmetic.
	if okx && oky {
		if iy < 0 {
			return IntValue(int64(uint64(ix) << uint64(-iy))), nil
		}
		return IntValue(int64(uint64(ix) >> uint64(iy))), nil
	}
	res, err, ok := metabin(t, "__shr", x, y)
	if ok {
		return res, err
	}
	return NilValue, binaryBitwiseError("shr", x, y, okx, oky)
}

func bnot(t *Thread, x Value) (Value, error) {
	ix, okx := ToIntNoString(x)
	if okx {
		return IntValue(^ix), nil
	}
	res, err, ok := metaun(t, "__bnot", x)
	if ok {
		return res, err
	}
	return NilValue, binaryBitwiseError("not", x, x, false, true)
}

func binaryBitwiseError(op string, x, y Value, okx, oky bool) error {
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
		return fmt.Errorf("number has no integer representation")
	}
	return fmt.Errorf("attempt to perform bitwise %s on a %s value", op, wrongVal.CustomTypeName())
}
