package base

import (
	"bytes"
	"errors"

	rt "github.com/arnodel/golua/runtime"
)

func tonumber(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	nargs := c.NArgs()
	next := c.Next()
	if nargs == 0 {
		return c, errors.New("1 argument required")
	}
	n := c.Arg(0)
	if nargs == 1 {
		n, _ = rt.ToNumber(n)
		next.Push(n)
		return next, nil
	}
	base, ok := c.Arg(1).(rt.Int)
	if !ok {
		return c, errors.New("#2 (base) must be an integer")
	}
	if base < 2 || base > 36 {
		return c, errors.New("#2 (base) out of range")
	}
	s, ok := n.(rt.String)
	if !ok {
		return c, errors.New("#1 must be a string")
	}
	digits := bytes.Trim([]byte(s), " ")
	if len(digits) == 0 {
		return next, nil
	}
	var number rt.Int
	var sign rt.Int = 1
	if digits[0] == '-' {
		sign = -1
		digits = digits[1:]
		if len(digits) == 0 {
			return next, nil
		}
	}
	for _, digit := range digits {
		var d rt.Int
		switch {
		case '0' <= digit && digit <= '9':
			d = rt.Int(digit - '0')
		case 'a' <= digit && digit <= 'z':
			d = rt.Int(digit - 'a' + 10)
		case 'A' <= digit && digit <= 'Z':
			d = rt.Int(digit - 'A' + 10)
		default:
			return next, nil
		}
		if d >= base {
			return next, nil
		}
		number = number*base + d
	}
	next.Push(sign * number)
	return next, nil
}
