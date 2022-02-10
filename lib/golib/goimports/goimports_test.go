//go:build !windows
// +build !windows

package goimports

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestLoadGoPackage(t *testing.T) {
	pluginRoot, err := ioutil.TempDir("", "goplugins")
	defer os.RemoveAll(pluginRoot)
	if err != nil {
		t.Fatalf("unable to create temp dir: %s", err)
	}
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
