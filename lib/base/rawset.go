package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func rawset(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(3); err != nil {
		return nil, err.AddContext(c)
	}
	tbl, err := c.TableArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	key := c.Arg(1)
	if rt.IsNil(key) {
		return nil, rt.NewErrorS("#2 must not be nil").AddContext(c)
	}
	if err := tbl.SetCheck(key, c.Arg(2)); err != nil {
		return nil, rt.NewErrorE(err).AddContext(c)
	}
	return c.PushingNext(c.Arg(0)), nil
}
