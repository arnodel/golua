package base

import (
	"errors"

	rt "github.com/arnodel/golua/runtime"
)

func rawequal(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if c.NArgs() < 2 {
		return c, errors.New("2 arguments required")
	}
	res, _ := rt.RawEqual(c.Arg(0), c.Arg(1))
	c.Next().Push(rt.Bool(res))
	return c.Next(), nil
}
