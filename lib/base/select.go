package base

import (
	"errors"

	rt "github.com/arnodel/golua/runtime"
)

func selectF(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if c.NArgs() == 0 {
		return c, errors.New("1 argument required")
	}
	n, tp := rt.ToInt(c.Arg(0))
	if tp != rt.IsInt {
		return c, errors.New("#1 must be an integer")
	}
	if n < 1 {
		return c, errors.New("#1 out of range")
	}
	if int(n) <= len(c.Etc()) {
		c.Next().Push(c.Etc()[n-1])
	}
	return c.Next(), nil
}
