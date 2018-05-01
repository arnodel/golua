package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func setmetatable(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if c.NArgs() < 2 {
		return nil, rt.NewErrorS("2 arguments required").AddContext(c)
	}
	tbl, ok := c.Arg(0).(*rt.Table)
	if !ok {
		return nil, rt.NewErrorS("#1 must be a table").AddContext(c)
	}
	if rt.RawGet(tbl.Metatable(), "__metatable") != nil {
		return nil, rt.NewErrorS("cannot set metatable").AddContext(c)
	}
	if rt.IsNil(c.Arg(1)) {
		tbl.SetMetatable(nil)
	} else if meta, ok := c.Arg(1).(*rt.Table); ok {
		tbl.SetMetatable(meta)
	} else {
		return nil, rt.NewErrorS("#2 must be a table").AddContext(c)
	}
	c.Next().Push(c.Arg(1))
	return c.Next(), nil
}
