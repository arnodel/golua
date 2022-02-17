package base

import rt "github.com/arnodel/golua/runtime"

func assert(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	arg := c.Arg(0)
	etc := c.Etc()
	if !rt.Truth(arg) {
		var msg rt.Value
		if len(etc) == 0 {
			msg = rt.StringValue("assertion failed!")
		} else {
			msg = etc[0]
		}
		err := rt.NewError(msg).AddContext(c.Next(), 1)
		return nil, err
	}
	next := c.Next()
	t.Push1(next, arg)
	t.Push(next, etc...)
	return next, nil
}
