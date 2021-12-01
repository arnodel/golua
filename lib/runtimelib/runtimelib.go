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
		r.SetEnvGoFunc(pkg, "stopcontext", stopcontext, 0, false),
		r.SetEnvGoFunc(pkg, "shouldstop", shouldstop, 0, false),
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
		memQuotaV   = quotas.Get(rt.StringValue("memlimit")) // deprecated
		cpuQuotaV   = quotas.Get(rt.StringValue("cpulimit")) // deprecated
		flagsV      = quotas.Get(rt.StringValue("flags"))
		limitsV     = quotas.Get(rt.StringValue("limits"))
		softLimitsV = quotas.Get(rt.StringValue("softlimits"))
		hardLimits  rt.RuntimeResources
		softLimits  rt.RuntimeResources
		f           = c.Arg(1)
		fArgs       = c.Etc()
		flags       rt.ComplianceFlags
	)
	if !rt.IsNil(limitsV) {
		var err *rt.Error
		hardLimits, err = getResources(t, limitsV)
		if err != nil {
			return nil, err
		}
	}
	if !rt.IsNil(softLimitsV) {
		var err *rt.Error
		softLimits, err = getResources(t, softLimitsV)
		if err != nil {
			return nil, err
		}
	}
	if !rt.IsNil(memQuotaV) {
		hardLimits.Mem, err = validateResVal("memlimit", memQuotaV)
		if err != nil {
			return nil, err
		}
	}
	if !rt.IsNil(cpuQuotaV) {
		hardLimits.Cpu, err = validateResVal("cpulimit", cpuQuotaV)
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
		HardLimits:    hardLimits,
		SoftLimits:    softLimits,
		RequiredFlags: flags,
	}, func() *rt.Error {
		return rt.Call(t, f, fArgs, res)
	})
	t.Push1(next, newContextValue(t.Runtime, ctx))
	switch ctx.Status() {
	case rt.StatusDone:
		t.Push(next, res.Etc()...)
	case rt.StatusError:
		t.Push1(next, err.Value())
	}
	return next, nil
}

func stopcontext(t *rt.Thread, c *rt.GoCont) (next rt.Cont, retErr *rt.Error) {
	t.TerminateContext("stopped")
	return nil, nil
}

func shouldstop(t *rt.Thread, c *rt.GoCont) (next rt.Cont, retErr *rt.Error) {
	return c.PushingNext1(t.Runtime, rt.BoolValue(t.ShouldStop())), nil
}

func getResources(t *rt.Thread, resources rt.Value) (res rt.RuntimeResources, err *rt.Error) {
	res.Cpu, err = getResVal(t, resources, "cpu")
	if err != nil {
		return
	}
	res.Mem, err = getResVal(t, resources, "mem")
	if err != nil {
		return
	}
	res.Time, err = getResVal(t, resources, "time")
	if err != nil {
		return
	}
	return
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
	n, ok := rt.ToIntNoString(val)
	if !ok {
		return 0, rt.NewErrorF("%s must be an integer", name)
	}
	if n <= 0 {
		return 0, rt.NewErrorF("%s must be a positive integer", name)
	}
	return uint64(n), nil
}
