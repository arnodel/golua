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
		rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyTimeSafe|rt.ComplyIoSafe,

		r.SetEnvGoFunc(pkg, "callcontext", callcontext, 2, true),
		r.SetEnvGoFunc(pkg, "context", context, 0, false),
		r.SetEnvGoFunc(pkg, "killcontext", killnow, 1, false),
		r.SetEnvGoFunc(pkg, "stopcontext", stopnow, 1, false),
		r.SetEnvGoFunc(pkg, "contextdue", due, 1, false),
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
		flagsV      = quotas.Get(rt.StringValue("flags"))
		limitsV     = quotas.Get(rt.StringValue("kill"))
		softLimitsV = quotas.Get(rt.StringValue("stop"))
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

func getResources(t *rt.Thread, resources rt.Value) (res rt.RuntimeResources, err *rt.Error) {
	res.Cpu, err = getResVal(t, resources, cpuString)
	if err != nil {
		return
	}
	res.Memory, err = getResVal(t, resources, memoryString)
	if err != nil {
		return
	}
	res.Millis, err = getTimeVal(t, resources)
	if err != nil {
		return
	}
	return
}

func getResVal(t *rt.Thread, resources rt.Value, key rt.Value) (uint64, *rt.Error) {
	val, err := rt.Index(t, resources, key)
	if err != nil {
		return 0, err
	}
	return validateResVal(key, val)
}

func validateResVal(key rt.Value, val rt.Value) (uint64, *rt.Error) {
	if rt.IsNil(val) {
		return 0, nil
	}
	n, ok := rt.ToIntNoString(val)
	if !ok {
		name, _ := key.ToString()
		return 0, rt.NewErrorF("%s must be an integer", name)
	}
	if n <= 0 {
		name, _ := key.ToString()
		return 0, rt.NewErrorF("%s must be a positive integer", name)
	}
	return uint64(n), nil
}

func getTimeVal(t *rt.Thread, resources rt.Value) (uint64, *rt.Error) {
	val, err := rt.Index(t, resources, secondsString)
	if err != nil {
		return 0, err
	}
	if !rt.IsNil(val) {
		return validateTimeVal(val, 1000, secondsName)
	}
	val, err = rt.Index(t, resources, millisString)
	if err != nil {
		return 0, err
	}
	return validateTimeVal(val, 1, millisName)
}

func validateTimeVal(val rt.Value, factor float64, name string) (uint64, *rt.Error) {
	if rt.IsNil(val) {
		return 0, nil
	}
	s, ok := rt.ToFloat(val)
	if !ok {
		return 0, rt.NewErrorF("%s must be a numeric value", name)
	}
	if s <= 0 {
		return 0, rt.NewErrorF("%s must be positive", name)
	}
	return uint64(s * factor), nil
}

const (
	secondsName = "seconds"
	millisName  = "millis"
	cpuName     = "cpu"
	memoryName  = "memory"
)

var (
	secondsString = rt.StringValue(secondsName)
	millisString  = rt.StringValue(millisName)
	cpuString     = rt.StringValue(cpuName)
	memoryString  = rt.StringValue(memoryName)
)
