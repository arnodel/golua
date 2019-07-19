package golib

import (
	"log"

	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
)

var LibLoader = packagelib.Loader{
	Load: load,
	Name: "golib",
}

type govalueKeyType struct{}

var govalueKey = govalueKeyType{}

func load(r *rt.Runtime) rt.Value {
	meta := rt.NewTable()
	rt.SetEnvGoFunc(meta, "__index", goValueIndex, 2, false)
	rt.SetEnvGoFunc(meta, "__newindex", goValueSetIndex, 3, false)
	rt.SetEnvGoFunc(meta, "__call", goValueCall, 1, true)

	r.SetRegistry(govalueKey, meta)
	return nil
}

func NewGoValue(r *rt.Runtime, x interface{}) *rt.UserData {
	meta := r.Registry(govalueKey).(*rt.Table)
	return rt.NewUserData(ToGoValue(x), meta)
}

func goValueIndex(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err.AddContext(c)
	}
	gv, meta, err := GoValueArg(c, 0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	val, ok := gv.Index(c.Arg(1), meta)
	if !ok {
		return nil, rt.NewErrorF("unable to get index of go value").AddContext(c)
	}
	return c.PushingNext(val), nil
}

func goValueSetIndex(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(3); err != nil {
		return nil, err.AddContext(c)
	}
	gv, _, err := GoValueArg(c, 0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	ok := gv.SetIndex(c.Arg(1), c.Arg(2))
	if !ok {
		return nil, rt.NewErrorF("unable to set index of go value").AddContext(c)
	}
	return c.Next(), nil
}

func goValueCall(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	gv, meta, err := GoValueArg(c, 0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	res, ok := gv.Call(c.Etc(), meta)
	if !ok {
		return nil, rt.NewErrorF("unable to call go value").AddContext(c)
	}
	log.Print("XXX:", res)
	return c.PushingNext(res...), nil
}

// FileArg turns a continuation argument into a *File.
func GoValueArg(c *rt.GoCont, n int) (GoValue, *rt.Table, *rt.Error) {
	f, meta, ok := ValueToGoValue(c.Arg(n))
	if ok {
		return f, meta, nil
	}
	return GoValue{}, nil, rt.NewErrorF("#%d must be a go value", n+1)
}

// ValueToFile turns a lua value to a *File if possible.
func ValueToGoValue(v rt.Value) (GoValue, *rt.Table, bool) {
	u, ok := v.(*rt.UserData)
	var goVal GoValue
	if ok {
		goVal, ok = u.Value().(GoValue)
	}
	return goVal, u.Metatable(), ok
}
