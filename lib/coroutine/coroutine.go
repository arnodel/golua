package coroutine

import (
	"errors"

	rt "github.com/arnodel/golua/runtime"
)

func Load(r *rt.Runtime) {
	env := r.GlobalEnv()
	pkg := rt.NewTable()
	rt.SetEnv(env, "coroutine", pkg)
	rt.SetEnvFunc(pkg, "create", create)
	rt.SetEnvFunc(pkg, "resume", resume)
	rt.SetEnvFunc(pkg, "yield", yield)
}

func create(t *rt.Thread, args []rt.Value, next rt.Cont) (rt.Cont, error) {
	if len(args) == 0 {
		return nil, errors.New("coroutine.create requires 1 argument")
	}
	f, ok := args[0].(rt.Callable)
	if !ok {
		return nil, errors.New("First argument of coroutine.create must be a function")
	}
	co := rt.NewThread(t.Runtime)
	co.Start(f)
	next.Push(co)
	return next, nil
}

func resume(t *rt.Thread, args []rt.Value, next rt.Cont) (rt.Cont, error) {
	if len(args) == 0 {
		return nil, errors.New("coroutine.resume requires 1 argument")
	}
	co, ok := args[0].(*rt.Thread)
	if !ok {
		return nil, errors.New("First argument of coroutine.resume must be a thread")
	}
	res, err := co.Resume(t, args[1:])
	if err == nil {
		next.Push(rt.Bool(true))
		rt.Push(next, res...)
	} else {
		next.Push(rt.Bool(false))
		next.Push(rt.ValueFromError(err))
	}
	return next, nil
}

func yield(t *rt.Thread, args []rt.Value, next rt.Cont) (rt.Cont, error) {
	res, err := t.Yield(args)
	if err != nil {
		return nil, err
	}
	rt.Push(next, res...)
	return next, nil
}
