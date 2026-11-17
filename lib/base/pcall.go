package base

import (
	rt "github.com/arnodel/golua/runtime"
)

// pcall and xpcall hand over to the function they call: no Go code waits for
// it, so it can yield (see rt.Thread.ProtectedCall).

func pcall(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	return t.ProtectedCall("pcall", c.Arg(0), c.Etc(), nil, c.Next()), nil
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
	return t.ProtectedCall("xpcall", c.Arg(0), c.Etc(), msgHandler, c.Next()), nil
}
