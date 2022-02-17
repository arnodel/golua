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

func newContextValue(r *rt.Runtime, ctx rt.RuntimeContext) rt.Value {
	return r.NewUserDataValue(ctx, getRegistry(r).contextMeta)
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

func newResourcesValue(r *rt.Runtime, res rt.RuntimeResources) rt.Value {
	return r.NewUserDataValue(res, getRegistry(r).resourcesMeta)
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

func contextArg(c *rt.GoCont, n int) (rt.RuntimeContext, error) {
	ctx, ok := valueToContext(c.Arg(n))
	if ok {
		return ctx, nil
	}
	return nil, fmt.Errorf("#%d must be a runtime context", n+1)
}

func optContextArg(t *rt.Thread, c *rt.GoCont, n int) (rt.RuntimeContext, error) {
	if n >= c.NArgs() {
		return t.RuntimeContext(), nil
	}
	return contextArg(c, n)
}

func resourcesArg(c *rt.GoCont, n int) (rt.RuntimeResources, error) {
	res, ok := valueToResources(c.Arg(n))
	if ok {
		return res, nil
	}
	return res, fmt.Errorf("#%d must be a runtime resources", n+1)
}

func context__index(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
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
	case "status":
		val = statusValue(ctx.Status())
	case "parent":
		val = rt.NilValue
	case "flags":
		val = rt.StringValue(strings.Join(ctx.RequiredFlags().Names(), " "))
	case "due":
		val = rt.BoolValue(ctx.Due())
	case "killnow":
		val = rt.FunctionValue(killnowGoF)
	case "stopnow":
		val = rt.FunctionValue(stopnowGoF)
	}
	return c.PushingNext1(t.Runtime, val), nil
}

func context__tostring(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	ctx, err := contextArg(c, 0)
	if err != nil {
		return nil, err
	}
	return c.PushingNext1(t.Runtime, statusValue(ctx.Status())), nil
}

func resources__index(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	res, err := resourcesArg(c, 0)
	if err != nil {
		return nil, err
	}
	key, err := c.StringArg(1)
	if err != nil {
		return nil, err
	}
	val := rt.NilValue
	switch key {
	case cpuName:
		n := res.Cpu
		if n > 0 {
			val = resToVal(n)
		}
	case memoryName:
		n := res.Memory
		if n > 0 {
			val = resToVal(n)
		}
	case secondsName:
		n := res.Millis
		if n > 0 {
			val = rt.FloatValue(float64(n) / 1000)
		}
	case millisName:
		n := res.Millis
		if n > 0 {
			val = rt.FloatValue(float64(n))
		}
	default:
		// We'll return nil
	}
	return c.PushingNext1(t.Runtime, val), nil
}

func resources__tostring(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	res, err := resourcesArg(c, 0)
	if err != nil {
		return nil, err
	}
	vals := make([]string, 0, 3)
	if res.Cpu > 0 {
		vals = append(vals, fmt.Sprintf("%s=%d", cpuName, res.Cpu))
	}
	if res.Memory > 0 {
		vals = append(vals, fmt.Sprintf("%s=%d", memoryName, res.Memory))
	}
	if res.Millis > 0 {
		vals = append(vals, fmt.Sprintf("%s=%g", secondsName, float64(res.Millis)/1000))
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

func killnow(t *rt.Thread, c *rt.GoCont) (next rt.Cont, err error) {
	ctx, err := optContextArg(t, c, 0)
	if err != nil {
		return nil, err
	}
	ctx.SetStopLevel(rt.HardStop)
	return nil, nil
}

func stopnow(t *rt.Thread, c *rt.GoCont) (next rt.Cont, err error) {
	ctx, err := optContextArg(t, c, 0)
	if err != nil {
		return nil, err
	}
	ctx.SetStopLevel(rt.SoftStop)
	return c.Next(), nil
}

func due(t *rt.Thread, c *rt.GoCont) (next rt.Cont, retErr error) {
	ctx, err := optContextArg(t, c, 0)
	if err != nil {
		return nil, err
	}
	return c.PushingNext1(t.Runtime, rt.BoolValue(ctx.Due())), nil
}

var (
	killnowGoF = rt.NewGoFunction(killnow, "killnow", 1, false)
	stopnowGoF = rt.NewGoFunction(stopnow, "stopnow", 1, false)
	dueGoF     = rt.NewGoFunction(due, "due", 1, false)
)

func init() {
	rt.SolemnlyDeclareCompliance(
		rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyTimeSafe|rt.ComplyIoSafe,
		killnowGoF,
		stopnowGoF,
		dueGoF,
	)
}
