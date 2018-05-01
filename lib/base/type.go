package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func typeString(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if c.NArgs() == 0 {
		return nil, rt.NewErrorS("type needs 1 argument").AddContext(c)
	}
	c.Next().Push(rt.Type(c.Arg(0)))
	return c.Next(), nil
}
