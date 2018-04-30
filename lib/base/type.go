package base

import (
	"errors"

	rt "github.com/arnodel/golua/runtime"
)

func typeString(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if c.NArgs() == 0 {
		return c, errors.New("type needs 1 argument")
	}
	c.Next().Push(rt.Type(c.Arg(0)))
	return c.Next(), nil
}
