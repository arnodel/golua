package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func rawequal(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err.AddContext(c)
	}
	res, _ := rt.RawEqual(c.Arg(0), c.Arg(1))
	c.Next().Push(rt.Bool(res))
	return c.Next(), nil
}
