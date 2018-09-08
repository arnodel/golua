package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func setmetatable(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err.AddContext(c)
	}
	tbl, err := c.TableArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	if rt.RawGet(tbl.Metatable(), "__metatable") != nil {
		return nil, rt.NewErrorS("cannot set metatable").AddContext(c)
	}
	if rt.IsNil(c.Arg(1)) {
		tbl.SetMetatable(nil)
	} else if meta, err := c.TableArg(1); err == nil {
		tbl.SetMetatable(meta)
	} else {
		return nil, err.AddContext(c)
	}
	return c.PushingNext(c.Arg(0)), nil
}
