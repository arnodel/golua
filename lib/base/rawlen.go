package base

import (
	"errors"

	rt "github.com/arnodel/golua/runtime"
)

func rawlen(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if c.NArgs() == 0 {
		return c, errors.New("1 argument required")
	}
	next := c.Next()
	switch x := c.Arg(0).(type) {
	case rt.String:
		next.Push(rt.Int(len(x)))
		return next, nil
	case *rt.Table:
		next.Push(x.Len())
		return next, nil
	}
	return c, errors.New("#1 must be a string or table")
}
