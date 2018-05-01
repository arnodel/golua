package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func rawset(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if c.NArgs() < 3 {
		return nil, rt.NewErrorS("3 arguments required").AddContext(c)
	}
	tbl, ok := c.Arg(0).(*rt.Table)
	if !ok {
		return nil, rt.NewErrorS("#1 must be a table").AddContext(c)
	}
	key := c.Arg(1)
	if rt.IsNil(key) {
		return nil, rt.NewErrorS("#2 must not be nil").AddContext(c)
	}
	tbl.Set(key, c.Arg(2))
	c.Next().Push(c.Arg(0))
	return c.Next(), nil
}
