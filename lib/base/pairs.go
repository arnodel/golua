package base

import rt "github.com/arnodel/golua/runtime"

func pairs(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	coll := c.Arg(0)
	next := c.Next()
	if pairsMeta := rt.RawGet(t.RawMetatable(coll), pairsMetaName); !pairsMeta.IsNil() {
		// The metamethod returns where pairs would: this is a tail call, so
		// no Go code waits for it and it can yield.
		metaCont, err := rt.Continue(t, pairsMeta, next)
		if err != nil {
			return nil, err
		}
		t.Push1(metaCont, coll)
		return metaCont, nil
	}
	t.Push(next, rt.FunctionValue(nextGoFunc), coll, rt.NilValue)
	return next, nil
}

var pairsMetaName = rt.StringValue("__pairs")
