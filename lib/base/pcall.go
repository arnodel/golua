package base

import (
	"errors"

	rt "github.com/arnodel/golua/runtime"
)

func pcall(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if c.NArgs() == 0 {
		return c, errors.New("1 argument required")
	}
	next := c.Next()
	res := rt.NewTerminationWith(0, true)
	if err := rt.Call(t, c.Arg(0), c.Etc(), res); err != nil {
		next.Push(rt.Bool(false))
		next.Push(rt.ValueFromError(err))
	} else {
		next.Push(rt.Bool(true))
		rt.Push(next, res.Etc()...)
	}
	return next, nil
}
