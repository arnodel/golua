package packagelib

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	rt "github.com/arnodel/golua/runtime"
)

var pkgKey = rt.String("package")
var preloadKey = rt.String("preload")
var pathKey = rt.String("path")
var configKey = rt.String("config")
var loadedKey = rt.String("loaded")
var searchersKey = rt.String("searchers")

const defaultPath = `./?.lua;./?/init.lua`

// Loader is used to register libraries
type Loader struct {
	// Function that creates the package and returns it
	Load func(r *rt.Runtime) rt.Value

	// Name of the package
	Name string
}

// Run will create the package, associate it with its name in the global env and
// cache it.
func (l Loader) Run(r *rt.Runtime) {
	pkg := l.Load(r)
	if l.Name == "" || pkg == nil {
		return
	}
	rt.SetEnv(r.GlobalEnv(), l.Name, pkg)
	err := SavePackage(r.MainThread(), rt.String(l.Name), pkg)
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
	r.SetRegistry(pkgKey, pkg)
	pkg.Set(loadedKey, rt.NewTable())
	pkg.Set(preloadKey, rt.NewTable())
	searchers := rt.NewTable()
	searchers.Set(rt.Int(1), searchPreloadGoFunc)
	searchers.Set(rt.Int(2), searchLuaGoFunc)
	pkg.Set(searchersKey, searchers)
	pkg.Set(pathKey, rt.String(defaultPath))
	pkg.Set(configKey, rt.String(defaultConfig.String()))
	rt.SetEnvGoFunc(env, "require", require, 1, false)
	return pkg
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
	confStr, ok := pkg.Get(configKey).(rt.String)
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
	pkg := pkgTable(t)

	// First check is the module is already loaded
	loaded, ok := pkg.Get(loadedKey).(*rt.Table)
	if !ok {
		return nil, rt.NewErrorS("package.loaded must be a table").AddContext(c)
	}
	next := c.Next()
	if mod := loaded.Get(name); !rt.IsNil(mod) {
		next.Push(mod)
		return next, nil
	}

	// If not, then go through the searchers
	searchers, ok := pkg.Get(searchersKey).(*rt.Table)
	if !ok {
		return nil, rt.NewErrorS("package.searchers must be a table").AddContext(c)
	}

	for i := rt.Int(1); ; i++ {
		searcher := searchers.Get(i)
		if rt.IsNil(searcher) {
			err = rt.NewErrorF("could not find package '%s'", name)
			break
		}
		res := rt.NewTerminationWith(2, false)
		if err = rt.Call(t, searcher, []rt.Value{name}, res); err != nil {
			break
		}
		// We got a loader, so call it
		if loader, ok := res.Get(0).(rt.Callable); ok {
			val := res.Get(1)
			res = rt.NewTerminationWith(2, false)
			if err = rt.Call(t, loader, []rt.Value{name, val}, res); err != nil {
				break
			}
			var mod rt.Value = rt.Bool(true)
			if r0 := res.Get(0); !rt.IsNil(r0) {
				mod = r0
			}
			loaded.Set(name, mod)
			next.Push(mod)
			return next, nil
		}
	}
	return nil, err.AddContext(c)
}

func searchpath(name, path, dot string, conf *config) (string, []string) {
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
	loader := pkgTable(t).Get(preloadKey).(*rt.Table).Get(s)
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
	path, ok := pkg.Get(pathKey).(rt.String)
	if !ok {
		return nil, rt.NewErrorS("package.path must be a string").AddContext(c)
	}
	conf := getConfig(pkg)
	found, templates := searchpath(string(s), string(path), ".", conf)
	next := c.Next()
	if found == "" {
		next.Push(rt.String(strings.Join(templates, "\n")))
	} else {
		next.Push(loadLuaGoFunc)
		next.Push(rt.String(found))
	}
	return next, nil
}

var loadLuaGoFunc = rt.NewGoFunction(loadLua, "loadlua", 2, false)
var searchLuaGoFunc = rt.NewGoFunction(searchLua, "searchlua", 1, false)
var searchPreloadGoFunc = rt.NewGoFunction(searchPreload, "searchpreload", 1, false)

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
	return rt.Continue(t, clos, c.Next())
}

func pkgTable(t *rt.Thread) *rt.Table {
	return t.Registry(pkgKey).(*rt.Table)
}

func SavePackage(t *rt.Thread, name rt.String, val rt.Value) error {
	pkg := pkgTable(t)

	// First check is the module is already loaded
	loaded, ok := pkg.Get(loadedKey).(*rt.Table)
	if !ok {
		return errors.New("package.loaded must be a table")
	}
	loaded.Set(name, val)
	return nil
}
