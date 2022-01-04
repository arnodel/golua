package runtimelib

import (
	"fmt"
	"strings"

	rt "github.com/arnodel/golua/runtime"
)

var contextRegistryKey = rt.AsValue(contextRegistry{})

func createContextMetatable(r *rt.Runtime) {
	contextMeta := rt.NewTable()

	rt.SolemnlyDeclareCompliance(
		rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyTimeSafe|rt.ComplyIoSafe,

		r.SetEnvGoFunc(contextMeta, "__index", context__index, 2, false),
		r.SetEnvGoFunc(contextMeta, "__tostring", context__tostring, 1, false),
	)

	resourcesMeta := rt.NewTable()
	rt.SolemnlyDeclareCompliance(
		rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyTimeSafe|rt.ComplyIoSafe,

		r.SetEnvGoFunc(resourcesMeta, "__index", resources__index, 2, false),
		r.SetEnvGoFunc(resourcesMeta, "__tostring", resources__tostring, 1, false),
	)
	r.SetRegistry(contextRegistryKey, rt.AsValue(&contextRegistry{
		contextMeta:   contextMeta,
		resourcesMeta: resourcesMeta,
	}))
}

type contextRegistry struct {
	contextMeta   *rt.Table
	resourcesMeta *rt.Table
}

func getRegistry(r *rt.Runtime) *contextRegistry {
	return r.Registry(contextRegistryKey).Interface().(*contextRegistry)
}

func newContextUserData(r *rt.Runtime, ctx rt.RuntimeContext) *rt.UserData {
	return rt.NewUserData(ctx, getRegistry(r).contextMeta)
}

func newContextValue(r *rt.Runtime, ctx rt.RuntimeContext) rt.Value {
	return rt.UserDataValue(newContextUserData(r, ctx))
}

func valueToContext(v rt.Value) (rt.RuntimeContext, bool) {
	u, ok := v.TryUserData()
	if !ok {
		return nil, false
	}
	ctx, ok := u.Value().(rt.RuntimeContext)
	if !ok {
		return nil, false
	}
	return ctx, true
}

func newResourcesUserData(r *rt.Runtime, res rt.RuntimeResources) *rt.UserData {
	return rt.NewUserData(res, getRegistry(r).resourcesMeta)
}

func newResourcesValue(r *rt.Runtime, res rt.RuntimeResources) rt.Value {
	return rt.UserDataValue(newResourcesUserData(r, res))
}

func valueToResources(v rt.Value) (res rt.RuntimeResources, ok bool) {
	var u *rt.UserData
	u, ok = v.TryUserData()
	if !ok {
		return
	}
	res, ok = u.Value().(rt.RuntimeResources)
	return
}

func contextArg(c *rt.GoCont, n int) (rt.RuntimeContext, *rt.Error) {
	ctx, ok := valueToContext(c.Arg(n))
	if ok {
		return ctx, nil
	}
	return nil, rt.NewErrorF("#%d must be a runtime context", n+1)
}

func resourcesArg(c *rt.GoCont, n int) (rt.RuntimeResources, *rt.Error) {
	res, ok := valueToResources(c.Arg(n))
	if ok {
		return res, nil
	}
	return res, rt.NewErrorF("#%d must be a runtime resources", n+1)
}

func context__index(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	ctx, err := contextArg(c, 0)
	if err != nil {
		return nil, err
	}
	key, err := c.StringArg(1)
	if err != nil {
		return nil, err
	}
	val := rt.NilValue
	switch key {
	case "kill":
		val = newResourcesValue(t.Runtime, ctx.HardLimits())
	case "stop":
		val = newResourcesValue(t.Runtime, ctx.SoftLimits())
	case "used":
		val = newResourcesValue(t.Runtime, ctx.UsedResources())
	case "cpulimit": // Deprecated
		{
			limit := ctx.HardLimits().Cpu
			if limit > 0 {
				val = resToVal(limit)
			}
		}
	case "memlimit": // Deprecated
		{
			limit := ctx.HardLimits().Mem
			if limit > 0 {
				val = resToVal(limit)
			}
		}
	case "cpuused": // Deprecated
		val = resToVal(ctx.UsedResources().Cpu)
	case "memused": // Deprecated
		val = resToVal(ctx.UsedResources().Mem)
	case "status":
		val = statusValue(ctx.Status())
	case "parent":
		val = rt.NilValue
	case "flags":
		val = rt.StringValue(strings.Join(ctx.RequiredFlags().Names(), " "))
	}
	return c.PushingNext1(t.Runtime, val), nil
}

func context__tostring(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	ctx, err := contextArg(c, 0)
	if err != nil {
		return nil, err
	}
	return c.PushingNext1(t.Runtime, statusValue(ctx.Status())), nil
}

func resources__index(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	res, err := resourcesArg(c, 0)
	if err != nil {
		return nil, err
	}
	key, err := c.StringArg(1)
	if err != nil {
		return nil, err
	}
	var n uint64
	switch key {
	case "cpu":
		n = res.Cpu
	case "memory":
		n = res.Mem
	case "time":
		n = res.Time
	default:
		// We'll return nil
	}
	val := rt.NilValue
	if n > 0 {
		val = resToVal(n)
	}
	return c.PushingNext1(t.Runtime, val), nil
}

func resources__tostring(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	res, err := resourcesArg(c, 0)
	if err != nil {
		return nil, err
	}
	vals := make([]string, 0, 3)
	if res.Cpu > 0 {
		vals = append(vals, fmt.Sprintf("cpu=%d", res.Cpu))
	}
	if res.Mem > 0 {
		vals = append(vals, fmt.Sprintf("mem=%d", res.Mem))
	}
	if res.Time > 0 {
		vals = append(vals, fmt.Sprintf("time=%d", res.Time))
	}
	s := "[" + strings.Join(vals, ",") + "]"
	t.RequireBytes(len(s))
	return c.PushingNext1(t.Runtime, rt.StringValue(s)), nil
}

func resToVal(v uint64) rt.Value {
	return rt.IntValue(int64(v))
}

func statusValue(st rt.RuntimeContextStatus) rt.Value {
	s := st.String()
	if s == "" {
		return rt.NilValue
	}
	return rt.StringValue(s)
}
