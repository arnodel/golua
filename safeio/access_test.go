package safeio

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

type accessor struct{ a *FSAccess }

func (a accessor) FSAccess() *FSAccess { return a.a }

// setupDirs creates the following tree in a temporary directory and makes it
// the current directory.
//
//	sandbox/a.txt
//	sandbox/sub/b.txt
//	sandbox-evil/e.txt
//	other/s.txt
func setupDirs(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"sandbox/a.txt", "sandbox/sub/b.txt", "sandbox-evil/e.txt", "other/s.txt"} {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)
	return dir
}

func symlink(t *testing.T, oldname, newname string) {
	t.Helper()
	if err := os.Symlink(oldname, newname); err != nil {
		t.Skipf("cannot create symlinks: %s", err)
	}
}

func restrict(t *testing.T, parent *FSAccess, grants ...FSGrant) accessor {
	t.Helper()
	a, err := parent.Restrict(grants)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	return accessor{a}
}

func checkOpen(t *testing.T, a accessor, name string, flag int, wantErr error) {
	t.Helper()
	f, err := OpenFile(a, name, flag, 0o644)
	if f != nil {
		f.Close()
	}
	switch {
	case wantErr == nil && err != nil:
		t.Errorf("open %q: unexpected error %v", name, err)
	case wantErr != nil && !errors.Is(err, wantErr):
		t.Errorf("open %q: expected error %v, got %v", name, wantErr, err)
	}
}

func checkOpenFails(t *testing.T, a accessor, name string, flag int) {
	t.Helper()
	f, err := OpenFile(a, name, flag, 0o644)
	if err == nil {
		f.Close()
		t.Errorf("open %q: expected an error", name)
	}
}

func TestUnrestricted(t *testing.T) {
	setupDirs(t)
	a := accessor{Unrestricted}
	checkOpen(t, a, "other/s.txt", os.O_RDONLY, nil)
	checkOpen(t, a, "other/new.txt", os.O_WRONLY|os.O_CREATE, nil)
	if err := RenameFile(a, "other/new.txt", "sandbox/new.txt"); err != nil {
		t.Error(err)
	}
	if err := RemoveFile(a, "sandbox/new.txt"); err != nil {
		t.Error(err)
	}
}

func TestNoAccess(t *testing.T) {
	setupDirs(t)
	a := accessor{NoAccess}
	checkOpen(t, a, "sandbox/a.txt", os.O_RDONLY, ErrNotAllowed)
	if _, err := TempFile(a, "", ""); !errors.Is(err, ErrNotAllowed) {
		t.Errorf("TempFile: expected ErrNotAllowed, got %v", err)
	}
	if _, err := NoAccess.Restrict([]FSGrant{{Dir: "sandbox", Actions: AllActions}}); !errors.Is(err, ErrNotAllowed) {
		t.Errorf("Restrict: expected ErrNotAllowed, got %v", err)
	}
}

func TestReadConfinedToDir(t *testing.T) {
	dir := setupDirs(t)
	a := restrict(t, Unrestricted, FSGrant{Dir: "sandbox", Actions: ReadAction})

	checkOpen(t, a, "sandbox/a.txt", os.O_RDONLY, nil)
	checkOpen(t, a, "./sandbox/sub/b.txt", os.O_RDONLY, nil)
	checkOpen(t, a, filepath.Join(dir, "sandbox", "a.txt"), os.O_RDONLY, nil)
	checkOpen(t, a, "sandbox/sub/../a.txt", os.O_RDONLY, nil)

	checkOpen(t, a, "other/s.txt", os.O_RDONLY, ErrNotAllowed)
	checkOpen(t, a, "sandbox-evil/e.txt", os.O_RDONLY, ErrNotAllowed)
	checkOpen(t, a, "sandbox/../other/s.txt", os.O_RDONLY, ErrNotAllowed)
	checkOpen(t, a, filepath.Join(dir, "other", "s.txt"), os.O_RDONLY, ErrNotAllowed)

	// Only reading is allowed
	checkOpen(t, a, "sandbox/a.txt", os.O_WRONLY, ErrNotAllowed)
	checkOpen(t, a, "sandbox/a.txt", os.O_RDWR, ErrNotAllowed)
	checkOpen(t, a, "sandbox/new.txt", os.O_RDONLY|os.O_CREATE, ErrNotAllowed)
}

func TestSymlinkEscape(t *testing.T) {
	setupDirs(t)
	symlink(t, filepath.Join("..", "other", "s.txt"), filepath.Join("sandbox", "out.txt"))
	symlink(t, filepath.Join("..", "other"), filepath.Join("sandbox", "outdir"))
	symlink(t, "a.txt", filepath.Join("sandbox", "in.txt"))
	a := restrict(t, Unrestricted, FSGrant{Dir: "sandbox", Actions: AllActions})

	checkOpen(t, a, "sandbox/in.txt", os.O_RDONLY, nil)
	checkOpenFails(t, a, "sandbox/out.txt", os.O_RDONLY)
	checkOpenFails(t, a, "sandbox/outdir/s.txt", os.O_RDONLY)
	checkOpenFails(t, a, "sandbox/outdir/new.txt", os.O_WRONLY|os.O_CREATE)
	if _, err := os.Stat("other/new.txt"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("file created outside of sandbox")
	}
	if err := RenameFile(a, "sandbox/a.txt", "sandbox/outdir/moved.txt"); err == nil {
		t.Errorf("rename through symlink should fail")
	}
}

