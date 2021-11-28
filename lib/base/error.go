package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func errorF(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	next := c.Next()
	if c.NArgs() >= 2 {
		level, err := c.IntArg(1)
		if err != nil {
			return nil, err
		}
		if level < 1 {
			return nil, rt.NewErrorS("#2 must be > 0")
		}
		for level > 1 && next != nil {
			next = next.Next()
			level--
		}
	}
	return nil, rt.NewError(c.Arg(0))
}
