package stringlib

import rt "github.com/arnodel/golua/runtime"

func Load(r *rt.Runtime) {
	pkg := rt.NewTable()
	rt.SetEnv(r.GlobalEnv(), "string", pkg)
	rt.SetEnvGoFunc(pkg, "byte", bytef, 3, false)
	stringMeta := rt.NewTable()
	rt.SetEnv(stringMeta, "__index", pkg)
	r.SetStringMeta(stringMeta)
}

func bytef(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	s, err := c.StringArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	var i rt.Int = 1
	var j rt.Int = 1
	if c.NArgs() >= 2 {
		var err *rt.Error
		i, err = c.IntArg(1)
		if err != nil {
			return nil, err.AddContext(c)
		}
		j = i
	}
	if c.NArgs() >= 3 {
		var err *rt.Error
		j, err = c.IntArg(2)
		if err != nil {
			return nil, err.AddContext(c)
		}
	}
	if j < 0 {
		j = rt.Int(len(s)+1) + j
	} else if j > rt.Int(len(s)) {
		j = rt.Int(len(s))
	}
	if i < 0 {
		i = rt.Int(len(s)+1) + i
	}
	if i <= 0 {
		i = rt.Int(1)
	}
	next := c.Next()
	for i <= j {
		next.Push(rt.Int(s[i-1]))
		i++
	}
	return next, nil
}
