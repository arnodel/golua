package base

import rt "github.com/arnodel/golua/runtime"

func assert(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if c.NArgs() == 0 {
		return nil, rt.NewErrorS("1 argument required").AddContext(c)
	}
	if !rt.Truth(c.Arg(0)) {
		var msg rt.Value
		if c.NArgs() < 2 {
			msg = rt.String("assertion failed!")
		} else {
			msg = c.Arg(1)
		}
		return nil, rt.NewError(msg).AddContext(c)
	}
	return c.Next(), nil
}
