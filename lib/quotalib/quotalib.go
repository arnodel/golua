package quotalib

import (
	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
)

var LibLoader = packagelib.Loader{
	Load: load,
	Name: "quota",
}

func load(r *rt.Runtime) rt.Value {
	pkg := rt.NewTable()
	r.SetEnvGoFunc(pkg, "mem", getMemQuota, 0, false)
	r.SetEnvGoFunc(pkg, "cpu", getCPUQuota, 0, false)
	r.SetEnvGoFunc(pkg, "rcall", rcall, 3, true)

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
