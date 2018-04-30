package base

import (
	"errors"

	rt "github.com/arnodel/golua/runtime"
)

func getmetatable(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if c.NArgs() == 0 {
		return c, errors.New("getmetatable expects 1 argument")
	}
	c.Next().Push(t.Metatable(c.Arg(0)))
	return c.Next(), nil
}
