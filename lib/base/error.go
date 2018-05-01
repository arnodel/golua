package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func errorF(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	next := c.Next()
	switch nargs := c.NArgs(); {
	case nargs >= 2:
		level, tp := rt.ToInt(c.Arg(1))
		if tp != rt.IsInt || level < 1 {
			return nil, rt.NewErrorS("level must be an integer > 0").AddContext(c)
		}
		for level > 1 && next != nil {
			next = next.Next()
			level--
		}
		fallthrough
	case nargs >= 1:
		return nil, rt.NewError(c.Arg(0)).AddContext(c)
	default:
		return nil, rt.NewErrorS("1 argument required").AddContext(c)
	}
}
