package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func selectF(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	n, err := c.IntArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	if n < 1 {
		return nil, rt.NewErrorS("#1 out of range").AddContext(c)
	}
	if int(n) <= len(c.Etc()) {
		c.Next().Push(c.Etc()[n-1])
	}
	return c.Next(), nil
}
