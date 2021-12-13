package safeio

import (
	"errors"
	"io/fs"
	"io/ioutil"
	"os"

	rt "github.com/arnodel/golua/runtime"
)

func OpenFile(r *rt.Runtime, name string, flag int, perm fs.FileMode) (*os.File, error) {
	if r.RequiredFlags()&rt.ComplyIoSafe != 0 {
		return nil, ErrNotAllowed
	}
	return os.OpenFile(name, flag, perm)
}

func TempFile(r *rt.Runtime, dir string, pattern string) (*os.File, error) {
	if r.RequiredFlags()&rt.ComplyIoSafe != 0 {
		return nil, ErrNotAllowed
	}
	return ioutil.TempFile(dir, pattern)
}

func RemoveFile(r *rt.Runtime, name string) error {
	if r.RequiredFlags()&rt.ComplyIoSafe != 0 {
		return ErrNotAllowed
	}
	return os.Remove(name)
}

func RenameFile(r *rt.Runtime, oldName, newName string) error {
	if r.RequiredFlags()&rt.ComplyIoSafe != 0 {
		return ErrNotAllowed
	}
	return os.Rename(oldName, newName)
}

var ErrNotAllowed = errors.New("safeio: operation not allowed")
