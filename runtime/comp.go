package runtime

// RawEqual returns two values.  The second one is true if raw equality makes
// sense for x and y.  The first one returns whether x and y are raw equal.
func RawEqual(x, y Value) (bool, bool) {
	if x.Equals(y) {
		return true, true
	}
	switch x.NumberType() {
	case IntType:
		if fy, ok := y.TryFloat(); ok {
			return equalIntAndFloat(x.AsInt(), fy), true
		}
	case FloatType:
		if ny, ok := y.TryInt(); ok {
			return equalIntAndFloat(ny, x.AsFloat()), true
		}
	}
	return false, false
}

func equalIntAndFloat(n int64, f float64) bool {
	nf := int64(f)
	return float64(nf) == f && nf == n
}

func eq(t *Thread, x, y Value) (bool, *Error) {
	if res, ok := RawEqual(x, y); ok {
		return res, nil
	}
	if _, ok := x.TryTable(); ok {
		if _, ok := y.TryTable(); !ok {
			return false, nil
		}
	} else {
		// TODO: deal with UserData
		return false, nil
	}
	res, err, ok := metabin(t, "__eq", x, y)
	if ok {
		return Truth(res), err
	}
	return false, nil
}

// Lt returns whether x < y is true (and an error if it's not possible to
// compare them).
func Lt(t *Thread, x, y Value) (bool, *Error) {
	switch x.NumberType() {
	case IntType:
		switch y.NumberType() {
		case IntType:
			return x.AsInt() < y.AsInt(), nil
		case FloatType:
			return ltIntAndFloat(x.AsInt(), y.AsFloat()), nil
		}
	case FloatType:
		switch y.NumberType() {
		case IntType:
			return ltFloatAndInt(x.AsFloat(), y.AsInt()), nil
		case FloatType:
			return x.AsFloat() < y.AsFloat(), nil
		}
	}
	if sx, ok := x.TryString(); ok {
		if sy, ok := y.TryString(); ok {
			return sx < sy, nil
		}
	}
	res, err, ok := metabin(t, "__lt", x, y)
	if ok {
		return Truth(res), err
	}
	return false, compareError(x, y)
}

func ltIntAndFloat(n int64, f float64) bool {
	nf := int64(f)
	if float64(nf) == f {
		return n < nf
	}
	return float64(n) < f
}

func ltFloatAndInt(f float64, n int64) bool {
	nf := int64(f)
	if float64(nf) == f {
		return nf < n
	}
	return f < float64(n)
}

func leIntAndFloat(n int64, f float64) bool {
	nf := int64(f)
	if float64(nf) == f {
		return n <= nf
	}
	return float64(n) <= f
}

func leFloatAndInt(f float64, n int64) bool {
	nf := int64(f)
	if float64(nf) == f {
		return nf <= n
	}
	return f <= float64(n)
}

func le(t *Thread, x, y Value) (bool, *Error) {
	switch x.NumberType() {
	case IntType:
		switch y.NumberType() {
		case IntType:
			return x.AsInt() <= y.AsInt(), nil
		case FloatType:
			return leIntAndFloat(x.AsInt(), y.AsFloat()), nil
		}
	case FloatType:
		switch y.NumberType() {
		case IntType:
			return leFloatAndInt(x.AsFloat(), y.AsInt()), nil
		case FloatType:
			return x.AsFloat() <= y.AsFloat(), nil
		}
	}
	if sx, ok := x.TryString(); ok {
		if sy, ok := y.TryString(); ok {
			return sx <= sy, nil
		}
	}
	res, err, ok := metabin(t, "__le", x, y)
	if ok {
		return Truth(res), err
	}
	return false, compareError(x, y)
}

func compareError(x, y Value) *Error {
	return NewErrorF("attempt to compare a %s value with a %s value", x.CustomTypeName(), y.CustomTypeName())
}
