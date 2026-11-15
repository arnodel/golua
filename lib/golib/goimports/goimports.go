//go:build !windows
// +build !windows

package goimports

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"plugin"
	"runtime"
	"runtime/debug"
	"strings"
	"text/template"
)

const Supported = true

// A plugin can only be opened by a program built with the same version of Go
// as the plugin.  Opening one built with another version does not fail: it
// ends the program, with "fatal error: runtime: no plugin module data", which
// nothing can recover from.  So a plugin built with another version must
// never be opened, and there are two ways to come by one:
//
//   - the plugins are kept from one run to the next, and the program may have
//     been built again since with a newer Go.  So the plugins of each version
//     of Go are kept in a directory of their own (pluginDir);
//   - a plugin is built with the go command, which may not be of the version
//     the program was built with.  So no plugin is built unless it is
//     (checkGoCommand).

// LoadGoPackage builds a plugin for a package if necessary, compiles it and loads
// it.  The value of the "Exports" symbol is returned if successful.
func LoadGoPackage(pkg string, pluginRoot string, forceBuild bool) (map[string]interface{}, error) {
	if !validPackagePath(pkg) {
		return nil, fmt.Errorf("%q is not the path of a Go package", pkg)
	}
	pluginDir := pluginDir(pluginRoot, pkg)
	filename := path.Base(pluginDir) + ".so"
	pluginPath := path.Join(pluginDir, filename)
	var p *plugin.Plugin
	var err error
	if !forceBuild {
		p, err = plugin.Open(pluginPath)
		forceBuild = err != nil
	}
	if forceBuild {
		err = buildPlugin(pkg, pluginPath)
		if err != nil {
			return nil, err
		}
		p, err = plugin.Open(pluginPath)
		if err != nil {
			return nil, err
		}
	}
	exportsI, err := p.Lookup("Exports")
	if err != nil {
		return nil, err
	}
	exports, ok := exportsI.(*map[string]interface{})
	if !ok {
		return nil, errors.New("Exports has incorrect type")
	}
	return *exports, nil
}

// validPackagePath reports whether pkg can be the import path of a Go
// package, as far as it matters here: the plugin of a package is kept in a
// directory named after it, so its path must be made of plain names, and
// lead nowhere but under the plugin root.
func validPackagePath(pkg string) bool {
	if pkg == "" || strings.ContainsAny(pkg, "\\\x00") {
		return false
	}
	for _, name := range strings.Split(pkg, "/") {
		if name == "" || name == "." || name == ".." {
			return false
		}
	}
	return true
}

// pluginDir returns the directory under root for the plugin of pkg built
// with this program's version of Go.
func pluginDir(root, pkg string) string {
	return path.Join(root, goVersionDir(runtime.Version()), pkg)
}

// goVersionDir returns a directory name for a version of Go, as
// runtime.Version gives it: "go1.27.4" as it is, and a development version,
// which has spaces and other characters, with those replaced.
func goVersionDir(version string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '+':
			return r
		}
		return '_'
	}, version)
}

// checkGoCommand returns an error unless the go command is of the version of
// Go that this program was built with.
func checkGoCommand() error {
	out, err := exec.Command("go", "env", "GOVERSION").Output()
	if err != nil {
		return fmt.Errorf("cannot run the go command: %s", err)
	}
	if v := strings.TrimSpace(string(out)); v != runtime.Version() {
		return fmt.Errorf("the go command is %s but this program was built with %s: "+
			"a Go package can only be imported when they are the same version", v, runtime.Version())
	}
	return nil
}

// buildProfile returns the profile this program was built with (profile-
// guided optimisation, the -pgo flag of go build), "off" if there was none.
// A plugin has to be built with the same one: a package compiled with a
// profile is not the package compiled without, and a plugin is refused when
// a package it shares with the program is not the same on both sides.
func buildProfile() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "-pgo" && s.Value != "" {
				return s.Value
			}
		}
	}
	return "off"
}

type libModel struct {
	Package     string
	PackageName string
	Funcs       []string
	Types       []string
	vars        []string
	consts      []string
}

func getPackagePath(pkg string) (string, string, error) {
	res, err := exec.Command("go", "list", "-f", "{{.Dir}} {{.Name}}", pkg).Output()
	if err != nil {
		return "", "", err
	}
	bits := strings.Split(strings.TrimSpace(string(res)), " ")
	return bits[0], bits[1], nil
}

