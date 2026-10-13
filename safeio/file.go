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
// Opening with os.O_CREATE an existing file does not require CreateAction.
func OpenFile(a FSAccessor, name string, flag int, perm fs.FileMode) (*os.File, error) {
	acc := a.FSAccess()
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
// creating files in dir (os.TempDir() if dir is empty).
func TempFile(a FSAccessor, dir string, pattern string) (*os.File, error) {
	acc := a.FSAccess()
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

// RemoveFile is like os.Remove but only succeeds if a's FSAccess allows it.
func RemoveFile(a FSAccessor, name string) error {
	acc := a.FSAccess()
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
// existing file).
func RenameFile(a FSAccessor, oldName, newName string) error {
	acc := a.FSAccess()
	if acc.IsUnrestricted() {
		return os.Rename(oldName, newName)
	}
	root, oldRel, newRel, err := acc.resolve2(oldName, newName, DeleteAction|CreateAction)
	if err != nil {
		return err
	}
	return root.Rename(oldRel, newRel)
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
