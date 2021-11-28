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

	rt.SolemnlyDeclareCompliance(
		rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyIoSafe,

		r.SetEnvGoFunc(pkg, "callcontext", callcontext, 2, true),
		r.SetEnvGoFunc(pkg, "context", context, 0, false),
		r.SetEnvGoFunc(pkg, "cancelcontext", cancel, 0, false),
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
		return nil, err
	}
	var (
		memQuotaV  = quotas.Get(rt.StringValue("memlimit")) // deprecated
		cpuQuotaV  = quotas.Get(rt.StringValue("cpulimit")) // deprecated
		flagsV     = quotas.Get(rt.StringValue("flags"))
		limitsV    = quotas.Get(rt.StringValue("limits"))
		memQuota   uint64
		cpuQuota   uint64
		timerQuota uint64
		f          = c.Arg(1)
		fArgs      = c.Etc()
		flags      rt.ComplianceFlags
	)
	if !rt.IsNil(limitsV) {
		var err *rt.Error
		cpuQuota, err = getResVal(t, limitsV, "cpu")
		if err != nil {
			return nil, err
		}
		memQuota, err = getResVal(t, limitsV, "mem")
		if err != nil {
			return nil, err
		}
		timerQuota, err = getResVal(t, limitsV, "timer")
		if err != nil {
			return nil, err
		}
	}
	if !rt.IsNil(memQuotaV) {
		memQuota, err = validateResVal("memlimit", memQuotaV)
		if err != nil {
			return nil, err
		}
	}
	if !rt.IsNil(cpuQuotaV) {
		cpuQuota, err = validateResVal("cpulimit", cpuQuotaV)
		if err != nil {
			return nil, err
		}
	}
	if !rt.IsNil(flagsV) {
		flagsStr, ok := flagsV.TryString()
		if !ok {
			return nil, rt.NewErrorS("flags must be a string")
		}
		for _, name := range strings.Fields(flagsStr) {
			flags, ok = flags.AddFlagWithName(name)
			if !ok {
				return nil, rt.NewErrorF("unknown flag: %q", name)
			}
		}
	}

	next = c.Next()
	res := rt.NewTerminationWith(c, 0, true)

	ctx, err := t.CallContext(rt.RuntimeContextDef{
		HardLimits: rt.RuntimeResources{
			Cpu:   cpuQuota,
			Mem:   memQuota,
			Timer: timerQuota,
		},
		SafetyFlags: flags,
	}, func() *rt.Error {
		return rt.Call(t, f, fArgs, res)
	})
	t.Push1(next, newContextValue(t.Runtime, ctx))
	switch ctx.Status() {
	case rt.RCS_Done:
		t.Push(next, res.Etc()...)
	case rt.RCS_Error:
		t.Push1(next, err.Value())
	}
	return next, nil
}

func cancel(t *rt.Thread, c *rt.GoCont) (next rt.Cont, retErr *rt.Error) {
	t.TerminateContext("cancelled")
	return nil, nil
}

func getResVal(t *rt.Thread, resources rt.Value, name string) (uint64, *rt.Error) {
	val, err := rt.Index(t, resources, rt.StringValue(name))
	if err != nil {
		return 0, err
	}
	return validateResVal(name, val)
}

func validateResVal(name string, val rt.Value) (uint64, *rt.Error) {
	if rt.IsNil(val) {
		return 0, nil
	}
	n, ok := val.TryInt()
	if !ok {
		return 0, rt.NewErrorF("%s must be an integer", name)
	}
	if n <= 0 {
		return 0, rt.NewErrorF("%s must be a positive integer", name)
	}
	return uint64(n), nil
}
