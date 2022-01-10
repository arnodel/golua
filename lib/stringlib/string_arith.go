package stringlib

import rt "github.com/arnodel/golua/runtime"

var (
	string__add = stringBinOp(rt.Add, "__add")
)

func stringBinOp(f func(x, y rt.Value) (rt.Value, bool), op string) func(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	return func(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
		if err := c.CheckNArgs(2); err != nil {
			return nil, err
		}
		x, kx := rt.ToNumberValue(c.Arg(0))
		y, ky := rt.ToNumberValue(c.Arg(1))
		if kx != rt.NaN && ky != rt.NaN {
			z, _ := f(x, y)
			return c.PushingNext1(t.Runtime, z), nil
		}
		if y.Type() != rt.StringType {
			next := c.Next()
			err, ok := rt.Metacall(t, y, op, []rt.Value{x, y}, next)
			if ok {
				if err != nil {
					return nil, err
				}
				return next, nil
			}
		}
		return nil, rt.BinaryArithmeticError(op[2:], x, y)
	}
}
