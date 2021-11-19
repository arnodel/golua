package runtimelib

import rt "github.com/arnodel/golua/runtime"

type contextMetaKeyType struct{}

var contextMetaKey = rt.AsValue(contextMetaKeyType{})

func createContextMetatable(r *rt.Runtime) {
	meta := rt.NewTable()
	r.SetEnvGoFunc(meta, "__index", context__index, 2, false)
	r.SetEnvGoFunc(meta, "__tostring", context__tostring, 1, false)
	r.SetRegistry(contextMetaKey, rt.AsValue(meta))
}

func newContextUserData(r *rt.Runtime, ctx rt.RuntimeContext) *rt.UserData {
	meta := r.Registry(contextMetaKey).AsTable()
	return rt.NewUserData(ctx, meta)
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

func contextArg(c *rt.GoCont, n int) (rt.RuntimeContext, *rt.Error) {
	ctx, ok := valueToContext(c.Arg(n))
	if ok {
		return ctx, nil
	}
	return nil, rt.NewErrorF("#%d must be a runtime context", n+1)
}

func context__index(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	ctx, err := contextArg(c, 0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	key, err := c.StringArg(1)
	if err != nil {
		return nil, err.AddContext(c)
	}
	val := rt.NilValue
	switch key {
	case "cpulimit":
		{
			limit := ctx.CpuLimit()
			if limit > 0 {
				val = resToVal(limit)
			}
		}
	case "memlimit":
		{
			limit := ctx.MemLimit()
			if limit > 0 {
				val = resToVal(limit)
			}
		}
	case "cpuused":
		val = resToVal(ctx.CpuUsed())
	case "memused":
		val = resToVal(ctx.MemUsed())
	case "status":
		val = statusValue(ctx.Status())
	case "parent":
		val = rt.NilValue
	case "io":
		if rt.RCF_NoIO.IsSet(ctx) {
			val = rt.StringValue("off")
		} else {
			val = rt.StringValue("on")
		}
	}
	return c.PushingNext1(t.Runtime, val), nil
}

func context__tostring(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	ctx, err := contextArg(c, 0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	return c.PushingNext1(t.Runtime, statusValue(ctx.Status())), nil
}

func resToVal(v uint64) rt.Value {
	return rt.IntValue(int64(v))
}

func statusValue(st rt.RuntimeContextStatus) rt.Value {
	switch st {
	case rt.RCS_Live:
		return rt.StringValue(liveStatusString)
	case rt.RCS_Done:
		return rt.StringValue(doneStatusString)
	case rt.RCS_Killed:
		return rt.StringValue(killedStatusString)
	default:
		return rt.NilValue
	}
}

const (
	liveStatusString   = "live"
	doneStatusString   = "done"
	killedStatusString = "killed"
)
