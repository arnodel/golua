package coroutine

import (
	rt "github.com/arnodel/golua/runtime"
)

func Load(r *rt.Runtime) {
	env := r.GlobalEnv()
	pkg := rt.NewTable()
	rt.SetEnv(env, "coroutine", pkg)
	rt.SetEnvGoFunc(pkg, "create", create, 1, false)
	rt.SetEnvGoFunc(pkg, "resume", resume, 1, true)
	rt.SetEnvGoFunc(pkg, "yield", yield, 0, true)
}

func create(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if c.NArgs() == 0 {
		return nil, rt.NewErrorS("1 argument required").AddContext(c)
	}
	f, ok := c.Arg(0).(rt.Callable)
	if !ok {
		return nil, rt.NewErrorS("#1 must be a function").AddContext(c)
	}
	co := rt.NewThread(t.Runtime)
	co.Start(f)
	c.Next().Push(co)
	return c.Next(), nil
}

func resume(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if c.NArgs() == 0 {
		return nil, rt.NewErrorS("1 argument required").AddContext(c)
	}
	co, ok := c.Arg(0).(*rt.Thread)
	if !ok {
		return nil, rt.NewErrorS("#1 must be a thread").AddContext(c)
	}
	res, err := co.Resume(t, c.Etc())
	next := c.Next()
	if err == nil {
		next.Push(rt.Bool(true))
		rt.Push(next, res...)
	} else {
		next.Push(rt.Bool(false))
		next.Push(err.Value())
	}
	return next, nil
}

func yield(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	res, err := t.Yield(c.Etc())
	if err != nil {
		return nil, err
	}
	rt.Push(c.Next(), res...)
	return c.Next(), nil
}
