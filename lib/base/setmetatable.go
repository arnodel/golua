package base

import (
	"errors"

	rt "github.com/arnodel/golua/runtime"
)

func setmetatable(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if c.NArgs() < 2 {
		return c, errors.New("2 arguments required")
	}
	tbl, ok := c.Arg(0).(*rt.Table)
	if !ok {
		return c, errors.New("#1 must be a table")
	}
	if rt.RawGet(tbl.Metatable(), "__metatable") != nil {
		return c, errors.New("cannot set metatable")
	}
	if rt.IsNil(c.Arg(1)) {
		tbl.SetMetatable(nil)
	} else if meta, ok := c.Arg(1).(*rt.Table); ok {
		tbl.SetMetatable(meta)
	} else {
		return c, errors.New("#2 must be a table")
	}
	c.Next().Push(c.Arg(1))
	return c.Next(), nil
}
