package base

import (
	"errors"

	rt "github.com/arnodel/golua/runtime"
)

func errorF(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	next := c.Next()
	switch nargs := c.NArgs(); {
	case nargs >= 2:
		level, tp := rt.ToInt(c.Arg(1))
		if tp != rt.IsInt || level < 1 {
			return c, errors.New("level must be an integer > 0")
		}
		for level > 1 && next != nil {
			next = next.Next()
			level--
		}
		fallthrough
	case nargs >= 1:
		return next, rt.ErrorFromValue(c.Arg(0))
	default:
		return c, errors.New("1 argument required")
	}
}
