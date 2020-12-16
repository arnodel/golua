package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func selectF(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	n, err := c.IntArg(0)
	if err != nil {
		var s string
		s, err = c.StringArg(0)
		if err != nil || s != "#" {
			return nil, rt.NewErrorS("#1 must be an integer or '#'").AddContext(c)
		}
		return c.PushingNext(rt.IntValue(int64(len(c.Etc())))), nil
	}
	etc := c.Etc()
	if n < 0 {
		n += int64(len(etc)) + 1
	}
	if n < 1 {
		return nil, rt.NewErrorS("#1 out of range").AddContext(c)
	}
	next := c.Next()
	if int(n) <= len(etc) {
		rt.Push(next, etc[n-1:]...)
	}
	return c.Next(), nil
}
