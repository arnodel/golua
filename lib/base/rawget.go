package base

import (
	"errors"

	rt "github.com/arnodel/golua/runtime"
)

func rawget(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if c.NArgs() < 2 {
		return c, errors.New("2 arguments required")
	}
	tbl, ok := c.Arg(0).(*rt.Table)
	if !ok {
		return c, errors.New("#1 must be a table")
	}
	c.Next().Push(rt.RawGet(tbl, c.Arg(1)))
	return c.Next(), nil
}
