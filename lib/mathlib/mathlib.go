package mathlib

import (
	"math"
	"math/rand"

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
	rt.SetEnvGoFunc(pkg, "max", max, 1, true)
	rt.SetEnv(pkg, "maxinteger", rt.Int(math.MaxInt64))
	rt.SetEnvGoFunc(pkg, "min", min, 1, true)
	rt.SetEnv(pkg, "mininteger", rt.Int(math.MinInt64))
	rt.SetEnvGoFunc(pkg, "modf", modf, 1, false)
	rt.SetEnv(pkg, "pi", rt.Float(math.Pi))
	rt.SetEnvGoFunc(pkg, "rad", rad, 1, false)
	rt.SetEnvGoFunc(pkg, "random", random, 2, false)
	rt.SetEnvGoFunc(pkg, "randomseed", randomseed, 1, false)
	rt.SetEnvGoFunc(pkg, "sin", sin, 1, false)
	rt.SetEnvGoFunc(pkg, "sqrt", sqrt, 1, false)
	rt.SetEnvGoFunc(pkg, "tan", tan, 1, false)
	rt.SetEnvGoFunc(pkg, "tointeger", tointeger, 1, false)
	rt.SetEnvGoFunc(pkg, "type", typef, 1, false)
	rt.SetEnvGoFunc(pkg, "ult", ult, 2, false)
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

func max(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	x := c.Arg(0)
	for _, y := range c.Etc() {
		lt, err := rt.Lt(t, x, y)
		if err != nil {
			return nil, err.AddContext(c)
		}
		if lt {
			x = y
		}
	}
	return c.PushingNext(x), nil
}

func min(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	x := c.Arg(0)
	for _, y := range c.Etc() {
		lt, err := rt.Lt(t, y, x)
		if err != nil {
			return nil, err.AddContext(c)
		}
		if lt {
			x = y
		}
	}
	return c.PushingNext(x), nil
}

func modf(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	i, f := math.Modf(float64(x))
	next := c.Next()
	next.Push(rt.Float(i))
	next.Push(rt.Float(f))
	return next, nil
}

func rad(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	y := x * math.Pi / 180
	return c.PushingNext(y), nil
}

// TODO: have a per runtime random generator
func random(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	var err *rt.Error
	var m rt.Int = 1
	var n rt.Int
	if c.NArgs() >= 2 {
		m, err = c.IntArg(0)
		if err == nil {
			n, err = c.IntArg(1)
		}
	} else {
		n, err = c.IntArg(0)
	}
	if err != nil {
		return nil, err.AddContext(c)
	}
	if m > n {
		return nil, rt.NewErrorS("#2 must be >= #1").AddContext(c)
	}
	r := m + rt.Int(rand.Intn(int(n-m)))
	return c.PushingNext(r), nil
}

func randomseed(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	seed, err := c.IntArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	rand.Seed(int64(seed))
	return c.Next(), nil
}

func sin(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	y := rt.Float(math.Sin(float64(x)))
	return c.PushingNext(y), nil
}

func sqrt(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	y := rt.Float(math.Sqrt(float64(x)))
	return c.PushingNext(y), nil
}

func tan(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	y := rt.Float(math.Tan(float64(x)))
	return c.PushingNext(y), nil
}

func tointeger(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	n, err := c.IntArg(0)
	if err != nil {
		return c.PushingNext(nil), nil
	}
	return c.PushingNext(n), nil
}

func typef(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	var tp rt.Value
	switch c.Arg(0).(type) {
	case rt.Int:
		tp = rt.String("integer")
	case rt.Float:
		tp = rt.String("float")
	}
	return c.PushingNext(tp), nil
}

func ult(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err.AddContext(c)
	}
	x, err := c.IntArg(0)
	var y rt.Int
	if err == nil {
		y, err = c.IntArg(1)
	}
	if err != nil {
		return nil, err.AddContext(c)
	}
	lt := rt.Bool(uint64(x) < uint64(y))
	return c.PushingNext(lt), nil
}
