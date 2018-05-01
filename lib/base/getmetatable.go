package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func getmetatable(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if c.NArgs() == 0 {
		return nil, rt.NewErrorS("getmetatable expects 1 argument").AddContext(c)
	}
	c.Next().Push(t.Metatable(c.Arg(0)))
	return c.Next(), nil
}
