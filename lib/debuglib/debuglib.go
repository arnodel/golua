package debuglib

import (
	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
)

func Load(r *rt.Runtime) {
	pkg := rt.NewTable()
	rt.SetEnv(r.GlobalEnv(), "debug", pkg)
	rt.SetEnvGoFunc(pkg, "getinfo", getinfo, 3, false)
	rt.SetEnvGoFunc(pkg, "getupvalue", getupvalue, 2, false)
	rt.SetEnvGoFunc(pkg, "setupvalue", setupvalue, 3, false)
	rt.SetEnvGoFunc(pkg, "upvaluejoin", upvaluejoin, 4, false)
	_ = packagelib.SavePackage(r.MainThread(), rt.String("debug"), pkg)
}

func getinfo(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	var (
		thread *rt.Thread = t
		idx    rt.Int
		cont   rt.Cont
		what   rt.String
		fIdx   int
	)
	thread, ok := c.Arg(0).(*rt.Thread)
	if !ok {
		thread = t
	}
	if c.NArgs() < 1+fIdx {
		return nil, rt.NewErrorS("missing argument: f")
	}
	switch arg := c.Arg(fIdx).(type) {
	case rt.Int:
		idx = arg
	case rt.Callable:
		term := rt.NewTerminationWith(0, false)
		cont = arg.Continuation(term)
	case rt.Float:
		var tp rt.NumberType
		idx, tp = arg.ToInt()
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
		next.Push(nil)
	} else if info := cont.DebugInfo(); info == nil {
		next.Push(nil)
	} else {
		res := rt.NewTable()
		rt.SetEnv(res, "name", rt.String(info.Name))
		rt.SetEnv(res, "currentline", rt.Int(info.CurrentLine))
		rt.SetEnv(res, "source", rt.String(info.Source))
		next.Push(res)
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
		next.Push(nil)
	} else {
		rt.Push(next, rt.String(f.Code.UpNames[up]), f.GetUpvalue(up))
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
		next.Push(nil)
	} else {
		next.Push(rt.String(f.Code.UpNames[up]))
		f.SetUpValue(up, c.Arg(2))
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
