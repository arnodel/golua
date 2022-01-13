package mathlib

import (
	"math"
	"math/rand"
	"time"

	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
)

var LibLoader = packagelib.Loader{
	Load: load,
	Name: "math",
}

func load(r *rt.Runtime) (rt.Value, func()) {
	pkg := rt.NewTable()
	r.SetEnv(pkg, "huge", rt.FloatValue(math.Inf(1)))
	r.SetEnv(pkg, "maxinteger", rt.IntValue(math.MaxInt64))
	r.SetEnv(pkg, "mininteger", rt.IntValue(math.MinInt64))
	r.SetEnv(pkg, "pi", rt.FloatValue(math.Pi))

	rt.SolemnlyDeclareCompliance(
		rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyTimeSafe|rt.ComplyIoSafe,

		r.SetEnvGoFunc(pkg, "abs", abs, 1, false),
		r.SetEnvGoFunc(pkg, "acos", acos, 1, false),
		r.SetEnvGoFunc(pkg, "asin", asin, 1, false),
		r.SetEnvGoFunc(pkg, "atan", atan, 2, false),
		r.SetEnvGoFunc(pkg, "ceil", ceil, 1, false),
		r.SetEnvGoFunc(pkg, "cos", cos, 1, false),
		r.SetEnvGoFunc(pkg, "deg", deg, 1, false),
		r.SetEnvGoFunc(pkg, "exp", exp, 1, false),
		r.SetEnvGoFunc(pkg, "floor", floor, 1, false),
		r.SetEnvGoFunc(pkg, "fmod", fmod, 2, false),
		r.SetEnvGoFunc(pkg, "log", log, 2, false),
		r.SetEnvGoFunc(pkg, "max", max, 1, true),
		r.SetEnvGoFunc(pkg, "min", min, 1, true),
		r.SetEnvGoFunc(pkg, "modf", modf, 1, false),
		r.SetEnvGoFunc(pkg, "rad", rad, 1, false),
		r.SetEnvGoFunc(pkg, "random", random, 2, false),
		r.SetEnvGoFunc(pkg, "randomseed", randomseed, 2, false),
		r.SetEnvGoFunc(pkg, "sin", sin, 1, false),
		r.SetEnvGoFunc(pkg, "sqrt", sqrt, 1, false),
		r.SetEnvGoFunc(pkg, "tan", tan, 1, false),
		r.SetEnvGoFunc(pkg, "tointeger", tointeger, 1, false),
		r.SetEnvGoFunc(pkg, "type", typef, 1, false),
		r.SetEnvGoFunc(pkg, "ult", ult, 2, false),
	)

	return rt.TableValue(pkg), nil
}

func abs(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	next := c.Next()
	n, f, tp := rt.ToNumber(c.Arg(0))
	switch tp {
	case rt.IsInt:
		if n < 0 {
			n = -n
		}
		t.Push1(next, rt.IntValue(n))
	case rt.IsFloat:
		t.Push1(next, rt.FloatValue(math.Abs(f)))
	default:
		return nil, rt.NewErrorS("#1 must be a number")
	}
	return next, nil
}

func acos(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	y := rt.FloatValue(math.Acos(x))
	return c.PushingNext1(t.Runtime, y), nil
}

func asin(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	y := rt.FloatValue(math.Asin(x))
	return c.PushingNext1(t.Runtime, y), nil
}

func atan(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	y, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	var x float64 = 1
	if c.NArgs() >= 2 {
		x, err = c.FloatArg(1)
		if err != nil {
			return nil, err
		}
	}
	z := rt.FloatValue(math.Atan2(y, x))
	return c.PushingNext1(t.Runtime, z), nil
}

func ceil(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	next := c.Next()
	n, f, tp := rt.ToNumber(c.Arg(0))
	switch tp {
	case rt.IsInt:
		t.Push1(next, rt.IntValue(n))
	case rt.IsFloat:
		f = math.Ceil(f)
		n = int64(f)
		if float64(n) == f {
			t.Push1(next, rt.IntValue(n))
		} else {
			t.Push1(next, rt.FloatValue(f))
		}
	default:
		return nil, rt.NewErrorS("#1 must be a number")
	}
	return next, nil
}

func cos(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	y := rt.FloatValue(math.Cos(x))
	return c.PushingNext1(t.Runtime, y), nil
}

func deg(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	y := rt.FloatValue(x * 180 / math.Pi)
	return c.PushingNext1(t.Runtime, y), nil
}

func exp(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	y := rt.FloatValue(math.Exp(x))
	return c.PushingNext1(t.Runtime, y), nil
}

func floor(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	next := c.Next()
	n, f, tp := rt.ToNumber(c.Arg(0))
	switch tp {
	case rt.IsInt:
		t.Push1(next, rt.IntValue(n))
	case rt.IsFloat:
		f = math.Floor(f)
		n = int64(f)
		if float64(n) == f {
			t.Push1(next, rt.IntValue(n))
		} else {
			t.Push1(next, rt.FloatValue(f))
		}
	default:
		return nil, rt.NewErrorS("#1 must be a number")
	}
	return next, nil
}

