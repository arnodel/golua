package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func loadfile(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	chunk, chunkName, err := loadChunk(t, c.Args())
	defer t.ReleaseBytes(len(chunk))
	if err != nil {
		return t.ProcessIoError(c.Next(), err)
	}
	var (
		next      = c.Next()
		chunkMode = "bt"
		chunkEnv  = rt.TableValue(t.GlobalEnv())
	)
	switch nargs := c.NArgs(); {
	case nargs >= 3:
		chunkEnv = c.Arg(2)
		fallthrough
	case nargs >= 2:
		mode, err := c.StringArg(1)
		if err != nil {
			return nil, err
		}
		chunkMode = string(mode)
	}
	clos, err := t.LoadFromSourceOrCode(chunkName, chunk, chunkMode, chunkEnv, true)
	if err != nil {
		t.Push(next, rt.NilValue, rt.StringValue(err.Error()))
	} else {
		t.Push1(next, rt.FunctionValue(clos))
	}
	return next, nil
}
