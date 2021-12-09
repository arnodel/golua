package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func errorF(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var (
		err   *rt.Error
		level int64 = 1
	)
	if c.NArgs() == 0 {
		err = rt.NewError(rt.NilValue)
	} else {
		err = rt.NewError(c.Arg(0))
	}
	if c.NArgs() >= 2 {
		var argErr *rt.Error
		level, argErr = c.IntArg(1)
		if argErr != nil {
			return nil, argErr
		}
	}
	if level != 1 {
		err.AddContext(c.Next(), int(level))
	}
	return nil, err
}
