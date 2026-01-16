package base

import (
	rt "github.com/arnodel/golua/runtime"
)

var noErrorObject = rt.StringValue("<no error object>")

func errorF(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	var (
		err    *rt.Error
		level  int64 = 1
		errObj rt.Value
	)
	if c.NArgs() > 0 {
		errObj = c.Arg(0)
	}
	if c.NArgs() >= 2 {
		var argErr error
		level, argErr = c.IntArg(1)
		if argErr != nil {
			return nil, argErr
		}
	}
	// Lua 5.5: if error object is nil, replace with message and force level 0
	if errObj.IsNil() {
		errObj = noErrorObject
		level = 0
	}
	err = rt.NewError(errObj)
	if level != 1 {
		err = err.AddContext(c.Next(), int(level))
	}
	return nil, err
}
