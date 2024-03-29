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
	"strings"
	"text/template"
)

const Supported = true

// LoadGoPackage builds a plugin for a package if necessary, compiles it and loads
// it.  The value of the "Exports" symbol is returned if successful.
func LoadGoPackage(pkg string, pluginRoot string, forceBuild bool) (map[string]interface{}, error) {
	pluginDir := path.Join(pluginRoot, pkg)
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
	pluginDir := path.Dir(pluginPath)
	pluginFile := path.Base(pluginPath)
	os.MkdirAll(pluginDir, 0777)
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
		andRun("go", "build", "-o", pluginFile, "-buildmode=plugin")

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
