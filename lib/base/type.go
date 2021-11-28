package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func typeString(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	return c.PushingNext1(t.Runtime, rt.StringValue(rt.Type(c.Arg(0)))), nil
}
