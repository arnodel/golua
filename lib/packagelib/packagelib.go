package packagelib

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	rt "github.com/arnodel/golua/runtime"
)

var (
	pkgKey       = rt.StringValue("package")
	preloadKey   = rt.StringValue("preload")
	pathKey      = rt.StringValue("path")
	configKey    = rt.StringValue("config")
	loadedKey    = rt.StringValue("loaded")
	searchersKey = rt.StringValue("searchers")
)

const defaultPath = `./?.lua;./?/init.lua`

// Loader is used to register libraries
type Loader struct {
	// Function that creates the package and returns it
	Load func(r *rt.Runtime) (rt.Value, func())

	// Function that cleans up at the end (optional)
	Cleanup func(r *rt.Runtime)

	// Name of the package
	Name string
}

// Run will create the package, associate it with its name in the global env and
// cache it.
func (l Loader) Run(r *rt.Runtime) func() {
	pkg, cleanup := l.Load(r)
	if l.Name == "" || pkg.IsNil() {
		return cleanup
	}
	r.SetEnv(r.GlobalEnv(), l.Name, pkg)
	err := savePackage(r, l.Name, pkg)
	if err != nil {
		panic(fmt.Sprintf("Unable to load %s: %s", l.Name, err))
	}
	return cleanup
}

// LibLoader allows loading the package lib.
var LibLoader = Loader{
	Load: load,
	Name: "package",
}

func load(r *rt.Runtime) (rt.Value, func()) {
	env := r.GlobalEnv()
	pkg := rt.NewTable()
	pkgVal := rt.TableValue(pkg)
	r.SetRegistry(pkgKey, pkgVal)
	r.SetTable(pkg, loadedKey, rt.TableValue(rt.NewTable()))
	r.SetTable(pkg, preloadKey, rt.TableValue(rt.NewTable()))
	searchers := rt.NewTable()
	r.SetTable(searchers, rt.IntValue(1), rt.FunctionValue(searchPreloadGoFunc))
	r.SetTable(searchers, rt.IntValue(2), rt.FunctionValue(searchLuaGoFunc))
	r.SetTable(pkg, searchersKey, rt.TableValue(searchers))
	r.SetTable(pkg, pathKey, rt.StringValue(defaultPath))
	r.SetTable(pkg, configKey, rt.StringValue(defaultConfig.String()))

	r.SetEnvGoFunc(pkg, "searchpath", searchpath, 4, false)
	r.SetEnvGoFunc(env, "require", require, 1, false)

	return pkgVal, nil
}

type config struct {
	dirSep                 string
	pathSep                string
	placeholder            string
	windowsExecPlaceholder string
	suffixSep              string
}

func (c *config) String() string {
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
		c.dirSep, c.pathSep, c.placeholder,
		c.windowsExecPlaceholder, c.suffixSep)
}

var defaultConfig = config{"/", ";", "?", "!", "-"}

func getConfig(pkg *rt.Table) *config {
	conf := new(config)
	*conf = defaultConfig
	confStr, ok := pkg.Get(configKey).TryString()
	if !ok {
		return conf
	}
	lines := strings.Split(string(confStr), "\n")
	if len(lines) >= 1 {
		conf.dirSep = lines[0]
	}
	if len(lines) >= 2 {
		conf.pathSep = lines[1]
	}
	if len(lines) >= 3 {
		conf.placeholder = lines[2]
	}
	if len(lines) >= 4 {
		conf.windowsExecPlaceholder = lines[3]
	}
	if len(lines) >= 5 {
		conf.suffixSep = lines[4]
	}
	return conf
}

func require(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	name, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	nameVal := c.Arg(0)
	pkg := pkgTable(t.Runtime)

	// First check is the module is already loaded
	loaded, ok := pkg.Get(loadedKey).TryTable()
	if !ok {
		return nil, errors.New("package.loaded must be a table")
	}
	next := c.Next()
	if mod := loaded.Get(nameVal); !mod.IsNil() {
		t.Push1(next, mod)
		return next, nil
	}

	// If not, then go through the searchers
	searchers, ok := pkg.Get(searchersKey).TryTable()
	if !ok {
		return nil, errors.New("package.searchers must be a table")
	}

	for i := int64(1); ; i++ {
		searcher := searchers.Get(rt.IntValue(i))
		if searcher.IsNil() {
			err = fmt.Errorf("could not find package '%s'", name)
			break
		}
		res := rt.NewTerminationWith(c, 2, false)
		if err = rt.Call(t, searcher, []rt.Value{nameVal}, res); err != nil {
			break
		}
		loader := res.Get(0)
		// We got a loader, so call it
		if _, ok := loader.TryCallable(); ok {
			val := res.Get(1)
			res = rt.NewTerminationWith(c, 2, false)
			if err = rt.Call(t, loader, []rt.Value{nameVal, val}, res); err != nil {
				break
			}
			mod := rt.BoolValue(true)
			if r0 := res.Get(0); !r0.IsNil() {
				mod = r0
			}
			t.SetTable(loaded, nameVal, mod)
			t.Push1(next, mod)
			return next, nil
		}
	}
	return nil, err
}

