package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func rawget(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if c.NArgs() < 2 {
		return nil, rt.NewErrorS("2 arguments required").AddContext(c)
	}
	tbl, ok := c.Arg(0).(*rt.Table)
	if !ok {
		return nil, rt.NewErrorS("#1 must be a table").AddContext(c)
	}
	c.Next().Push(rt.RawGet(tbl, c.Arg(1)))
	return c.Next(), nil
}
