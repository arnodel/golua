package iolib

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	rt "github.com/arnodel/golua/runtime"
)

type File struct {
	file   *os.File
	closed bool
}

func FileArg(c *rt.GoCont, n int) (*File, *rt.Error) {
	f, ok := ValueToFile(c.Arg(n))
	if ok {
		return f, nil
	}
	return nil, rt.NewErrorF("#%d must be a file", n+1)
}

func ValueToFile(v rt.Value) (*File, bool) {
	u, ok := v.(*rt.UserData)
	if ok {
		return u.Value().(*File), true
	}
	return nil, false
}

func TempFile() (*File, error) {
	f, err := ioutil.TempFile("", "golua")
	if err != nil {
		return nil, err
	}
	ff := &File{file: f}

	// This is meant to remove the file when it becomes unreachable.
	// In fact that is done when it is garbage collected.
	runtime.SetFinalizer(ff, func(ff *File) {
		_ = ff.file.Close()
		_ = os.Remove(ff.Name())
	})
	return ff, nil
}

func (f *File) IsClosed() bool {
	return f.closed
}

func (f *File) Close() error {
	f.closed = true
	err := f.file.Close()
	return err
}

func (f *File) Flush() error {
	return f.file.Sync()
}

func OpenFile(name, mode string) (*File, error) {
	var flag int
	switch strings.TrimSuffix(mode, "b") {
	case "r":
		flag = os.O_RDONLY
	case "w":
		flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	case "a":
		flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	case "r+":
		flag = os.O_RDWR
	case "w+":
		flag = os.O_RDWR | os.O_CREATE | os.O_TRUNC
	case "a+":
		flag = os.O_RDWR | os.O_CREATE | os.O_APPEND
	default:
		return nil, errors.New("invalid mode")
	}
	f, err := os.OpenFile(name, flag, 0666)
	if err != nil {
		return nil, err
	}
	return &File{file: f}, nil
}

func (f *File) ReadLine(withEnd bool) (rt.Value, error) {
	file := f.file
	var buf bytes.Buffer
	b := []byte{0}
	for {
		_, err := file.Read(b)
		if err != nil {
			if err != io.EOF || buf.Len() == 0 {
				return nil, err
			}
			break
		}
		end := b[0] == '\n'
		if withEnd || !end {
			buf.Write(b)
		}
		if end {
			break
		}
	}
	return rt.String(buf.String()), nil
}

func (f *File) Read(n int) (rt.Value, error) {
	b := make([]byte, n)
	n, err := f.file.Read(b)
	if err == nil || err == io.EOF && n > 0 {
		return rt.String(b[:n]), nil
	}
	return nil, err
}

func (f *File) ReadAll() (rt.Value, error) {
	b, err := ioutil.ReadAll(f.file)
	if err != nil {
		return nil, err
	}
	return rt.String(b), nil
}

func (f *File) ReadNumber() (rt.Value, error) {
	return nil, errors.New("readNumber unimplemented")
}

func (f *File) WriteString(s string) error {
	_, err := f.file.Write([]byte(s))
	return err
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	return f.file.Seek(offset, whence)

}

func (f *File) Name() string {
	return f.file.Name()
}
