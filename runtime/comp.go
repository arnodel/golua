package runtime

func RawEqual(x, y Value) (bool, bool) {
	if x == y {
		return true, true
	}
	switch xx := x.(type) {
	case Int:
		if yy, ok := y.(Float); ok {
			return Float(xx) == yy, true
		}
	case Float:
		if yy, ok := y.(Int); ok {
			return xx == Float(yy), true
		}
	}
	return false, false
}

func eq(t *Thread, x, y Value) (bool, *Error) {
	if res, ok := RawEqual(x, y); ok {
		return res, nil
	}
	switch x.(type) {
	case *Table:
		if _, ok := y.(*Table); !ok {
			return false, nil
		}
	// case *UserData:
	//     deal with that!
	default:
		return false, nil
	}
	res, err, ok := metabin(t, "__eq", x, y)
	if ok {
		return truth(res), err
	}
	return false, NewErrorS("eq expects eqable values")
}

func lt(t *Thread, x, y Value) (bool, *Error) {
	switch xx := x.(type) {
	case Int:
		switch yy := y.(type) {
		case Int:
			return xx < yy, nil
		case Float:
			return Float(xx) < yy, nil
		}
	case Float:
		switch yy := y.(type) {
		case Int:
			return xx < Float(yy), nil
		case Float:
			return xx < yy, nil
		}
	case String:
		if yy, ok := y.(String); ok {
			return xx < yy, nil
		}
	}
	res, err, ok := metabin(t, "__lt", x, y)
	if ok {
		return truth(res), err
	}
	return false, NewErrorS("lt expects ltable values")
}

func le(t *Thread, x, y Value) (bool, *Error) {
	switch xx := x.(type) {
	case Int:
		switch yy := y.(type) {
		case Int:
			return xx <= yy, nil
		case Float:
			return Float(xx) <= yy, nil
		}
	case Float:
		switch yy := y.(type) {
		case Int:
			return xx <= Float(yy), nil
		case Float:
			return xx <= yy, nil
		}
	case String:
		if yy, ok := y.(String); ok {
			return xx <= yy, nil
		}
	}
	res, err, ok := metabin(t, "__le", x, y)
	if ok {
		return truth(res), err
	}
	res, err, ok = metabin(t, "__lt", y, x)
	if ok {
		return !truth(res), err
	}
	return false, NewErrorS("le expects leable values")
}
