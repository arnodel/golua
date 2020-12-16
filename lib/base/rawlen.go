package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func rawlen(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	next := c.Next()
	switch x := c.Arg(0); x.Type() {
	case rt.StringType:
		next.Push(rt.IntValue(int64(len(x.AsString()))))
		return next, nil
	case rt.TableType:
		next.Push(rt.IntValue(x.AsTable().Len()))
		return next, nil
	}
	return nil, rt.NewErrorS("#1 must be a string or table").AddContext(c)
}