func searchpath(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	var (
		name, path string
		sep        = "."
		conf       = *getConfig(pkgTable(t.Runtime))
		rep        = conf.dirSep
	)

	err := c.CheckNArgs(2)
	if err == nil {
		name, err = c.StringArg(0)
	}
	if err == nil {
		path, err = c.StringArg(1)
	}
	if err == nil && c.NArgs() >= 3 {
		sep, err = c.StringArg(2)
	}
	if err == nil && c.NArgs() >= 4 {
		rep, err = c.StringArg(3)
	}
	if err != nil {
		return nil, err
	}
	conf.dirSep = string(rep)
	found, templates := searchPath(string(name), string(path), string(sep), &conf)
	next := c.Next()
	if found != "" {
		t.Push1(next, rt.StringValue(found))
	} else {
		t.Push(next, rt.NilValue, rt.StringValue("tried: "+strings.Join(templates, "\n")))
	}
	return next, nil
}

func searchPath(name, path, dot string, conf *config) (string, []string) {
	namePath := strings.Replace(name, dot, conf.dirSep, -1)
	templates := strings.Split(path, conf.pathSep)
	for i, template := range templates {
		searchpath := strings.Replace(template, conf.placeholder, namePath, -1)
		f, err := os.Open(searchpath)
		f.Close()
		if err == nil {
			return searchpath, nil
		}
		templates[i] = searchpath
	}
	return "", templates
}

func searchPreload(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	s, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	loader := pkgTable(t.Runtime).Get(preloadKey).AsTable().Get(rt.StringValue(s))
	return c.PushingNext1(t.Runtime, loader), nil
}

func searchLua(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	s, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	pkg := pkgTable(t.Runtime)
	path, ok := pkg.Get(pathKey).TryString()
	if !ok {
		return nil, errors.New("package.path must be a string")
	}
	conf := getConfig(pkg)
	found, templates := searchPath(string(s), string(path), ".", conf)
	next := c.Next()
	if found == "" {
		t.Push1(next, rt.StringValue(strings.Join(templates, "\n")))
	} else {
		t.Push1(next, rt.FunctionValue(loadLuaGoFunc))
		t.Push1(next, rt.StringValue(found))
	}
	return next, nil
}

var (
	loadLuaGoFunc       = rt.NewGoFunction(loadLua, "loadlua", 2, false)
	searchLuaGoFunc     = rt.NewGoFunction(searchLua, "searchlua", 1, false)
	searchPreloadGoFunc = rt.NewGoFunction(searchPreload, "searchpreload", 1, false)
)

func loadLua(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}
	// Arg 0 is the module name - dunno what to do with it.
	filePath, err := c.StringArg(1)
	if err != nil {
		return nil, err
	}
	src, readErr := ioutil.ReadFile(string(filePath))
	if readErr != nil {
		return nil, fmt.Errorf("error reading file: %s", readErr)
	}
	clos, compErr := t.LoadFromSourceOrCode(string(filePath), src, "bt", rt.TableValue(t.GlobalEnv()), true)
	if compErr != nil {
		return nil, fmt.Errorf("error compiling file: %s", compErr)
	}
	return rt.Continue(t, rt.FunctionValue(clos), c.Next())
}

func pkgTable(r *rt.Runtime) *rt.Table {
	return r.Registry(pkgKey).AsTable()
}

func savePackage(r *rt.Runtime, name string, val rt.Value) error {
	pkg := pkgTable(r)

	// First check is the module is already loaded
	loaded, ok := pkg.Get(loadedKey).TryTable()
	if !ok {
		return errors.New("package.loaded must be a table")
	}
	r.SetTable(loaded, rt.StringValue(name), val)
	return nil
}
