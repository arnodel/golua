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

	if r.QuotaModificationsInLuaAllowed() {
		r.SetEnvGoFunc(pkg, "reset", resetQuota, 0, false)
		r.SetEnvGoFunc(pkg, "rcall", rcall, 3, true)
	}
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

func resetQuota(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	t.ResetQuota()
	return c.Next(), nil
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

	// Record the current quota values, to be restored at the end
	memUsed, memQuota := t.MemQuotaStatus()
	cpuUsed, cpuQuota := t.CPUQuotaStatus()

	if fcpuQuota >= 0 {
		t.UpdateCPUQuota(uint64(fcpuQuota))
	}
	if fmemQuota >= 0 {
		t.UpdateMemQuota(uint64(fmemQuota))
	}
	t.ResetQuota()

	next = c.Next()
	res := rt.NewTerminationWith(0, true)
	defer func() {
		// In any case, restore the quota values.  Do it before we push the
		// return value to avoid another QuotaExceededError!
		t.ResetQuota()
		t.UpdateCPUQuota(cpuQuota)
		t.UpdateMemQuota(memQuota)
		t.RequireCPU(cpuUsed)
		t.RequireMem(memUsed)
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
