package mathlib

import (
	"math"

	rt "github.com/arnodel/golua/runtime"
)

func Load(r *rt.Runtime) {
	pkg := rt.NewTable()
	rt.SetEnv(r.GlobalEnv(), "math", pkg)
	rt.SetEnvGoFunc(pkg, "abs", abs, 1, false)
	rt.SetEnvGoFunc(pkg, "acos", acos, 1, false)
	rt.SetEnvGoFunc(pkg, "asin", asin, 1, false)
	rt.SetEnvGoFunc(pkg, "atan", atan, 2, false)
	rt.SetEnvGoFunc(pkg, "ceil", ceil, 1, false)
	rt.SetEnvGoFunc(pkg, "cos", cos, 1, false)
	rt.SetEnvGoFunc(pkg, "deg", deg, 1, false)
	rt.SetEnvGoFunc(pkg, "exp", exp, 1, false)
	rt.SetEnvGoFunc(pkg, "floor", floor, 1, false)
	rt.SetEnvGoFunc(pkg, "fmod", fmod, 2, false)
	rt.SetEnv(pkg, "huge", rt.Float(math.Inf(1)))
	rt.SetEnvGoFunc(pkg, "log", log, 2, false)

}

func abs(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	next := c.Next()
	x, tp := rt.ToNumber(c.Arg(0))
	switch tp {
	case rt.IsInt:
		n := x.(rt.Int)
		if n < 0 {
			n = -n
		}
		next.Push(n)
	case rt.IsFloat:
		f := rt.Float(math.Abs(float64(x.(rt.Float))))
		next.Push(f)
	default:
		return nil, rt.NewErrorS("#1 must be a number").AddContext(c)
	}
	return next, nil
}

func acos(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	y := rt.Float(math.Acos(float64(x)))
	return c.PushingNext(y), nil
}

func asin(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	y := rt.Float(math.Asin(float64(x)))
	return c.PushingNext(y), nil
}

func atan(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	y, err := c.FloatArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	var x rt.Float = 1
	if c.NArgs() >= 2 {
		y, err = c.FloatArg(1)
		if err != nil {
			return nil, err.AddContext(c)
		}
	}
	z := rt.Float(math.Atan2(float64(y), float64(x)))
	return c.PushingNext(z), nil
}

func ceil(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	next := c.Next()
	x, tp := rt.ToNumber(c.Arg(0))
	switch tp {
	case rt.IsInt:
		next.Push(x)
	case rt.IsFloat:
		y := rt.Float(math.Ceil(float64(x.(rt.Float))))
		next.Push(y)
	default:
		return nil, rt.NewErrorS("#1 must be a number").AddContext(c)
	}
	return next, nil
}

func cos(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	y := rt.Float(math.Cos(float64(x)))
	return c.PushingNext(y), nil
}

func deg(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	y := x * 180 / math.Pi
	return c.PushingNext(y), nil
}

func exp(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	y := rt.Float(math.Exp(float64(x)))
	return c.PushingNext(y), nil
}

func floor(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	next := c.Next()
	x, tp := rt.ToNumber(c.Arg(0))
	switch tp {
	case rt.IsInt:
		next.Push(x)
	case rt.IsFloat:
		y := rt.Float(math.Floor(float64(x.(rt.Float))))
		next.Push(y)
	default:
		return nil, rt.NewErrorS("#1 must be a number").AddContext(c)
	}
	return next, nil
}

func fmod(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err.AddContext(c)
	}
	res, err := rt.Mod(t, c.Arg(0), c.Arg(1))
	if err != nil {
		return nil, err.AddContext(c)
	}
	return c.PushingNext(res), nil
}

func log(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	y := math.Log(float64(x))
	if c.NArgs() >= 2 {
		b, err := c.FloatArg(1)
		if err != nil {
			return nil, err.AddContext(c)
		}
		y = y / math.Log(float64(b))
	}
	return c.PushingNext(rt.Float(y)), nil
}
