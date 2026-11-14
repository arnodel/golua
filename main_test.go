//go:build !windows
// +build !windows

package main

import (
	"debug/buildinfo"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The golua command, built as "go build" and "go install" build it, must be
// able to import a Go package.  It is built with a profile (default.pgo),
// and a plugin is only accepted by a program built the same way as it was.
// The tests of the packages cannot see that: a test binary is built without
// the profile.  So this one builds the command and has it import a package.
func TestCommandImportsGoPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the golua command and a plugin")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "golua")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("building the command: %s\n%s", err, out)
	}
	if info, err := buildinfo.ReadFile(bin); err == nil {
		for _, s := range info.Settings {
			if s.Key == "-pgo" {
				t.Logf("the command was built with the profile %s", s.Value)
			}
		}
	}
	cmd := exec.Command(bin, "-e", `local go = require("golib"); print(go.import("fmt").Sprintf("%s-%d", "imported", 42))`)
	cmd.Env = append(os.Environ(), "GOLUA_PLUGINS_ROOT="+filepath.Join(dir, "plugins"))
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "imported-42") {
		t.Fatalf("importing fmt: %v\n%s", err, out)
	}
}
