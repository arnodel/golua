package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func rawlen(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	next := c.Next()
	switch x := c.Arg(0).(type) {
	case rt.String:
		next.Push(rt.Int(len(x)))
		return next, nil
	case *rt.Table:
		next.Push(x.Len())
		return next, nil
	}
	return nil, rt.NewErrorS("#1 must be a string or table").AddContext(c)
}
