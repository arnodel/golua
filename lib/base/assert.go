package base

import rt "github.com/arnodel/golua/runtime"

func assert(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	arg := c.Arg(0)
	etc := c.Etc()
	if !rt.Truth(arg) {
		var msg rt.Value
		if len(etc) == 0 {
			msg = rt.String("assertion failed!")
		} else {
			msg = etc[0]
		}
		return nil, rt.NewError(msg).AddContext(c)
	}
	next := c.Next()
	next.Push(arg)
	rt.Push(next, etc...)
	return next, nil
}
