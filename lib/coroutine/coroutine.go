package coroutine

import (
	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
)

// LibLoader allows loading the coroutine lib
var LibLoader = packagelib.Loader{
	Load: load,
	Name: "coroutine",
}

func load(r *rt.Runtime) rt.Value {
	pkg := rt.NewTable()
	rt.SetEnvGoFunc(pkg, "create", create, 1, false)
	rt.SetEnvGoFunc(pkg, "isyieldable", isyieldable, 0, false)
	rt.SetEnvGoFunc(pkg, "resume", resume, 1, true)
	rt.SetEnvGoFunc(pkg, "running", running, 0, false)
	rt.SetEnvGoFunc(pkg, "status", status, 1, false)
	rt.SetEnvGoFunc(pkg, "wrap", wrap, 1, false)
	rt.SetEnvGoFunc(pkg, "yield", yield, 0, true)
	return rt.TableValue(pkg)
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
	c.Next().Push(rt.ThreadValue(co))
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
		next.Push(rt.BoolValue(true))
		rt.Push(next, res...)
	} else {
		next.Push(rt.BoolValue(false))
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
	next.Push(rt.BoolValue(!t.IsMain()))
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
	next.Push(rt.StringValue(status))
	return next, nil
}

func running(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	next := c.Next()
	next.Push(rt.ThreadValue(t))
	next.Push(rt.BoolValue(t.IsMain()))
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
	next.Push(rt.FunctionValue(w))
	return next, nil
}
