package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func selectF(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if c.NArgs() == 0 {
		return nil, rt.NewErrorS("1 argument required").AddContext(c)
	}
	n, tp := rt.ToInt(c.Arg(0))
	if tp != rt.IsInt {
		return nil, rt.NewErrorS("#1 must be an integer").AddContext(c)
	}
	if n < 1 {
		return nil, rt.NewErrorS("#1 out of range").AddContext(c)
	}
	if int(n) <= len(c.Etc()) {
		c.Next().Push(c.Etc()[n-1])
	}
	return c.Next(), nil
}
