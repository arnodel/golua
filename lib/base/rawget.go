package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func rawget(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err.AddContext(c)
	}
	tbl, err := c.TableArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	c.Next().Push(rt.RawGet(tbl, c.Arg(1)))
	return c.Next(), nil
}
