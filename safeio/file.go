package safeio

import (
	"errors"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// OpenFile is like os.OpenFile but only succeeds if a's FSAccess allows it.
// Opening with os.O_CREATE an existing file does not require CreateAction.  A
// relative name is resolved against a's working directory.
func OpenFile(a FSAccessor, name string, flag int, perm fs.FileMode) (*os.File, error) {
	f, err := openFile(a.FSAccess(), InWorkDir(a, name), flag, perm)
	return f, withPath(err, name)
}

func openFile(acc *FSAccess, name string, flag int, perm fs.FileMode) (*os.File, error) {
	if acc.IsUnrestricted() {
		return os.OpenFile(name, flag, perm)
	}
	actions := osFlagToFSActions(flag)
	if flag&os.O_CREATE != 0 && flag&os.O_EXCL == 0 {
		// If the file already exists, the user is allowed to open it even if
		// they are not allowed to create it so first try to open it without
		// creating it.
		f, err := acc.openFile(name, flag&^os.O_CREATE, perm, actions&^CreateAction)
		if err == nil || !errors.Is(err, fs.ErrNotExist) {
			return f, err
		}
	}
	return acc.openFile(name, flag, perm, actions)
}

func (a *FSAccess) openFile(name string, flag int, perm fs.FileMode, actions FSAction) (*os.File, error) {
	root, rel, err := a.resolve(name, actions)
	if err != nil {
		return nil, err
	}
	return root.OpenFile(rel, flag, perm)
}

// TempFile is like os.CreateTemp but only succeeds if a's FSAccess allows
// creating files in dir (os.TempDir() if dir is empty).  A relative dir is
// resolved against a's working directory.
func TempFile(a FSAccessor, dir string, pattern string) (*os.File, error) {
	acc := a.FSAccess()
	dir = InWorkDir(a, dir)
	if acc.IsUnrestricted() {
		return os.CreateTemp(dir, pattern)
	}
	if dir == "" {
		dir = os.TempDir()
	}
	root, rel, err := acc.resolve(dir, CreateAction|ReadAction|WriteAction)
	if err != nil {
		return nil, err
	}
	prefix, suffix, _ := strings.Cut(pattern, "*")
	for i := 0; i < 10000; i++ {
		name := filepath.Join(rel, prefix+nextRandom()+suffix)
		f, err := root.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		if !errors.Is(err, fs.ErrExist) {
			return f, err
		}
	}
	return nil, &fs.PathError{Op: "createtemp", Path: filepath.Join(dir, pattern), Err: fs.ErrExist}
}

// RemoveFile is like os.Remove but only succeeds if a's FSAccess allows it.  A
// relative name is resolved against a's working directory.
func RemoveFile(a FSAccessor, name string) error {
	return withPath(removeFile(a.FSAccess(), InWorkDir(a, name)), name)
}

func removeFile(acc *FSAccess, name string) error {
	if acc.IsUnrestricted() {
		return os.Remove(name)
	}
	root, rel, err := acc.resolve(name, DeleteAction)
	if err != nil {
		return err
	}
	return root.Remove(rel)
}

// RenameFile is like os.Rename but only succeeds if a's FSAccess allows it.
// When access is restricted, both files must be in the same granted directory,
// which must allow deleting and creating files (renaming can replace an
// existing file).  Relative names are resolved against a's working directory.
func RenameFile(a FSAccessor, oldName, newName string) error {
	err := renameFile(a.FSAccess(), InWorkDir(a, oldName), InWorkDir(a, newName))
	var le *os.LinkError
	if errors.As(err, &le) {
		le.Old, le.New = oldName, newName
	}
	return err
}

func renameFile(acc *FSAccess, oldName, newName string) error {
	if acc.IsUnrestricted() {
		return os.Rename(oldName, newName)
	}
	root, oldRel, newRel, err := acc.resolve2(oldName, newName, DeleteAction|CreateAction)
	if err != nil {
		return err
	}
	return root.Rename(oldRel, newRel)
}

// InWorkDir resolves a relative name against a's working directory.  An empty
// name is left alone so that it is still invalid.
func InWorkDir(a FSAccessor, name string) string {
	wd := a.WorkDir()
	if wd == "" || name == "" || filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(wd, name)
}

// withPath makes a *fs.PathError report the name as the caller wrote it, rather
// than as resolved (which would reveal the working directory, or for restricted
// access the name relative to a granted directory).
func withPath(err error, name string) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		pe.Path = name
	}
	return err
}

func osFlagToFSActions(flag int) FSAction {
	var actions FSAction
	switch {
	case flag&os.O_WRONLY != 0:
		actions |= WriteAction
	case flag&os.O_RDWR != 0:
		actions |= ReadAction | WriteAction
	default: // os.O_RDONLY is 0...
		actions |= ReadAction
	}
	if flag&(os.O_TRUNC|os.O_APPEND) != 0 {
		actions |= WriteAction
	}
	if flag&os.O_CREATE != 0 {
		actions |= CreateAction
	}
	return actions
}

func nextRandom() string {
	return strconv.FormatUint(uint64(rand.Uint32()), 10)
}
