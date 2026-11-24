package golib

import (
	"errors"
	"fmt"
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

// TODO: a better value?
var govalueKey = rt.AsValue(govalueKeyType{})

func load(r *rt.Runtime) (rt.Value, func()) {
	pkg := rt.NewTable()

	if goimports.Supported {
		r.SetEnvGoFunc(pkg, "import", goimport, 2, false)
	}

	meta := rt.NewTable()
	r.SetEnvGoFunc(meta, "__index", goValueIndex, 2, false)
	r.SetEnvGoFunc(meta, "__newindex", goValueSetIndex, 3, false)
	r.SetEnvGoFunc(meta, "__call", goValueCall, 1, true)
	r.SetEnvGoFunc(meta, "__tostring", goValueToString, 1, false)

	r.SetRegistry(govalueKey, rt.TableValue(meta))

	return rt.TableValue(pkg), nil
}

func getMeta(r *rt.Runtime) *rt.Table {
	return r.Registry(govalueKey).AsTable()
}

// NewGoValue will return a UserData representing the go value.
func NewGoValue(r *rt.Runtime, x interface{}) rt.Value {
	return r.NewUserDataValue(x, getMeta(r))
}

func goValueToString(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	u, err := c.UserDataArg(0)
	if err != nil {
		return nil, err
	}
	return c.PushingNext1(t.Runtime, rt.StringValue(fmt.Sprintf("%#v", u.Value()))), nil
}

func goValueIndex(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}
	u, err := c.UserDataArg(0)
	if err != nil {
		return nil, err
	}
	val, indexErr := goIndex(t, u, c.Arg(1))
	if indexErr != nil {
		return nil, indexErr
	}
	return c.PushingNext1(t.Runtime, val), nil
}

func goValueSetIndex(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.CheckNArgs(3); err != nil {
		return nil, err
	}
	u, err := c.UserDataArg(0)
	if err != nil {
		return nil, err
	}
	setIndexErr := goSetIndex(t, u, c.Arg(1), c.Arg(2))
	if setIndexErr != nil {
		return nil, setIndexErr
	}
	return c.Next(), nil
}

func goValueCall(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	u, err := c.UserDataArg(0)
	if err != nil {
		return nil, err
	}
	res, callErr := goCall(t, u, c.Etc())
	if callErr != nil {
		return nil, callErr
	}
	return c.PushingNext(t.Runtime, res...), nil
}

func goimport(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	path, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	root, err := pluginsRoot()
	if err != nil {
		return nil, fmt.Errorf("cannot import go packages: %s", err)
	}
	forceBuild := c.NArgs() >= 2 && rt.Truth(c.Arg(1))
	exports, loadErr := goimports.LoadGoPackage(string(path), root, forceBuild)
	if loadErr != nil {
		return nil, fmt.Errorf("cannot import go package %s: %s", path, loadErr)
	}
	return c.PushingNext1(t.Runtime, NewGoValue(t.Runtime, exports)), nil
}

// pluginsRoot returns the directory the plugins of Go packages are kept in:
// the one GOLUA_PLUGINS_ROOT names, or else .golua/goplugins in the user's
// home directory.  It is looked up at each import rather than once when the
// program starts, so that a program embedding golua can set
// GOLUA_PLUGINS_ROOT itself before its Lua code imports a package.
func pluginsRoot() (string, error) {
	if root, ok := os.LookupEnv("GOLUA_PLUGINS_ROOT"); ok {
		if root == "" {
			return "", errors.New("GOLUA_PLUGINS_ROOT is empty")
		}
		return root, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("GOLUA_PLUGINS_ROOT is not set and there is no home directory (%s)", err)
	}
	return path.Join(home, ".golua/goplugins"), nil
}
