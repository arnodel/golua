package base

import rt "github.com/arnodel/golua/runtime"

func ipairsIteratorF(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if c.NArgs() < 2 {
		return nil, rt.NewErrorS("2 arguments required").AddContext(c)
	}
	coll := c.Arg(0)
	n, tp := rt.ToInt(c.Arg(1))
	if tp != rt.IsInt {
		return nil, rt.NewErrorS("#2 must be an integer").AddContext(c)
	}
	lv, err := rt.Len(t, coll)
	if err != nil {
		return nil, err.AddContext(c)
	}
	li, tp := rt.ToInt(lv)
	if tp != rt.IsInt {
		return nil, rt.NewErrorS("length of #1 not an integer").AddContext(c)
	}
	next := c.Next()
	if n < li {
		n += 1
		v, err := rt.Index(t, c.Arg(0), n)
		if err != nil {
			return nil, err.AddContext(c)
		}
		next.Push(n)
		next.Push(v)
	}
	return next, nil
}

var ipairsIterator = rt.NewGoFunction(ipairsIteratorF, 2, false)

func ipairs(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if c.NArgs() == 0 {
		return nil, rt.NewErrorS("1 argument required").AddContext(c)
	}
	next := c.Next()
	next.Push(ipairsIterator)
	next.Push(c.Arg(0))
	next.Push(rt.Int(0))
	return next, nil
}
