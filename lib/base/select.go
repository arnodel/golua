package base

import (
	"errors"

	rt "github.com/arnodel/golua/runtime"
)

func selectF(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	n, err := c.IntArg(0)
	if err != nil {
		var s string
		s, err = c.StringArg(0)
		if err != nil || s != "#" {
			return nil, errors.New("#1 must be an integer or '#'")
		}
		return c.PushingNext1(t.Runtime, rt.IntValue(int64(len(c.Etc())))), nil
	}
	etc := c.Etc()
	if n < 0 {
		n += int64(len(etc)) + 1
	}
	if n < 1 {
		return nil, errors.New("#1 out of range")
	}
	next := c.Next()
	if int(n) <= len(etc) {
		t.Push(next, etc[n-1:]...)
	}
	return next, nil
}
