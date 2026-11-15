//go:build !windows
// +build !windows

package goimports

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadGoPackage(t *testing.T) {
	pluginRoot := t.TempDir()
	pkg, err := LoadGoPackage("fmt", pluginRoot, true)
	if err != nil {
		t.Fatalf("error loading fmt package: %s", err)
	}
	_, ok := pkg["Sprintf"]
	if !ok {
		t.Fatalf("expected Sprintf to be exported")
	}
	_, err = LoadGoPackage("fmt", pluginRoot, false)
	if err != nil {
		t.Errorf("expected second import to be successful")
	}
}

// A plugin is kept in a directory for the version of Go it was built with,
// so that a program built with another version never opens it: that would
// not fail, it would end the program.
func TestPluginsAreKeptByGoVersion(t *testing.T) {
	// Not fmt: a program cannot load the plugin of one package from two
	// places, and TestLoadGoPackage has loaded fmt's.
	root := t.TempDir()
	// What an earlier golua, or one built with another version of Go, may
	// have left where plugins used to be kept, and where another version
	// keeps them.  Neither is a plugin, so opening one would fail rather
	// than end the test, but neither must be touched.
	stale := []string{
		filepath.Join(root, "path", "path.so"),
		filepath.Join(root, "go0.0.1", "path", "path.so"),
	}
	for _, f := range stale {
		if err := os.MkdirAll(filepath.Dir(f), 0777); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, []byte("stale"), 0666); err != nil {
			t.Fatal(err)
		}
	}
	// And what a build that failed leaves in this version's directory, which
	// must not stop the plugin being built.
	left := filepath.Join(root, runtime.Version(), "path", "go.mod")
	if err := os.MkdirAll(filepath.Dir(left), 0777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(left, []byte("module left/behind\n"), 0666); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadGoPackage("path", root, false); err != nil {
		t.Fatalf("error loading path package: %s", err)
	}
	if _, err := os.Stat(filepath.Join(root, runtime.Version(), "path", "path.so")); err != nil {
		t.Errorf("no plugin in the directory of %s: %s", runtime.Version(), err)
	}
	for _, f := range stale {
		if b, err := os.ReadFile(f); err != nil || string(b) != "stale" {
			t.Errorf("%s was touched", f)
		}
	}
}

func TestGoVersionDir(t *testing.T) {
	for version, want := range map[string]string{
		"go1.27.4": "go1.27.4",
		"devel go1.28-0e1b14bc2e Tue Oct 6 10:27:12 2026 +0000": "devel_go1.28-0e1b14bc2e_Tue_Oct_6_10_27_12_2026_+0000",
		"go1.27.4 X:nocoverageredesign":                         "go1.27.4_X_nocoverageredesign",
		"../../etc":                                             ".._.._etc",
	} {
		if got := goVersionDir(version); got != want {
			t.Errorf("goVersionDir(%q) = %q, want %q", version, got, want)
		}
	}
}

// A package's plugin is kept in a directory named after it: a path that
// would lead out of the plugin root is not a package's.
func TestInvalidPackagePath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	for _, pkg := range []string{"", ".", "..", "../x", "a/../../x", "/abs/path", "a//b", "a/./b", "a\\b"} {
		if _, err := LoadGoPackage(pkg, root, false); err == nil {
			t.Errorf("%q: expected an error", pkg)
		}
	}
	if entries, _ := os.ReadDir(filepath.Dir(root)); len(entries) != 0 {
		t.Errorf("something was written: %v", entries)
	}
	for _, pkg := range []string{"fmt", "net/http", "github.com/arnodel/golua/runtime", "gopkg.in/yaml.v3"} {
		if !validPackagePath(pkg) {
			t.Errorf("%q: expected a valid path", pkg)
		}
	}
}

// A plugin built by a go command of another version than the program's would
// end the program when opened, so it is not built.
func TestGoCommandOfAnotherVersion(t *testing.T) {
	bin := t.TempDir()
	fake := "#!/bin/sh\nif [ \"$1\" = env ] && [ \"$2\" = GOVERSION ]; then echo go0.0.1; exit 0; fi\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "go"), []byte(fake), 0777); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	root := t.TempDir()
	_, err := LoadGoPackage("fmt", root, false)
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, s := range []string{"go0.0.1", runtime.Version()} {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("error %q does not mention %s", err, s)
		}
	}
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Errorf("something was written under the plugin root: %v", entries)
	}
}
