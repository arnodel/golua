package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func pcall(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	next := c.Next()
	res := rt.NewTerminationWith(0, true)
	if err := rt.Call(t, c.Arg(0), c.Etc(), res); err != nil {
		next.Push(rt.BoolValue(false))
		next.Push(err.Value())
	} else {
		next.Push(rt.BoolValue(true))
		rt.Push(next, res.Etc()...)
	}
	return next, nil
}
