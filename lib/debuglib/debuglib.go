package debuglib

import (
	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
)

// LibLoader can load the debug lib.
var LibLoader = packagelib.Loader{
	Load: load,
	Name: "debug",
}

func load(r *rt.Runtime) rt.Value {
	pkg := rt.NewTable()
	pkgVal := rt.TableValue(pkg)
	r.SetEnv(r.GlobalEnv(), "debug", pkgVal)
	r.SetEnvGoFunc(pkg, "getinfo", getinfo, 3, false)
	r.SetEnvGoFunc(pkg, "getupvalue", getupvalue, 2, false)
	r.SetEnvGoFunc(pkg, "setupvalue", setupvalue, 3, false)
	r.SetEnvGoFunc(pkg, "upvaluejoin", upvaluejoin, 4, false)
	r.SetEnvGoFunc(pkg, "setmetatable", setmetatable, 2, false)
	r.SetEnvGoFunc(pkg, "upvalueid", upvalueid, 2, false)
	return pkgVal
}

func getinfo(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	var (
		thread *rt.Thread = t
		idx    int64
		cont   rt.Cont
		what   string
		fIdx   int
	)
	thread, ok := c.Arg(0).TryThread()
	if !ok {
		thread = t
	}
	if c.NArgs() < 1+fIdx {
		return nil, rt.NewErrorS("missing argument: f")
	}
	switch arg := c.Arg(fIdx); arg.Type() {
	case rt.IntType:
		idx = arg.AsInt()
	case rt.FunctionType:
		term := rt.NewTerminationWith(0, false)
		cont = arg.AsFunction().Continuation(t.Runtime, term)
	case rt.FloatType:
		var tp rt.NumberType
		idx, tp = rt.FloatToInt(arg.AsFloat())
		if tp != rt.IsInt {
			return nil, rt.NewErrorS("f should be an integer or function").AddContext(c)
		}
	default:
		return nil, rt.NewErrorS("f should be an integer or function").AddContext(c)
	}
	if cont == nil {
		cont = thread.CurrentCont()
	}
	for idx > 0 && cont != nil {
		cont = cont.Next()
		idx--
	}
	// TODO: support what arg
	_ = what
	next := c.Next()
	if cont == nil {
		next.Push(rt.NilValue)
	} else if info := cont.DebugInfo(); info == nil {
		next.Push(rt.NilValue)
	} else {
		res := rt.NewTable()
		t.SetEnv(res, "name", rt.StringValue(info.Name))
		t.SetEnv(res, "currentline", rt.IntValue(int64(info.CurrentLine)))
		t.SetEnv(res, "source", rt.StringValue(info.Source))
		next.Push(rt.TableValue(res))
	}
	return next, nil
}

func getupvalue(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err.AddContext(c)
	}
	f, err := c.ClosureArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	upv, err := c.IntArg(1)
	if err != nil {
		return nil, err.AddContext(c)
	}
	up := int(upv) - 1
	next := c.Next()
	if up < 0 || up >= int(f.Code.UpvalueCount) {
		next.Push(rt.NilValue)
	} else {
		rt.Push(next, rt.StringValue(f.Code.UpNames[up]), f.GetUpvalue(up))
	}
	return next, nil
}

func setupvalue(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(3); err != nil {
		return nil, err.AddContext(c)
	}
	f, err := c.ClosureArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	upv, err := c.IntArg(1)
	if err != nil {
		return nil, err.AddContext(c)
	}
	up := int(upv) - 1
	next := c.Next()
	if up < 0 || up >= int(f.Code.UpvalueCount) {
		next.Push(rt.NilValue)
	} else {
		next.Push(rt.StringValue(f.Code.UpNames[up]))
		f.SetUpvalue(up, c.Arg(2))
	}
	return next, nil
}

func upvaluejoin(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(4); err != nil {
		return nil, err.AddContext(c)
	}
	f1, err := c.ClosureArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	upv1, err := c.IntArg(1)
	if err != nil {
		return nil, err.AddContext(c)
	}
	f2, err := c.ClosureArg(2)
	if err != nil {
		return nil, err.AddContext(c)
	}
	upv2, err := c.IntArg(3)
	if err != nil {
		return nil, err.AddContext(c)
	}
	up1 := int(upv1) - 1
	up2 := int(upv2) - 1
	if up1 < 0 || up1 >= int(f1.Code.UpvalueCount) || up2 < 0 || up2 >= int(f2.Code.UpvalueCount) {
		return nil, rt.NewErrorS("Invalid upvalue index").AddContext(c)
	}
	f1.Upvalues[up1] = f2.Upvalues[up2]
	return c.Next(), nil
}

func upvalueid(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err.AddContext(c)
	}
	f, err := c.ClosureArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	upv, err := c.IntArg(1)
	if err != nil {
		return nil, err.AddContext(c)
	}
	up := int(upv) - 1
	if up < 0 || up >= int(f.Code.UpvalueCount) {
		return nil, rt.NewErrorS("Invalid upvalue index").AddContext(c)
	}
	return c.PushingNext(rt.LightUserDataValue(rt.LightUserData{Data: f.Upvalues[up]})), nil
}

func setmetatable(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var err *rt.Error
	if err = c.CheckNArgs(2); err != nil {
		return nil, err.AddContext(c)
	}
	v := c.Arg(0)
	var meta *rt.Table
	if !c.Arg(1).IsNil() {
		meta, err = c.TableArg(1)
		if err != nil {
			return nil, err.AddContext(c)
		}
	}
	t.SetRawMetatable(v, meta)
	return c.PushingNext(v), nil
}