func buildPlugin(pkg string, pluginPath string) error {
	if err := checkGoCommand(); err != nil {
		return err
	}
	pluginDir := path.Dir(pluginPath)
	pluginFile := path.Base(pluginPath)
	os.MkdirAll(pluginDir, 0777)
	// An earlier build may have left its module files, which "go mod init"
	// refuses to replace: without this a build that failed, or a plugin
	// that would not open, could never be put right by building again.
	os.Remove(path.Join(pluginDir, "go.mod"))
	os.Remove(path.Join(pluginDir, "go.sum"))
	f, err := os.Create(path.Join(pluginDir, "plugin.go"))
	if err != nil {
		return err
	}
	if err := buildLib(pkg, f); err != nil {
		return err
	}
	runner := cmdRunner{dir: pluginDir}

	// Since go 1.16 we need a go.mod file
	runner.
		run("go", "mod", "init", "github.com/arnode/goluaplugins/"+pkg).
		andRun("go", "mod", "tidy").
		andRun("go", "build", "-o", pluginFile, "-buildmode=plugin", "-pgo="+buildProfile())

	if runner.err != nil {
		return errors.New(runner.errMsg())
	}
	return nil
}

type cmdRunner struct {
	dir    string
	err    error
	errBuf bytes.Buffer
}

func (r *cmdRunner) run(name string, args ...string) *cmdRunner {
	cmd := exec.Command(name, args...)
	cmd.Dir = r.dir
	r.errBuf = bytes.Buffer{}
	cmd.Stderr = &r.errBuf
	r.err = cmd.Run()
	return r
}

// Run if no previous error
func (r *cmdRunner) andRun(name string, args ...string) *cmdRunner {
	if r.err != nil {
		return r
	}
	return r.run(name, args...)
}

func (r *cmdRunner) errMsg() string {
	return string(r.errBuf.Bytes())
}

func buildLib(pkg string, out io.Writer) error {
	pkgDir, pkgName, err := getPackagePath(pkg)
	if err != nil {
		return err
	}
	log.Printf("Creating lib for package %s at %s", pkgName, pkgDir)
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, pkgDir, nil, 0)
	if err != nil {
		return fmt.Errorf("Error parsing package: %s", err)
	}
	model := &libModel{
		Package:     pkg,
		PackageName: pkgName,
	}
	fillModel(model, pkgs[pkgName])
	getTemplate().Execute(out, model)
	return nil
}

func fillModel(model *libModel, pkg *ast.Package) {
	fset := token.NewFileSet()
	files := make(map[string]*ast.File)
	for fName, f := range pkg.Files {
		if !strings.HasSuffix(fName, "_test.go") {
			files[fName] = f
		}
	}
	pkg, _ = ast.NewPackage(fset, files, nil, nil)
	for _, obj := range pkg.Scope.Objects {
		switch obj.Kind {
		case ast.Fun:
			fdecl := obj.Decl.(*ast.FuncDecl)
			if !fdecl.Name.IsExported() {
				continue
			}
			name := fdecl.Name.String()
			model.Funcs = append(model.Funcs, name)
		case ast.Var:
			// fmt.Printf("Var: %s %+v\n", obj.Name, obj.Decl)
		// case ast.Con:
		case ast.Typ:
			tdecl := obj.Decl.(*ast.TypeSpec)
			if !tdecl.Name.IsExported() {
				continue
			}
			model.Types = append(model.Types, tdecl.Name.String())
		default:
		}
	}
}

func getTemplate() *template.Template {
	tpl, err := template.New("lib").Parse(templateStr)
	if err != nil {
		panic(err)
	}
	return tpl
}

const templateStr = `
package main
{{ $pkgName := .PackageName }}
import "{{ .Package }}"

var Exports = map[string]interface{}{

	// Functions
{{ range .Funcs }}
	"{{ . }}": {{ $pkgName }}.{{ . }},
{{- end }}

	// Types
{{ range .Types }}
	"{{ . }}": func(x {{ $pkgName }}.{{ . }}) {{ $pkgName }}.{{ . }} { return x },
	"new{{ . }}": func() *{{ $pkgName }}.{{ . }} { return new({{ $pkgName }}.{{ . }}) },
{{- end }}
}
`
