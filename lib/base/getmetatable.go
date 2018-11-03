package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func getmetatable(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	return c.PushingNext(t.Metatable(c.Arg(0))), nil
}
