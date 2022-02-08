package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func setmetatable(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}
	tbl, err := c.TableArg(0)
	if err != nil {
		return nil, err
	}
	if !rt.RawGet(tbl.Metatable(), rt.StringValue("__metatable")).IsNil() {
		return nil, rt.NewErrorS("cannot set metatable")
	}
	if c.Arg(1).IsNil() {
		tbl.SetMetatable(nil)
	} else if meta, err := c.TableArg(1); err == nil {
		tbl.SetMetatable(meta)
	} else {
		return nil, err
	}
	return c.PushingNext1(t.Runtime, c.Arg(0)), nil
}
