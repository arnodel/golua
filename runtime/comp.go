package runtime

import "fmt"

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

// isZero returns true if x is a number and is equal to 0.
func isZero(x Value) bool {
	switch x.iface.(type) {
	case int64:
		return x.AsInt() == 0
	case float64:
		return x.AsFloat() == 0
	}
	return false
}

// isPositive returns true if x is a number and is > 0.
func isPositive(x Value) bool {
	switch x.iface.(type) {
	case int64:
		return x.AsInt() > 0
	case float64:
		return x.AsFloat() > 0
	}
	return false
}

func numIsLessThan(x, y Value) bool {
	switch x.iface.(type) {
	case int64:
		switch y.iface.(type) {
		case int64:
			return x.AsInt() < y.AsInt()
		case float64:
			return ltIntAndFloat(x.AsInt(), y.AsFloat())
		}
	case float64:
		switch y.iface.(type) {
		case int64:
			return ltFloatAndInt(x.AsFloat(), y.AsInt())
		case float64:
			return x.AsFloat() < y.AsFloat()
		}
	}
	return false
}

func isLessThan(x, y Value) (bool, bool) {
	switch x.iface.(type) {
	case int64:
		switch y.iface.(type) {
		case int64:
			return x.AsInt() < y.AsInt(), true
		case float64:
			return ltIntAndFloat(x.AsInt(), y.AsFloat()), true
		}
	case float64:
		switch y.iface.(type) {
		case int64:
			return ltFloatAndInt(x.AsFloat(), y.AsInt()), true
		case float64:
			return x.AsFloat() < y.AsFloat(), true
		}
	}
	return false, false
}

func equalIntAndFloat(n int64, f float64) bool {
	nf := int64(f)
	return float64(nf) == f && nf == n
}

func eq(t *Thread, x, y Value) (bool, error) {
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
func Lt(t *Thread, x, y Value) (bool, error) {
	lt, ok := isLessThan(x, y)
	if ok {
		return lt, nil
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

func le(t *Thread, x, y Value) (bool, error) {
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

func compareError(x, y Value) error {
	return fmt.Errorf("attempt to compare a %s value with a %s value", x.CustomTypeName(), y.CustomTypeName())
}
