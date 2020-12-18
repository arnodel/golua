package base

import (
	"bytes"

	rt "github.com/arnodel/golua/runtime"
)

func tonumber(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	nargs := c.NArgs()
	next := c.Next()
	n := c.Arg(0)
	if nargs == 1 {
		n, tp := rt.ToNumberValue(n)
		if tp != rt.NaN {
			next.Push(n)
		} else {
			next.Push(rt.NilValue)
		}
		return next, nil
	}
	base, err := c.IntArg(1)
	if err != nil {
		return nil, err.AddContext(c)
	}
	if base < 2 || base > 36 {
		return nil, rt.NewErrorS("#2 out of range").AddContext(c)
	}
	s, ok := n.TryString()
	if !ok {
		return nil, rt.NewErrorS("#1 must be a string").AddContext(c)
	}
	digits := bytes.Trim([]byte(s), " ")
	if len(digits) == 0 {
		return next, nil
	}
	var number int64
	var sign int64 = 1
	if digits[0] == '-' {
		sign = -1
		digits = digits[1:]
		if len(digits) == 0 {
			return next, nil
		}
	}
	for _, digit := range digits {
		var d int64
		switch {
		case '0' <= digit && digit <= '9':
			d = int64(digit - '0')
		case 'a' <= digit && digit <= 'z':
			d = int64(digit - 'a' + 10)
		case 'A' <= digit && digit <= 'Z':
			d = int64(digit - 'A' + 10)
		default:
			return next, nil
		}
		if d >= base {
			return next, nil
		}
		number = number*base + d
	}
	next.Push(rt.IntValue(sign * number))
	return next, nil
}
