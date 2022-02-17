package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func rawequal(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}
	res, _ := rt.RawEqual(c.Arg(0), c.Arg(1))
	return c.PushingNext1(t.Runtime, rt.BoolValue(res)), nil
}
