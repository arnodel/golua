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

func LoadGoPackage(pkg string, pluginRoot string) (map[string]interface{}, error) {
	pluginDir := path.Join(pluginRoot, pkg)
	filename := path.Base(pluginDir) + ".so"
	pluginPath := path.Join(pluginDir, filename)
	p, err := plugin.Open(pluginPath)
	if err != nil {
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

func getPluginPath(pkg string) {
	return
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
	cmd := exec.Command("go", "build", "-o", pluginFile, "-buildmode=plugin")
	cmd.Dir = pluginDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return errors.New(string(stderr.Bytes()))
	}
	return nil
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
{{- end }}
}
`
