package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func pcall(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	var err error
	if err = c.Check1Arg(); err != nil {
		return nil, err
	}
	next := c.Next()
	res := rt.NewTerminationWith(c, 0, true)
	_, err = t.CallContext(rt.RuntimeContextDef{}, func() error {
		return rt.Call(t, c.Arg(0), c.Etc(), res)
	})
	if err != nil {
		t.Push1(next, rt.BoolValue(false))
		t.Push1(next, rt.ErrorValue(err))
	} else {
		t.Push1(next, rt.BoolValue(true))
		t.Push(next, res.Etc()...)
	}
	return next, nil
}

func xpcall(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	var err error
	if err = c.CheckNArgs(2); err != nil {
		return nil, err
	}
	var msgHandler rt.Callable
	if !c.Arg(1).IsNil() {
		msgHandler, err = c.CallableArg(1)
		if err != nil {
			return nil, err
		}
	}
	next := c.Next()
	res := rt.NewTerminationWith(c, 0, true)

	_, err = t.CallContext(rt.RuntimeContextDef{
		MessageHandler: msgHandler,
	}, func() error {
		return rt.Call(t, c.Arg(0), c.Etc(), res)
	})
	if err != nil {
		t.Push1(next, rt.BoolValue(false))
		t.Push1(next, rt.ErrorValue(err))
	} else {
		t.Push1(next, rt.BoolValue(true))
		t.Push(next, res.Etc()...)
	}
	return next, nil
}
