package base

import rt "github.com/arnodel/golua/runtime"

func pairs(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if c.NArgs() == 0 {
		return nil, rt.NewErrorS("1 argument required").AddContext(c)
	}
	coll := c.Arg(0)
	next := c.Next()
	err, ok := rt.Metacall(t, coll, "__pairs", []rt.Value{coll}, next)
	if ok {
		if err != nil {
			return nil, err.AddContext(c)
		}
		return next, nil
	}
	next.Push(nextGoFunc)
	next.Push(coll)
	next.Push(nil)
	return next, nil
}