func fmod(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}
	x, _ := rt.ToNumberValue(c.Arg(0))
	y, _ := rt.ToNumberValue(c.Arg(1))
	res, ok, err := rt.Mod(x, y)
	if !ok {
		err = rt.NewErrorS("expected numeric arguments")
	}
	if err != nil {
		return nil, err
	}
	return c.PushingNext1(t.Runtime, res), nil
}

func log(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	y := math.Log(x)
	if c.NArgs() >= 2 {
		b, err := c.FloatArg(1)
		if err != nil {
			return nil, err
		}
		y = y / math.Log(b)
	}
	return c.PushingNext1(t.Runtime, rt.FloatValue(y)), nil
}

func max(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	x := c.Arg(0)
	for _, y := range c.Etc() {
		lt, err := rt.Lt(t, x, y)
		if err != nil {
			return nil, err
		}
		if lt {
			x = y
		}
	}
	return c.PushingNext1(t.Runtime, x), nil
}

func min(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	x := c.Arg(0)
	for _, y := range c.Etc() {
		lt, err := rt.Lt(t, y, x)
		if err != nil {
			return nil, err
		}
		if lt {
			x = y
		}
	}
	return c.PushingNext1(t.Runtime, x), nil
}

func modf(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	next := c.Next()
	arg := c.Arg(0)
	if _, ok := arg.TryInt(); ok {
		t.Push1(next, arg)
		t.Push1(next, rt.FloatValue(0))
		return next, nil
	}

	x, ok := arg.TryFloat()
	if !ok {
		return nil, rt.NewErrorS("#1 must be numeric")
	}
	var i, f float64
	if math.IsInf(x, 0) {
		i, f = x, 0
	} else {
		i, f = math.Modf(x)
	}
	t.Push1(next, rt.FloatValue(i))
	t.Push1(next, rt.FloatValue(f))
	return next, nil
}

func rad(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	y := rt.FloatValue(x * math.Pi / 180)
	return c.PushingNext1(t.Runtime, y), nil
}

// TODO: have a per runtime random generator
func random(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var (
		err *rt.Error
		m   int64 = 1
		n   int64
	)
	switch c.NArgs() {
	case 0:
		return c.PushingNext1(t.Runtime, rt.FloatValue(rand.Float64())), nil
	case 1:
		n, err = c.IntArg(0)
	case 2:
		m, err = c.IntArg(0)
		if err == nil {
			n, err = c.IntArg(1)
		}
	}
	if err != nil {
		return nil, err
	}
	if m > n {
		return nil, rt.NewErrorS("#2 must be >= #1")
	}
	var r int64
	if m <= 0 && m+math.MaxInt64 < n {
		return nil, rt.NewErrorS("interval too large")
	} else if m+math.MaxInt64 == n {
		r = rand.Int63()
	} else {
		r = rand.Int63n(n - m + 1)
	}
	return c.PushingNext1(t.Runtime, rt.IntValue(m+r)), nil
}

func randomseed(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var seed int64
	var err *rt.Error
	switch c.NArgs() {
	case 0:
		// Something "random"
		seed = time.Now().UnixNano()
	case 1:
		seed, err = c.IntArg(0)
		if err != nil {
			return nil, err
		}
	case 2:
		seed, err = c.IntArg(0)
		if err != nil {
			return nil, err
		}
		seed2, err := c.IntArg(1)
		if err != nil {
			return nil, err
		}
		// In Go the seed is only 64 bits so we mangle the seeds
		seed ^= seed2
	}
	rand.Seed(seed)
	return c.PushingNext(t.Runtime, rt.IntValue(seed), rt.IntValue(0)), nil
}

func sin(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	y := rt.FloatValue(math.Sin(x))
	return c.PushingNext1(t.Runtime, y), nil
}

func sqrt(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	y := rt.FloatValue(math.Sqrt(x))
	return c.PushingNext1(t.Runtime, y), nil
}

func tan(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	y := rt.FloatValue(math.Tan(x))
	return c.PushingNext1(t.Runtime, y), nil
}

func tointeger(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	n, err := c.IntArg(0)
	if err != nil {
		return c.PushingNext1(t.Runtime, rt.NilValue), nil
	}
	return c.PushingNext1(t.Runtime, rt.IntValue(n)), nil
}

func typef(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	var tp rt.Value
	switch c.Arg(0).NumberType() {
	case rt.IntType:
		tp = rt.StringValue("integer")
	case rt.FloatType:
		tp = rt.StringValue("float")
	}
	return c.PushingNext1(t.Runtime, tp), nil
}

func ult(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}
	x, err := c.IntArg(0)
	var y int64
	if err == nil {
		y, err = c.IntArg(1)
	}
	if err != nil {
		return nil, err
	}
	lt := rt.BoolValue(uint64(x) < uint64(y))
	return c.PushingNext1(t.Runtime, lt), nil
}
