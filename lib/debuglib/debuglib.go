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

	rt.SolemnlyDeclareCompliance(
		rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyIoSafe,

		r.SetEnvGoFunc(pkg, "getinfo", getinfo, 3, false),
		r.SetEnvGoFunc(pkg, "getupvalue", getupvalue, 2, false),
		r.SetEnvGoFunc(pkg, "setupvalue", setupvalue, 3, false),
		r.SetEnvGoFunc(pkg, "upvaluejoin", upvaluejoin, 4, false),
		r.SetEnvGoFunc(pkg, "setmetatable", setmetatable, 2, false),
		r.SetEnvGoFunc(pkg, "traceback", traceback, 3, false),
		r.SetEnvGoFunc(pkg, "upvalueid", upvalueid, 2, false),
	)

	return pkgVal
}

func getinfo(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	var (
		thread *rt.Thread
		idx    int64
		cont   rt.Cont
		what   string
		fIdx   int
	)
	thread, ok := c.Arg(0).TryThread()
	if !ok {
		thread = t
	} else {
		fIdx = 1
	}
	if c.NArgs() < 1+fIdx {
		return nil, rt.NewErrorS("missing argument: f")
	}
	switch arg := c.Arg(fIdx); arg.Type() {
	case rt.IntType:
		idx = arg.AsInt()
	case rt.FunctionType:
		term := rt.NewTerminationWith(c, 0, false)
		cont = arg.AsFunction().Continuation(t.Runtime, term)
	case rt.FloatType:
		var tp rt.NumberType
		idx, tp = rt.FloatToInt(arg.AsFloat())
		if tp != rt.IsInt {
			return nil, rt.NewErrorS("f should be an integer or function")
		}
	default:
		return nil, rt.NewErrorS("f should be an integer or function")
	}
	if cont == nil {
		cont = thread.CurrentCont()
	}
	for idx > 0 && cont != nil {
		cont = cont.Parent()
		idx--
	}
	// TODO: support what arg
	_ = what
	next := c.Next()
	if cont == nil {
		t.Push1(next, rt.NilValue)
	} else if info := cont.DebugInfo(); info == nil {
		t.Push1(next, rt.NilValue)
	} else {
		res := rt.NewTable()
		t.SetEnv(res, "name", rt.StringValue(info.Name))
		t.SetEnv(res, "currentline", rt.IntValue(int64(info.CurrentLine)))
		t.SetEnv(res, "source", rt.StringValue(info.Source))
		t.Push1(next, rt.TableValue(res))
	}
	return next, nil
}

func getupvalue(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}
	f, err := c.ClosureArg(0)
	if err != nil {
		return nil, err
	}
	upv, err := c.IntArg(1)
	if err != nil {
		return nil, err
	}
	up := int(upv) - 1
	next := c.Next()
	if up >= 0 && up < int(f.Code.UpvalueCount) {
		t.Push(next, rt.StringValue(f.Code.UpNames[up]), f.GetUpvalue(up))
	}
	return next, nil
}

func setupvalue(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(3); err != nil {
		return nil, err
	}
	f, err := c.ClosureArg(0)
	if err != nil {
		return nil, err
	}
	upv, err := c.IntArg(1)
	if err != nil {
		return nil, err
	}
	up := int(upv) - 1
	next := c.Next()
	if up >= 0 && up < int(f.Code.UpvalueCount) {
		t.Push1(next, rt.StringValue(f.Code.UpNames[up]))
		f.SetUpvalue(up, c.Arg(2))
	}
	return next, nil
}

func upvaluejoin(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(4); err != nil {
		return nil, err
	}
	f1, err := c.ClosureArg(0)
	if err != nil {
		return nil, err
	}
	upv1, err := c.IntArg(1)
	if err != nil {
		return nil, err
	}
	f2, err := c.ClosureArg(2)
	if err != nil {
		return nil, err
	}
	upv2, err := c.IntArg(3)
	if err != nil {
		return nil, err
	}
	up1 := int(upv1) - 1
	up2 := int(upv2) - 1
	if up1 < 0 || up1 >= int(f1.Code.UpvalueCount) || up2 < 0 || up2 >= int(f2.Code.UpvalueCount) {
		return nil, rt.NewErrorS("Invalid upvalue index")
	}
	f1.Upvalues[up1] = f2.Upvalues[up2]
	return c.Next(), nil
}

func upvalueid(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}
	f, err := c.ClosureArg(0)
	if err != nil {
		return nil, err
	}
	upv, err := c.IntArg(1)
	if err != nil {
		return nil, err
	}
	up := int(upv) - 1
	if up < 0 || up >= int(f.Code.UpvalueCount) {
		return nil, rt.NewErrorS("Invalid upvalue index")
	}
	return c.PushingNext1(t.Runtime, rt.LightUserDataValue(rt.LightUserData{Data: f.Upvalues[up]})), nil
}

func setmetatable(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var err *rt.Error
	if err = c.CheckNArgs(2); err != nil {
		return nil, err
	}
	v := c.Arg(0)
	var meta *rt.Table
	if !c.Arg(1).IsNil() {
		meta, err = c.TableArg(1)
		if err != nil {
			return nil, err
		}
	}
	t.SetRawMetatable(v, meta)
	return c.PushingNext1(t.Runtime, v), nil
}

func traceback(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var (
		cont            = t.CurrentCont()
		msgString       = ""
		nArgs           = c.NArgs()
		level     int64 = 1
	)
	if nArgs > 0 {
		msgIndex := 0
		arg0 := c.Arg(0)
		if arg0.Type() == rt.ThreadType {
			cont = arg0.AsThread().CurrentCont()
			msgIndex = 1
		}
		if nArgs > msgIndex {
			msg := c.Arg(msgIndex)
			if !msg.IsNil() {
				var ok bool
				msgString, ok = msg.TryString()
				if !ok {
					return c.PushingNext1(t.Runtime, msg), nil
				}
			}
		}
		if nArgs > msgIndex+1 {
			var err *rt.Error
			level, err = c.IntArg(msgIndex + 1)
			if err != nil {
				return nil, err
			}
		}
	}
	for level > 0 && cont != nil {
		cont = cont.Next()
		level--
	}
	tb := rt.StringValue(t.Traceback(msgString, cont))
	return c.PushingNext1(t.Runtime, tb), nil
}

var Traceback = rt.NewGoFunction(traceback, "traceback", 3, false)
