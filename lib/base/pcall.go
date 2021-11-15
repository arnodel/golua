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
		t.Push1(next, rt.BoolValue(false))
		t.Push1(next, err.Value())
	} else {
		t.Push1(next, rt.BoolValue(true))
		t.Push(next, res.Etc()...)
	}
	return next, nil
}