func TestCreate(t *testing.T) {
	setupDirs(t)
	rw := restrict(t, Unrestricted, FSGrant{Dir: "sandbox", Actions: ReadAction | WriteAction})

	// Existing files can be opened with O_CREATE without CreateAction
	checkOpen(t, rw, "sandbox/a.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, nil)
	checkOpen(t, rw, "sandbox/new.txt", os.O_WRONLY|os.O_CREATE, ErrNotAllowed)
	checkOpen(t, rw, "sandbox/a.txt", os.O_WRONLY|os.O_CREATE|os.O_EXCL, ErrNotAllowed)

	rwc := restrict(t, Unrestricted, FSGrant{Dir: "sandbox", Actions: ReadAction | WriteAction | CreateAction})
	checkOpen(t, rwc, "sandbox/new.txt", os.O_WRONLY|os.O_CREATE, nil)
	if _, err := os.Stat("sandbox/new.txt"); err != nil {
		t.Error(err)
	}
}

func TestRemoveAndRename(t *testing.T) {
	setupDirs(t)
	a := restrict(t, Unrestricted,
		FSGrant{Dir: "sandbox", Actions: AllActions},
		FSGrant{Dir: "sandbox-evil", Actions: AllActions},
		FSGrant{Dir: "other", Actions: ReadAction},
	)
	if err := RenameFile(a, "sandbox/a.txt", "sandbox/sub/a.txt"); err != nil {
		t.Errorf("rename within grant: %v", err)
	}
	if err := RenameFile(a, "sandbox/sub/a.txt", "sandbox-evil/a.txt"); !errors.Is(err, ErrNotAllowed) {
		t.Errorf("rename across grants: expected ErrNotAllowed, got %v", err)
	}
	if err := RenameFile(a, "other/s.txt", "sandbox/s.txt"); !errors.Is(err, ErrNotAllowed) {
		t.Errorf("rename from read-only grant: expected ErrNotAllowed, got %v", err)
	}
	if err := RemoveFile(a, "other/s.txt"); !errors.Is(err, ErrNotAllowed) {
		t.Errorf("remove in read-only grant: expected ErrNotAllowed, got %v", err)
	}
	if err := RemoveFile(a, "sandbox/sub/a.txt"); err != nil {
		t.Errorf("remove in grant: %v", err)
	}
}

func TestOverlappingGrants(t *testing.T) {
	setupDirs(t)
	// Read everything, write only in sandbox/sub
	a := restrict(t, Unrestricted,
		FSGrant{Dir: ".", Actions: ReadAction},
		FSGrant{Dir: "sandbox/sub", Actions: ReadAction | WriteAction},
	)
	checkOpen(t, a, "other/s.txt", os.O_RDONLY, nil)
	checkOpen(t, a, "sandbox/sub/b.txt", os.O_RDWR, nil)
	checkOpen(t, a, "sandbox/a.txt", os.O_RDWR, ErrNotAllowed)
}

func TestTempFile(t *testing.T) {
	setupDirs(t)
	a := restrict(t, Unrestricted, FSGrant{Dir: "sandbox", Actions: AllActions})
	f, err := TempFile(a, "sandbox", "tmp*.txt")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	abs, _ := filepath.Abs("sandbox")
	if filepath.Dir(f.Name()) != abs {
		t.Errorf("temp file %q not in sandbox", f.Name())
	}
	if _, err := TempFile(a, "other", ""); !errors.Is(err, ErrNotAllowed) {
		t.Errorf("TempFile outside grant: expected ErrNotAllowed, got %v", err)
	}
}

func TestRestrictNested(t *testing.T) {
	setupDirs(t)
	parent := restrict(t, Unrestricted,
		FSGrant{Dir: "sandbox", Actions: ReadAction | WriteAction},
	)

	// A child cannot widen the directories or the actions
	if _, err := parent.a.Restrict([]FSGrant{{Dir: "other", Actions: ReadAction}}); !errors.Is(err, ErrNotAllowed) {
		t.Errorf("expected ErrNotAllowed, got %v", err)
	}
	child := restrict(t, parent.a, FSGrant{Dir: "sandbox/sub", Actions: AllActions})
	checkOpen(t, child, "sandbox/sub/b.txt", os.O_RDWR, nil)
	checkOpen(t, child, "sandbox/sub/new.txt", os.O_WRONLY|os.O_CREATE, ErrNotAllowed)
	checkOpen(t, child, "sandbox/a.txt", os.O_RDONLY, ErrNotAllowed)

	// The child can still work after the parent is closed
	parent.a.Close()
	checkOpen(t, child, "sandbox/sub/b.txt", os.O_RDONLY, nil)
}

func TestRestrictNestedSymlink(t *testing.T) {
	setupDirs(t)
	symlink(t, filepath.Join("..", "other"), filepath.Join("sandbox", "outdir"))
	parent := restrict(t, Unrestricted, FSGrant{Dir: "sandbox", Actions: AllActions})
	if _, err := parent.a.Restrict([]FSGrant{{Dir: "sandbox/outdir", Actions: ReadAction}}); err == nil {
		t.Errorf("granting a symlink escaping the parent's directory should fail")
	}
}

func TestParseFSActions(t *testing.T) {
	tests := []struct {
		s    string
		want FSAction
		err  bool
	}{
		{"", NoActions, false},
		{"r", ReadAction, false},
		{"rw", ReadAction | WriteAction, false},
		{"cd", CreateAction | DeleteAction, false},
		{"*", AllActions, false},
		{"rx", 0, true},
	}
	for _, tt := range tests {
		got, err := ParseFSActions(tt.s)
		if (err != nil) != tt.err || got != tt.want {
			t.Errorf("ParseFSActions(%q) = %v, %v", tt.s, got, err)
		}
	}
}
