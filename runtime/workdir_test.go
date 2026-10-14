package runtime

import (
	"path/filepath"
	"testing"
)

func TestResolveWorkDir(t *testing.T) {
	cwd, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(cwd, "parent")
	tests := []struct{ parent, dir, want string }{
		{"", "", ""},                                  // root: process working directory
		{parent, "", parent},                          // inherited
		{"", "sub", filepath.Join(cwd, "sub")},        // relative to the process
		{parent, "sub", filepath.Join(parent, "sub")}, // relative to the parent
		{parent, "..", cwd},                           // cleaned
		{parent, cwd, cwd},                            // absolute
		{parent, filepath.Join(cwd, "a", "..", "b"), filepath.Join(cwd, "b")},
	}
	for _, tt := range tests {
		if got := resolveWorkDir(tt.parent, tt.dir); got != tt.want {
			t.Errorf("resolveWorkDir(%q, %q) = %q, want %q", tt.parent, tt.dir, got, tt.want)
		}
	}
}
