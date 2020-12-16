package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func typeString(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	c.Next().Push(rt.StringValue(rt.Type(c.Arg(0))))
	return c.Next(), nil
}
