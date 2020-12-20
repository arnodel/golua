package iolib

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	rt "github.com/arnodel/golua/runtime"
)

type bufReader interface {
	io.Reader
	Reset(r io.Reader)
}

type bufWriter interface {
	io.Writer
	Reset(w io.Writer)
	Flush() error
}
type nobufReader struct {
	io.Reader
}

var (
	_ bufReader = (*nobufReader)(nil)
	_ bufReader = (*bufio.Reader)(nil)
	_ bufWriter = (*nobufWriter)(nil)
	_ bufWriter = (*bufio.Writer)(nil)
)

func (u *nobufReader) Reset(r io.Reader) {
	u.Reader = r
}

type nobufWriter struct {
	io.Writer
}

func (u *nobufWriter) Reset(w io.Writer) {
	u.Writer = w
}

func (u *nobufWriter) Flush() error {
	return nil
}

// A File wraps an os.File for manipulation by iolib.
type File struct {
	file   *os.File
	closed bool
	reader bufReader
	writer bufWriter
}

// NewFile returns a new *File from an *os.File.
func NewFile(file *os.File, buffered bool) *File {
	f := &File{file: file}
	if buffered {
		f.reader = &nobufReader{file}
		f.writer = &nobufWriter{file}
	} else {
		f.reader = bufio.NewReader(file)
		f.writer = bufio.NewWriter(file)
	}
	return f
}

// FileArg turns a continuation argument into a *File.
func FileArg(c *rt.GoCont, n int) (*File, *rt.Error) {
	f, ok := ValueToFile(c.Arg(n))
	if ok {
		return f, nil
	}
	return nil, rt.NewErrorF("#%d must be a file", n+1)
}

// ValueToFile turns a lua value to a *File if possible.
func ValueToFile(v rt.Value) (*File, bool) {
	u, ok := v.TryUserData()
	if ok {
		return u.Value().(*File), true
	}
	return nil, false
}

// TempFile tries to make a temporary file, and if successful schedules the file
// to be removed when the process dies.
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

// IsClosed returns true if the file is close.
func (f *File) IsClosed() bool {
	return f.closed
}

// Close attempts to close the file, returns an error if not successful.
func (f *File) Close() error {
	f.closed = true
	err := f.file.Close()
	return err
}

// Flush attempts to sync the file, returns an error if a problem occurs.
func (f *File) Flush() error {
	if err := f.writer.Flush(); err != nil {
		return err
	}
	return f.file.Sync()
}

// OpenFile opens a file with the given name in the given lua mode.
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
	return NewFile(f, true), nil
}

// ReadLine reads a line from the file.  If withEnd is true, it will include the
// end of the line in the returned value.
func (f *File) ReadLine(withEnd bool) (rt.Value, error) {
	file := f.file
	var buf bytes.Buffer
	b := []byte{0}
	for {
		_, err := file.Read(b)
		if err != nil {
			if err != io.EOF || buf.Len() == 0 {
				return rt.NilValue, err
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
	return rt.StringValue(buf.String()), nil
}

// Read return a lua string made of up to n bytes.
func (f *File) Read(n int) (rt.Value, error) {
	b := make([]byte, n)
	n, err := f.reader.Read(b)
	if err == nil || err == io.EOF && n > 0 {
		return rt.StringValue(string(b[:n])), nil
	}
	return rt.NilValue, err
}

// ReadAll attempts to read the whole file and return a lua string containing
// it.
func (f *File) ReadAll() (rt.Value, error) {
	b, err := ioutil.ReadAll(f.reader)
	if err != nil {
		return rt.NilValue, err
	}
	return rt.StringValue(string(b)), nil
}

// ReadNumber tries to read a number from the file.
func (f *File) ReadNumber() (rt.Value, error) {
	return rt.NilValue, errors.New("readNumber unimplemented")
}

// WriteString writes a string to the file.
func (f *File) WriteString(s string) error {
	_, err := f.writer.Write([]byte(s))
	return err
}

// Seek seeks from the file.
func (f *File) Seek(offset int64, whence int) (n int64, err error) {
	n, err = f.file.Seek(offset, whence)
	f.reader.Reset(f.file)
	f.writer.Reset(f.file)
	return
}

// Name returns the file name.
func (f *File) Name() string {
	return f.file.Name()
}
