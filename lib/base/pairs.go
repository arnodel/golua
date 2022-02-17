package base

import rt "github.com/arnodel/golua/runtime"

func pairs(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	coll := c.Arg(0)
	next := c.Next()
	res := rt.NewTerminationWith(c, 0, true)
	err, ok := rt.Metacall(t, coll, "__pairs", []rt.Value{coll}, res)
	if ok {
		if err != nil {
			return nil, err
		}
		t.Push(next, res.Etc()...)
		return next, nil
	}
	t.Push(next, rt.FunctionValue(nextGoFunc), coll, rt.NilValue)
	return next, nil
}
