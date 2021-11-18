package runtimelib

import (
	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
)

type quotaKeyType struct{}

var quotaKey = rt.AsValue(quotaKeyType{})

var LibLoader = packagelib.Loader{
	Load: load,
	Name: "runtime",
}

func load(r *rt.Runtime) rt.Value {
	pkg := rt.NewTable()
	r.SetEnvGoFunc(pkg, "mem", getMemQuota, 0, false)
	r.SetEnvGoFunc(pkg, "cpu", getCPUQuota, 0, false)
	r.SetEnvGoFunc(pkg, "rcall", rcall, 3, true)
	r.SetEnvGoFunc(pkg, "callcontext", callcontext, 2, true)
	r.SetEnvGoFunc(pkg, "context", context, 0, false)

	methods := rt.NewTable()
	r.SetEnvGoFunc(methods, "call", callcontext, 2, true)
	meta := rt.NewTable()
	r.SetEnv(meta, "__index", rt.TableValue(methods))

	r.SetRegistry(quotaKey, rt.AsValue(meta))

	createContextMetatable(r)

	return rt.TableValue(pkg)
}

func getMemQuota(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	used, max := t.MemQuotaStatus()
	return c.PushingNext(
		t.Runtime,
		rt.IntValue(int64(used)),
		rt.IntValue(int64(max)),
	), nil
}

func getCPUQuota(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	used, max := t.CPUQuotaStatus()
	return c.PushingNext(
		t.Runtime,
		rt.IntValue(int64(used)),
		rt.IntValue(int64(max)),
	), nil
}

func rcall(t *rt.Thread, c *rt.GoCont) (next rt.Cont, retErr *rt.Error) {
	fcpuQuota, err := c.IntArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	fmemQuota, err := c.IntArg(1)
	if err != nil {
		return nil, err.AddContext(c)
	}
	f := c.Arg(2)
	fargs := c.Etc()

	if fcpuQuota < 0 || fmemQuota < 0 {
		return nil, rt.NewErrorS("cpu and mem quota must be non-negative").AddContext(c)
	}
	// Push new quotas
	t.PushQuota(uint64(fcpuQuota), uint64(fmemQuota))

	next = c.Next()
	res := rt.NewTerminationWith(0, true)
	defer func() {
		// In any case, pop the quotas
		t.PopQuota()
		if r := recover(); r != nil {
			_, ok := r.(rt.QuotaExceededError)
			if !ok {
				panic(r)
			}
			t.Push1(next, rt.BoolValue(false))
		}
	}()
	retErr = rt.Call(t, f, fargs, res)
	if retErr != nil {
		return nil, retErr
	}
	t.Push1(next, rt.BoolValue(true))
	t.Push(next, res.Etc()...)
	return next, nil
}

func context(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	ctx := newContextValue(t.Runtime, t.RuntimeContext())
	return c.PushingNext1(t.Runtime, ctx), nil
}

func callcontext(t *rt.Thread, c *rt.GoCont) (next rt.Cont, retErr *rt.Error) {
	quotas, err := c.TableArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	var (
		memQuotaV = quotas.Get(rt.StringValue("memlimit"))
		cpuQuotaV = quotas.Get(rt.StringValue("cpulimit"))
		memQuota  int64
		cpuQuota  int64
		ok        bool
		f         = c.Arg(1)
		fArgs     = c.Etc()
	)
	if !rt.IsNil(memQuotaV) {
		memQuota, ok = memQuotaV.TryInt()
		if !ok {
			return nil, rt.NewErrorS("memquota must be an integer").AddContext(c)
		}
		if memQuota <= 0 {
			return nil, rt.NewErrorS("memquota must be positive").AddContext(c)
		}
	}
	if !rt.IsNil(cpuQuotaV) {
		cpuQuota, ok = cpuQuotaV.TryInt()
		if !ok {
			return nil, rt.NewErrorS("cpuquota must be an integer").AddContext(c)
		}
		if cpuQuota <= 0 {
			return nil, rt.NewErrorS("cpuquota must be positive").AddContext(c)
		}
	}

	// Push new quotas
	t.PushQuota(uint64(cpuQuota), uint64(memQuota))

	next = c.Next()
	res := rt.NewTerminationWith(0, true)
	defer func() {

		// var (
		// 	memUsed, memQuota = t.MemQuotaStatus()
		// 	cpuUsed, cpuQuota = t.CPUQuotaStatus()
		// )
		// if memQuota > 0 {
		// 	t.SetEnv(quotas, "memused", rt.IntValue(int64(memUsed)))
		// 	t.SetEnv(quotas, "memquota", rt.IntValue(int64(memQuota)))
		// }
		// if cpuQuota > 0 {
		// 	t.SetEnv(quotas, "cpuused", rt.IntValue(int64(cpuUsed)))
		// 	t.SetEnv(quotas, "cpuquota", rt.IntValue(int64(cpuQuota)))
		// }
		// // In any case, pop the quotas
		// t.PopQuota()

		ctx := t.PopContext()
		if retErr != nil {
			// In this case there was an error, so no panic.  We return the
			// error normally. To avoid this a user can wrap f in a pcall.
			return
		}
		t.Push1(next, newContextValue(t.Runtime, ctx))
		if r := recover(); r != nil {
			_, ok := r.(rt.QuotaExceededError)
			if !ok {
				panic(r)
			}
		} else {
			t.Push(next, res.Etc()...)
		}
	}()
	retErr = rt.Call(t, f, fArgs, res)
	if retErr != nil {
		return nil, retErr
	}
	return
}
