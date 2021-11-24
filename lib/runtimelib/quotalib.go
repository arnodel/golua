package runtimelib

import (
	"strings"

	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
)

var LibLoader = packagelib.Loader{
	Load: load,
	Name: "runtime",
}

func load(r *rt.Runtime) rt.Value {
	if !rt.QuotasAvailable {
		return rt.NilValue
	}
	pkg := rt.NewTable()

	rt.SolemnlyDeclareSafetyFlags(
		rt.RCS_CpuSafe|rt.RCS_MemSafe|rt.RCS_IOSafe,

		r.SetEnvGoFunc(pkg, "callcontext", callcontext, 2, true),
		r.SetEnvGoFunc(pkg, "context", context, 0, false),
	)

	createContextMetatable(r)

	return rt.TableValue(pkg)
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
		flagsV    = quotas.Get(rt.StringValue("flags"))
		memQuota  int64
		cpuQuota  int64
		ok        bool
		f         = c.Arg(1)
		fArgs     = c.Etc()
		flags     rt.RuntimeSafetyFlags
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
	if !rt.IsNil(flagsV) {
		flagsStr, ok := flagsV.TryString()
		if !ok {
			return nil, rt.NewErrorS("flags must be a string").AddContext(c)
		}
		for _, name := range strings.Fields(flagsStr) {
			flags, ok = flags.AddFlagWithName(name)
			if !ok {
				return nil, rt.NewErrorF("unknown flag: %q", name).AddContext(c)
			}
		}
	}

	next = c.Next()
	res := rt.NewTerminationWith(0, true)

	ctx := t.CallContext(rt.RuntimeContextDef{
		CpuLimit:    uint64(cpuQuota),
		MemLimit:    uint64(memQuota),
		SafetyFlags: flags,
	}, func() {
		retErr = rt.Call(t, f, fArgs, res)
	})

	if retErr != nil {
		return nil, retErr
	}
	t.Push1(next, newContextValue(t.Runtime, ctx))
	if ctx.Status() == rt.RCS_Done {
		t.Push(next, res.Etc()...)
	}
	return next, nil
}
