package base

import rt "github.com/arnodel/golua/runtime"

func next(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var k rt.Value
	switch nargs := c.NArgs(); {
	case nargs >= 2:
		k = c.Arg(1)
		fallthrough
	case nargs >= 1:
		tbl, ok := c.Arg(0).(*rt.Table)
		if !ok {
			return nil, rt.NewErrorS("#1 must be a table").AddContext(c)
		}
		next := c.Next()
		nk, nv := tbl.Next(k)
		next.Push(nk)
		next.Push(nv)
		return next, nil
	default:
		return nil, rt.NewErrorS("1 argument required").AddContext(c)
	}
}

var nextGoFunc = rt.NewGoFunction(next, 2, false)
