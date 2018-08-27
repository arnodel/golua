package base

import rt "github.com/arnodel/golua/runtime"

func assert(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	arg := c.Arg(0)
	if !rt.Truth(arg) {
		var msg rt.Value
		if c.NArgs() < 2 {
			msg = rt.String("assertion failed!")
		} else {
			msg = c.Arg(1)
		}
		return nil, rt.NewError(msg).AddContext(c)
	}
	return c.PushingNext(arg), nil
}
