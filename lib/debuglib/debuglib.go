package debuglib

import (
	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
)

func Load(r *rt.Runtime) {
	pkg := rt.NewTable()
	rt.SetEnv(r.GlobalEnv(), "debug", pkg)
	rt.SetEnvGoFunc(pkg, "getinfo", getinfo, 3, false)
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
		for idx > 1 && cont != nil {
			cont = cont.Next()
			idx++
		}
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
