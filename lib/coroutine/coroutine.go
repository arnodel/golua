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

	rt.SolemnlyDeclareCompliance(
		rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyTimeSafe|rt.ComplyIoSafe,

		r.SetEnvGoFunc(pkg, "create", create, 1, false),
		r.SetEnvGoFunc(pkg, "isyieldable", isyieldable, 0, false),
		r.SetEnvGoFunc(pkg, "resume", resume, 1, true),
		r.SetEnvGoFunc(pkg, "running", running, 0, false),
		r.SetEnvGoFunc(pkg, "status", status, 1, false),
		r.SetEnvGoFunc(pkg, "wrap", wrap, 1, false),
		r.SetEnvGoFunc(pkg, "yield", yield, 0, true),
	)

	return rt.TableValue(pkg)
}

func create(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var f rt.Callable
	err := c.Check1Arg()
	if err == nil {
		f, err = c.CallableArg(0)
	}
	if err != nil {
		return nil, err
	}
	co := rt.NewThread(t.Runtime)
	co.Start(f)
	return c.PushingNext1(t.Runtime, rt.ThreadValue(co)), nil
}

func resume(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var co *rt.Thread
	err := c.Check1Arg()
	if err == nil {
		co, err = c.ThreadArg(0)
	}
	if err != nil {
		return nil, err
	}
	res, err := co.Resume(t, c.Etc())
	next := c.Next()
	if err == nil {
		t.Push1(next, rt.BoolValue(true))
		t.Push(next, res...)
	} else {
		t.Push1(next, rt.BoolValue(false))
		t.Push1(next, err.Value())
	}
	return next, nil
}

func yield(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	res, err := t.Yield(c.Etc())
	if err != nil {
		return nil, err
	}
	return c.PushingNext(t.Runtime, res...), nil
}

func isyieldable(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	next := c.Next()
	t.Push1(next, rt.BoolValue(!t.IsMain()))
	return next, nil
}

func status(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var co *rt.Thread
	err := c.Check1Arg()
	if err == nil {
		co, err = c.ThreadArg(0)
	}
	if err != nil {
		return nil, err
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
	t.Push1(next, rt.StringValue(status))
	return next, nil
}

func running(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	next := c.Next()
	t.Push1(next, rt.ThreadValue(t))
	t.Push1(next, rt.BoolValue(t.IsMain()))
	return next, nil
}

func wrap(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var f rt.Callable
	err := c.Check1Arg()
	if err == nil {
		f, err = c.CallableArg(0)
	}
	if err != nil {
		return nil, err
	}
	co := rt.NewThread(t.Runtime)
	co.Start(f)
	w := rt.NewGoFunction(func(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
		res, err := co.Resume(t, c.Etc())
		if err != nil {
			return nil, err
		}
		return c.PushingNext(t.Runtime, res...), nil
	}, "wrap", 0, true)
	w.SolemnlyDeclareCompliance(rt.ComplyCpuSafe | rt.ComplyMemSafe | rt.ComplyTimeSafe | rt.ComplyIoSafe)
	next := c.Next()
	t.Push1(next, rt.FunctionValue(w))
	return next, nil
}
