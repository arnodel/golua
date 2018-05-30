package base

import rt "github.com/arnodel/golua/runtime"

func ipairsIteratorF(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err.AddContext(c)
	}
	coll := c.Arg(0)
	n, err := c.IntArg(1)
	if err != nil {
		return nil, err.AddContext(c)
	}
	l, err := rt.Len(t, coll)
	if err != nil {
		return nil, err.AddContext(c)
	}
	next := c.Next()
	if n < l {
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

var ipairsIterator = rt.NewGoFunction(ipairsIteratorF, "ipairsiterator", 2, false)

func ipairs(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	next := c.Next()
	next.Push(ipairsIterator)
	next.Push(c.Arg(0))
	next.Push(rt.Int(0))
	return next, nil
}
