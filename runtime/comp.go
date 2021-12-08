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
			return float64(x.AsInt()) == fy, true
		}
	case FloatType:
		if ny, ok := y.TryInt(); ok {
			return x.AsFloat() == float64(ny), true
		}
	}
	return false, false
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
			return float64(x.AsInt()) < y.AsFloat(), nil
		}
	case FloatType:
		switch y.NumberType() {
		case IntType:
			return x.AsFloat() < float64(y.AsInt()), nil
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

func le(t *Thread, x, y Value) (bool, *Error) {
	switch x.NumberType() {
	case IntType:
		switch y.NumberType() {
		case IntType:
			return x.AsInt() <= y.AsInt(), nil
		case FloatType:
			return float64(x.AsInt()) <= y.AsFloat(), nil
		}
	case FloatType:
		switch y.NumberType() {
		case IntType:
			return x.AsFloat() <= float64(y.AsInt()), nil
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
	res, err, ok = metabin(t, "__lt", y, x)
	if ok {
		return !Truth(res), err
	}
	return false, compareError(x, y)
}

func compareError(x, y Value) *Error {
	return NewErrorF("attempt to compare a '%s' with a '%s'", x.TypeName(), y.TypeName())
}
