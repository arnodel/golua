package stringlib

import rt "github.com/arnodel/golua/runtime"

var (
	string__add  = stringBinOp(rt.Add, "__add")
	string__sub  = stringBinOp(rt.Sub, "__sub")
	string__mul  = stringBinOp(rt.Mul, "__mul")
	string__div  = stringBinOp(rt.Div, "__div")
	string__idiv = stringBinOpErr(rt.Idiv, "__idiv")
	string__mod  = stringBinOpErr(rt.Mod, "__mod")
	string__pow  = stringBinOp(rt.Pow, "__pow")
	string__unm  = stringUnOp(rt.Unm, "__unm")
)

func stringBinOp(f func(x, y rt.Value) (rt.Value, bool), op string) func(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	return func(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
		if err := c.CheckNArgs(2); err != nil {
			return nil, err
		}
		x, y := c.Arg(0), c.Arg(1)
		nx, kx := rt.ToNumberValue(x)
		ny, ky := rt.ToNumberValue(y)
		if kx != rt.NaN && ky != rt.NaN {
			z, _ := f(nx, ny)
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
		return nil, rt.BinaryArithmeticError(op[2:], nx, ny)
	}
}

func stringBinOpErr(f func(x, y rt.Value) (rt.Value, bool, *rt.Error), op string) func(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	return func(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
		if err := c.CheckNArgs(2); err != nil {
			return nil, err
		}
		x, y := c.Arg(0), c.Arg(1)
		nx, kx := rt.ToNumberValue(x)
		ny, ky := rt.ToNumberValue(y)
		if kx != rt.NaN && ky != rt.NaN {
			z, _, err := f(nx, ny)
			if err != nil {
				return nil, err
			}
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
		return nil, rt.BinaryArithmeticError(op[2:], nx, ny)
	}
}

func stringUnOp(f func(x rt.Value) (rt.Value, bool), op string) func(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	return func(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
		if err := c.Check1Arg(); err != nil {
			return nil, err
		}
		x := c.Arg(0)
		nx, _ := rt.ToNumberValue(x)
		z, ok := f(nx)
		if ok {
			return c.PushingNext1(t.Runtime, z), nil
		}
		return nil, rt.UnaryArithmeticError(op[2:], nx)
	}
}
