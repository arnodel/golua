package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func rawequal(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if c.NArgs() < 2 {
		return nil, rt.NewErrorS("2 arguments required").AddContext(c)
	}
	res, _ := rt.RawEqual(c.Arg(0), c.Arg(1))
	c.Next().Push(rt.Bool(res))
	return c.Next(), nil
}
