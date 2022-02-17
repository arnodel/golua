package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func typeString(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	return c.PushingNext1(t.Runtime, rt.StringValue(c.Arg(0).TypeName())), nil
}
