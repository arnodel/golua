package stringlib

import (
	"strings"

	rt "github.com/arnodel/golua/runtime"
)

func Load(r *rt.Runtime) {
	pkg := rt.NewTable()
	rt.SetEnv(r.GlobalEnv(), "string", pkg)

	rt.SetEnvGoFunc(pkg, "byte", bytef, 3, false)
	rt.SetEnvGoFunc(pkg, "char", char, 0, true)
	rt.SetEnvGoFunc(pkg, "find", find, 4, false)
	rt.SetEnvGoFunc(pkg, "gmatch", gmatch, 2, false)
	rt.SetEnvGoFunc(pkg, "gsub", gsub, 4, false)
	rt.SetEnvGoFunc(pkg, "len", lenf, 1, false)
	rt.SetEnvGoFunc(pkg, "lower", lower, 1, false)
	rt.SetEnvGoFunc(pkg, "match", match, 3, false)
	rt.SetEnvGoFunc(pkg, "upper", upper, 1, false)
	rt.SetEnvGoFunc(pkg, "rep", rep, 3, false)
	rt.SetEnvGoFunc(pkg, "reverse", reverse, 1, false)
	rt.SetEnvGoFunc(pkg, "sub", sub, 3, false)
	rt.SetEnvGoFunc(pkg, "format", format, 1, true)

	stringMeta := rt.NewTable()
	rt.SetEnv(stringMeta, "__index", pkg)
	r.SetStringMeta(stringMeta)
}

func pos(s rt.String, n rt.Int) int {
	p := int(n)
	if p < 0 {
		p = len(s) + 1 + p
	}
	return p
}

func maxpos(i, j int) int {
	if i > j {
		return i
	}
	return j
}

func minpos(i, j int) int {
	if i < j {
		return i
	}
	return j
}

func bytef(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	s, err := c.StringArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	i, j := 1, 1
	if c.NArgs() >= 2 {
		ii, err := c.IntArg(1)
		if err != nil {
			return nil, err.AddContext(c)
		}
		i = pos(s, ii)
		j = i
	}
	if c.NArgs() >= 3 {
		jj, err := c.IntArg(2)
		if err != nil {
			return nil, err.AddContext(c)
		}
		j = pos(s, jj)
	}
	next := c.Next()
	i = maxpos(1, i)
	j = minpos(len(s), j)
	for i <= j {
		next.Push(rt.Int(s[i-1]))
		i++
	}
	return next, nil
}

func char(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	vals := c.Etc()
	buf := make([]byte, len(vals))
	for i, v := range vals {
		x, tp := rt.ToInt(v)
		if tp != rt.IsInt {
			return nil, rt.NewErrorS("arguments must be integers").AddContext(c)
		}
		if x < 0 || x > 255 {
			return nil, rt.NewErrorF("#%d out of range", i+1).AddContext(c)
		}
		buf[i] = byte(x)
	}
	return c.PushingNext(rt.String(buf)), nil
}

func lenf(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	s, err := c.StringArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	return c.PushingNext(rt.Int(len(s))), nil
}

func lower(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	s, err := c.StringArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	s = rt.String(strings.ToLower(string(s)))
	return c.PushingNext(s), nil
}

func upper(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	s, err := c.StringArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	s = rt.String(strings.ToUpper(string(s)))
	return c.PushingNext(s), nil
}

func rep(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err.AddContext(c)
	}
	ls, err := c.StringArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	ln, err := c.IntArg(1)
	if err != nil {
		return nil, err.AddContext(c)
	}
	n := int(ln)
	if n < 0 {
		return nil, rt.NewErrorS("#2 out of range").AddContext(c)
	}
	var sep []byte
	if c.NArgs() >= 3 {
		lsep, err := c.StringArg(2)
		if err != nil {
			return nil, err.AddContext(c)
		}
		sep = []byte(lsep)
	}
	if n == 0 {
		return c.PushingNext(rt.String("")), nil
	}
	if n == 1 {
		return c.PushingNext(ls), nil
	}
	if sep == nil {
		return c.PushingNext(rt.String(strings.Repeat(string(ls), n))), nil
	}
	s := []byte(ls)
	builder := strings.Builder{}
	builder.Grow(n*len(s) + (n-1)*len(sep))
	builder.Write(s)
	for {
		n--
		if n == 0 {
			break
		}
		builder.Write(sep)
		builder.Write(s)
	}
	return c.PushingNext(rt.String(builder.String())), nil
}

func reverse(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	s, err := c.StringArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	sb := []byte(s)
	l := len(s) - 1
	for i := 0; 2*i <= l; i++ {
		sb[i], sb[l-i] = sb[l-i], sb[i]
	}
	return c.PushingNext(rt.String(sb)), nil
}

func sub(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err.AddContext(c)
	}
	s, err := c.StringArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	ii, err := c.IntArg(1)
	if err != nil {
		return nil, err.AddContext(c)
	}
	i := pos(s, ii)
	j := len(s)
	if c.NArgs() >= 3 {
		jj, err := c.IntArg(2)
		if err != nil {
			return nil, err.AddContext(c)
		}
		j = pos(s, jj)
	}
	var slice rt.String
	i = maxpos(1, i)
	j = minpos(len(s), j)
	if i <= len(s) && i <= j {
		slice = s[i-1 : j]
	}
	return c.PushingNext(slice), nil
}
