package base

import (
	"errors"

	rt "github.com/arnodel/golua/runtime"
)

func rawset(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if c.NArgs() < 3 {
		return c, errors.New("3 arguments required")
	}
	tbl, ok := c.Arg(0).(*rt.Table)
	if !ok {
		return c, errors.New("#1 must be a table")
	}
	key := c.Arg(1)
	if rt.IsNil(key) {
		return c, errors.New("#2 must not be nil")
	}
	tbl.Set(key, c.Arg(2))
	c.Next().Push(c.Arg(0))
	return c.Next(), nil
}
