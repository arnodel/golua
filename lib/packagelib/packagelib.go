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
	Load func(r *rt.Runtime) rt.Value

	// Function that cleans up at the end (optional)
	Cleanup func(r *rt.Runtime)

	// Name of the package
	Name string
}

// Run will create the package, associate it with its name in the global env and
// cache it.
func (l Loader) Run(r *rt.Runtime) {
	pkg := l.Load(r)
	if l.Name == "" || pkg.IsNil() {
		return
	}
	rt.SetEnv(r.GlobalEnv(), l.Name, pkg)
	err := savePackage(r.MainThread(), l.Name, pkg)
	if err != nil {
		panic("Unable to load " + l.Name)
	}
}

// LibLoader allows loading the package lib.
var LibLoader = Loader{
	Load: load,
	Name: "package",
}

func load(r *rt.Runtime) rt.Value {
	env := r.GlobalEnv()
	pkg := rt.NewTable()
	pkgVal := rt.TableValue(pkg)
	r.SetRegistry(pkgKey, pkgVal)
	pkg.Set(loadedKey, rt.TableValue(rt.NewTable()))
	pkg.Set(preloadKey, rt.TableValue(rt.NewTable()))
	searchers := rt.NewTable()
	searchers.Set(rt.IntValue(1), rt.FunctionValue(searchPreloadGoFunc))
	searchers.Set(rt.IntValue(2), rt.FunctionValue(searchLuaGoFunc))
	pkg.Set(searchersKey, rt.TableValue(searchers))
	pkg.Set(pathKey, rt.StringValue(defaultPath))
	pkg.Set(configKey, rt.StringValue(defaultConfig.String()))
	rt.SetEnvGoFunc(pkg, "searchpath", searchpath, 4, false)
	rt.SetEnvGoFunc(env, "require", require, 1, false)
	return pkgVal
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

func require(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	name, err := c.StringArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	nameVal := c.Arg(0)
	pkg := pkgTable(t)

	// First check is the module is already loaded
	loaded, ok := pkg.Get(loadedKey).TryTable()
	if !ok {
		return nil, rt.NewErrorS("package.loaded must be a table").AddContext(c)
	}
	next := c.Next()
	if mod := loaded.Get(nameVal); !mod.IsNil() {
		next.Push(mod)
		return next, nil
	}

	// If not, then go through the searchers
	searchers, ok := pkg.Get(searchersKey).TryTable()
	if !ok {
		return nil, rt.NewErrorS("package.searchers must be a table").AddContext(c)
	}

	for i := int64(1); ; i++ {
		searcher := searchers.Get(rt.IntValue(i))
		if rt.IsNil(searcher) {
			err = rt.NewErrorF("could not find package '%s'", name)
			break
		}
		res := rt.NewTerminationWith(2, false)
		if err = rt.Call(t, searcher, []rt.Value{nameVal}, res); err != nil {
			break
		}
		loader := res.Get(0)
		// We got a loader, so call it
		if _, ok := loader.TryCallable(); ok {
			val := res.Get(1)
			res = rt.NewTerminationWith(2, false)
			if err = rt.Call(t, loader, []rt.Value{nameVal, val}, res); err != nil {
				break
			}
			mod := rt.BoolValue(true)
			if r0 := res.Get(0); !r0.IsNil() {
				mod = r0
			}
			loaded.Set(nameVal, mod)
			next.Push(mod)
			return next, nil
		}
	}
	return nil, err.AddContext(c)
}

func searchpath(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var (
		name, path string
		sep        = "."
		conf       = *getConfig(pkgTable(t))
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
		return nil, err.AddContext(c)
	}
	conf.dirSep = string(rep)
	found, templates := searchPath(string(name), string(path), string(sep), &conf)
	next := c.Next()
	if found != "" {
		next.Push(rt.StringValue(found))
	} else {
		rt.Push(next, rt.NilValue, rt.StringValue("tried: "+strings.Join(templates, "\n")))
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

func searchPreload(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	s, err := c.StringArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	loader := pkgTable(t).Get(preloadKey).AsTable().Get(rt.StringValue(s))
	c.Next().Push(loader)
	return c.Next(), nil
}

func searchLua(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	s, err := c.StringArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	pkg := pkgTable(t)
	path, ok := pkg.Get(pathKey).TryString()
	if !ok {
		return nil, rt.NewErrorS("package.path must be a string").AddContext(c)
	}
	conf := getConfig(pkg)
	found, templates := searchPath(string(s), string(path), ".", conf)
	next := c.Next()
	if found == "" {
		next.Push(rt.StringValue(strings.Join(templates, "\n")))
	} else {
		next.Push(rt.FunctionValue(loadLuaGoFunc))
		next.Push(rt.StringValue(found))
	}
	return next, nil
}

var (
	loadLuaGoFunc       = rt.NewGoFunction(loadLua, "loadlua", 2, false)
	searchLuaGoFunc     = rt.NewGoFunction(searchLua, "searchlua", 1, false)
	searchPreloadGoFunc = rt.NewGoFunction(searchPreload, "searchpreload", 1, false)
)

func loadLua(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err.AddContext(c)
	}
	// Arg 0 is the module name - dunno what to do with it.
	filePath, err := c.StringArg(1)
	if err != nil {
		fmt.Printf("XXX %+v\n", c.Arg(1))
		return nil, err.AddContext(c)
	}
	src, readErr := ioutil.ReadFile(string(filePath))
	if readErr != nil {
		return nil, rt.NewErrorF("error reading file: %s", readErr)
	}
	clos, compErr := rt.CompileAndLoadLuaChunk(string(filePath), src, t.GlobalEnv())
	if compErr != nil {
		return nil, rt.NewErrorF("error compiling file: %s", compErr)
	}
	return rt.Continue(t, rt.FunctionValue(clos), c.Next())
}

func pkgTable(t *rt.Thread) *rt.Table {
	return t.Registry(pkgKey).AsTable()
}

func savePackage(t *rt.Thread, name string, val rt.Value) error {
	pkg := pkgTable(t)

	// First check is the module is already loaded
	loaded, ok := pkg.Get(loadedKey).TryTable()
	if !ok {
		return errors.New("package.loaded must be a table")
	}
	loaded.Set(rt.StringValue(name), val)
	return nil
}
