package base

import rt "github.com/arnodel/golua/runtime"

func next(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	var k rt.Value
	if c.NArgs() >= 2 {
		k = c.Arg(1)
	}
	tbl, err := c.TableArg(0)
	if err != nil {
		return nil, err
	}
	next := c.Next()
	nk, nv, ok := tbl.Next(k)
	if !ok {
		return nil, rt.NewErrorS("invalid key for 'next'")
	}
	t.Push1(next, nk)
	t.Push1(next, nv)
	return next, nil
}

var nextGoFunc = rt.NewGoFunction(next, "next", 2, false)
