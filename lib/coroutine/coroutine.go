package coroutine

import (
	rt "github.com/arnodel/golua/runtime"
)

func Load(r *rt.Runtime) {
	env := r.GlobalEnv()
	pkg := rt.NewTable()
	rt.SetEnv(env, "coroutine", pkg)
	rt.SetEnvGoFunc(pkg, "create", create, 1, false)
	rt.SetEnvGoFunc(pkg, "isyieldable", isyieldable, 0, false)
	rt.SetEnvGoFunc(pkg, "resume", resume, 1, true)
	rt.SetEnvGoFunc(pkg, "running", running, 0, false)
	rt.SetEnvGoFunc(pkg, "status", status, 1, false)
	rt.SetEnvGoFunc(pkg, "wrap", wrap, 1, false)
	rt.SetEnvGoFunc(pkg, "yield", yield, 0, true)
}

func create(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var f rt.Callable
	err := c.Check1Arg()
	if err == nil {
		f, err = c.CallableArg(0)
	}
	if err != nil {
		return nil, err.AddContext(c)
	}
	co := rt.NewThread(t.Runtime)
	co.Start(f)
	c.Next().Push(co)
	return c.Next(), nil
}

func resume(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var co *rt.Thread
	err := c.Check1Arg()
	if err == nil {
		co, err = c.ThreadArg(0)
	}
	if err != nil {
		return nil, err.AddContext(c)
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

func isyieldable(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	next := c.Next()
	next.Push(rt.Bool(!t.IsMain()))
	return next, nil
}

func status(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var co *rt.Thread
	err := c.Check1Arg()
	if err == nil {
		co, err = c.ThreadArg(0)
	}
	if err != nil {
		return nil, err.AddContext(c)
	}
	var status string
	if co == t {
		status = "running"
	} else {
		switch co.Status() {
		case rt.ThreadDead:
			status = "dead"
		case rt.ThreadSuspended:
			status = "suspended"
		case rt.ThreadOK:
			status = "normal"
		}
	}
	next := c.Next()
	next.Push(rt.String(status))
	return next, nil
}

func running(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	next := c.Next()
	next.Push(t)
	next.Push(rt.Bool(t.IsMain()))
	return next, nil
}

func wrap(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var f rt.Callable
	err := c.Check1Arg()
	if err == nil {
		f, err = c.CallableArg(0)
	}
	if err != nil {
		return nil, err.AddContext(c)
	}
	co := rt.NewThread(t.Runtime)
	co.Start(f)
	w := rt.NewGoFunction(func(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
		res, err := co.Resume(t, c.Etc())
		if err != nil {
			return nil, err.AddContext(c)
		}
		rt.Push(c.Next(), res...)
		return c.Next(), nil
	}, "wrap", 0, true)
	next := c.Next()
	next.Push(w)
	return next, nil
}
