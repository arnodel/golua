package base

import (
	"errors"

	rt "github.com/arnodel/golua/runtime"
)

func rawset(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.CheckNArgs(3); err != nil {
		return nil, err
	}
	tbl, err := c.TableArg(0)
	if err != nil {
		return nil, err
	}
	key := c.Arg(1)
	if key.IsNil() {
		return nil, errors.New("#2 must not be nil")
	}
	if err := t.SetTableCheck(tbl, key, c.Arg(2)); err != nil {
		return nil, err
	}
	return c.PushingNext1(t.Runtime, c.Arg(0)), nil
}
