package golib

import (
	"github.com/arnodel/golua/lib/golib/goimports"
	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
)

// LibLoader loads this library.
var LibLoader = packagelib.Loader{
	Load: load,
	Name: "golib",
}

type govalueKeyType struct{}

var govalueKey = govalueKeyType{}

func load(r *rt.Runtime) rt.Value {
	pkg := rt.NewTable()
	rt.SetEnvGoFunc(pkg, "import", goimport, 1, false)

	meta := rt.NewTable()
	rt.SetEnvGoFunc(meta, "__index", goValueIndex, 2, false)
	rt.SetEnvGoFunc(meta, "__newindex", goValueSetIndex, 3, false)
	rt.SetEnvGoFunc(meta, "__call", goValueCall, 1, true)

	r.SetRegistry(govalueKey, meta)

	return pkg
}

// NewGoValue will return a UserData representing the go value.
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
	val, indexErr := gv.Index(c.Arg(1), meta)
	if indexErr != nil {
		return nil, rt.NewErrorE(indexErr).AddContext(c)
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
	setIndexErr := gv.SetIndex(c.Arg(1), c.Arg(2))
	if setIndexErr != nil {
		return nil, rt.NewErrorE(setIndexErr).AddContext(c)
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
	res, callErr := gv.Call(c.Etc(), meta)
	if callErr != nil {
		return nil, rt.NewErrorE(callErr).AddContext(c)
	}
	return c.PushingNext(res...), nil
}

// GoValueArg turns a continuation argument into a *File.
func GoValueArg(c *rt.GoCont, n int) (GoValue, *rt.Table, *rt.Error) {
	f, meta, ok := ValueToGoValue(c.Arg(n))
	if ok {
		return f, meta, nil
	}
	return GoValue{}, nil, rt.NewErrorF("#%d must be a go value", n+1)
}

// ValueToGoValue turns a lua value to a *File if possible.
func ValueToGoValue(v rt.Value) (GoValue, *rt.Table, bool) {
	u, ok := v.(*rt.UserData)
	var goVal GoValue
	if ok {
		goVal, ok = u.Value().(GoValue)
	}
	if !ok {
		return goVal, nil, false
	}
	return goVal, u.Metatable(), true
}

func goimport(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	path, err := c.StringArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	exports, loadErr := goimports.LoadGoPackage(string(path), "/Users/adelobelle/goplugins")
	if loadErr != nil {
		return nil, rt.NewErrorF("cannot import go package %s: %s", path, loadErr)
	}
	return c.PushingNext(NewGoValue(t.Runtime, exports)), nil
}
