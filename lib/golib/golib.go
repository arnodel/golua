package golib

import (
	"fmt"
	"log"
	"os"
	"path"

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
	rt.SetEnvGoFunc(meta, "__tostring", goValueToString, 1, false)

	r.SetRegistry(govalueKey, meta)

	return pkg
}

func getMeta(r *rt.Runtime) *rt.Table {
	return r.Registry(govalueKey).(*rt.Table)
}

// NewGoValue will return a UserData representing the go value.
func NewGoValue(r *rt.Runtime, x interface{}) *rt.UserData {
	return rt.NewUserData(x, getMeta(r))
}

func goValueToString(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	u, err := c.UserDataArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	return c.PushingNext(rt.String(fmt.Sprintf("%#v", u.Value()))), nil
}

func goValueIndex(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err.AddContext(c)
	}
	u, err := c.UserDataArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	val, indexErr := goIndex(t, u, c.Arg(1))
	if indexErr != nil {
		return nil, rt.NewErrorE(indexErr).AddContext(c)
	}
	return c.PushingNext(val), nil
}

func goValueSetIndex(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(3); err != nil {
		return nil, err.AddContext(c)
	}
	u, err := c.UserDataArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	setIndexErr := goSetIndex(t, u, c.Arg(1), c.Arg(2))
	if setIndexErr != nil {
		return nil, rt.NewErrorE(setIndexErr).AddContext(c)
	}
	return c.Next(), nil
}

func goValueCall(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	u, err := c.UserDataArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	res, callErr := goCall(t, u, c.Etc())
	if callErr != nil {
		return nil, rt.NewErrorE(callErr).AddContext(c)
	}
	return c.PushingNext(res...), nil
}

func goimport(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if pluginsRoot == "" {
		return nil, rt.NewError("cannot import go packages: plugins root not set")
	}
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	path, err := c.StringArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	forceBuild := c.NArgs() >= 2 && rt.Truth(c.Arg(1))
	exports, loadErr := goimports.LoadGoPackage(string(path), pluginsRoot, forceBuild)
	if loadErr != nil {
		return nil, rt.NewErrorF("cannot import go package %s: %s", path, loadErr)
	}
	return c.PushingNext(NewGoValue(t.Runtime, exports)), nil
}

var pluginsRoot string

func init() {
	var ok bool
	pluginsRoot, ok = os.LookupEnv("GOLUA_PLUGINS_ROOT")
	if ok {
		return
	}
	home, err := os.UserHomeDir()
	if err == nil {
		pluginsRoot = path.Join(home, ".golua/goplugins")
		return
	}
	pluginsRoot = ""
	log.Print("Unable to set go plugins root")
}
