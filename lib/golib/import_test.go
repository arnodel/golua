//go:build !windows
// +build !windows

package golib_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arnodel/golua/lib"
	rt "github.com/arnodel/golua/runtime"
)

// go.import(pkg, true) builds the plugin of pkg again even though there is
// one already.
func TestImportForcesBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a plugin twice")
	}
	root := t.TempDir()
	t.Setenv("GOLUA_PLUGINS_ROOT", root)
	r := rt.New(os.Stdout)
	defer lib.LoadAll(r)()
	run := func(src string) {
		t.Helper()
		chunk, err := r.CompileAndLoadLuaChunk("test", []byte(src), rt.TableValue(r.GlobalEnv()))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := rt.Call1(r.MainThread(), rt.FunctionValue(chunk)); err != nil {
			t.Fatal(err)
		}
	}
	built := func() os.FileInfo {
		t.Helper()
		files, _ := filepath.Glob(filepath.Join(root, "*", "path", "*.so"))
		if len(files) != 1 {
			t.Fatalf("plugins built: %v, want one", files)
		}
		info, err := os.Stat(files[0])
		if err != nil {
			t.Fatal(err)
		}
		return info
	}

	run(`require("golib").import("path")`)
	first := built()
	run(`require("golib").import("path")`)
	if !built().ModTime().Equal(first.ModTime()) {
		t.Fatal("the plugin was built again without being asked to")
	}
	run(`require("golib").import("path", true)`)
	if built().ModTime().Equal(first.ModTime()) {
		t.Fatal("the plugin was not built again")
	}
}
