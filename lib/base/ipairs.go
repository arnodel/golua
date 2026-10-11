package base

import rt "github.com/arnodel/golua/runtime"

func ipairsIteratorF(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}
	coll := c.Arg(0)
	n, err := c.IntArg(1)
	if err != nil {
		return nil, err
	}
	next := c.Next()
	n++
	nv := rt.IntValue(n)
	v, err := rt.Index(t, coll, nv)
	if err != nil {
		return nil, err
	}
	if !v.IsNil() {
		t.Push1(next, nv)
		t.Push1(next, v)
	}
	return next, nil
}

var ipairsIterator = rt.NewGoFunction(ipairsIteratorF, "ipairsiterator", 2, false)

// ipairsIterator and nextGoFunc are shared by every runtime, so their
// compliance is declared once here rather than in Load: declaring it in Load
// writes to them each time a runtime is created, which is a data race when
// runtimes are created concurrently or while another runtime iterates.
func init() {
	rt.SolemnlyDeclareCompliance(
		rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyTimeSafe|rt.ComplyIoSafe,
		ipairsIterator,
		nextGoFunc,
	)
}

func ipairs(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	next := c.Next()
	t.Push1(next, rt.FunctionValue(ipairsIterator))
	t.Push1(next, c.Arg(0))
	t.Push1(next, rt.IntValue(0))
	return next, nil
}
