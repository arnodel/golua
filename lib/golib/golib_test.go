package golib

import (
	"os"
	"path"
	"testing"
)

// The plugins root is looked up when a package is imported, so a program
// can set GOLUA_PLUGINS_ROOT after it has started.
func TestPluginsRoot(t *testing.T) {
	t.Run("GOLUA_PLUGINS_ROOT names it", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("GOLUA_PLUGINS_ROOT", dir)
		root, err := pluginsRoot()
		if err != nil || root != dir {
			t.Fatalf("got %q, %v; want %q", root, err, dir)
		}
	})
	t.Run("GOLUA_PLUGINS_ROOT empty", func(t *testing.T) {
		t.Setenv("GOLUA_PLUGINS_ROOT", "")
		if root, err := pluginsRoot(); err == nil {
			t.Fatalf("got %q, want an error", root)
		}
	})
	t.Run("in the home directory", func(t *testing.T) {
		home := t.TempDir()
		unsetenv(t, "GOLUA_PLUGINS_ROOT")
		t.Setenv("HOME", home)
		root, err := pluginsRoot()
		want := path.Join(home, ".golua/goplugins")
		if err != nil || root != want {
			t.Fatalf("got %q, %v; want %q", root, err, want)
		}
	})
	t.Run("no home directory", func(t *testing.T) {
		unsetenv(t, "GOLUA_PLUGINS_ROOT")
		t.Setenv("HOME", "")
		if root, err := pluginsRoot(); err == nil {
			t.Fatalf("got %q, want an error", root)
		}
	})
}

// unsetenv unsets an environment variable for the rest of the test.
func unsetenv(t *testing.T, name string) {
	t.Setenv(name, "") // so that it is restored afterwards
	os.Unsetenv(name)
}
