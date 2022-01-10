package stringlib

import rt "github.com/arnodel/golua/runtime"

func string__add(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	x, y, err := get2numbers(c)
	if err != nil {
		return nil, err
	}
	z, _ := rt.Add(x, y)
	return c.PushingNext1(t.Runtime, z), nil
}

func get2numbers(c *rt.GoCont) (rt.Value, rt.Value, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return rt.NilValue, rt.NilValue, err
	}
	x, kx := rt.ToNumberValue(c.Arg(0))
	y, ky := rt.ToNumberValue(c.Arg(1))
	if kx == rt.NaN || ky == rt.NaN {
		return rt.NilValue, rt.NilValue, rt.BinaryArithmeticError("add", c.Arg(0), c.Arg(1), kx, ky)
	}
	return x, y, nil
}
