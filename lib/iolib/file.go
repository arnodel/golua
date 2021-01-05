package iolib

import (
	"bufio"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strings"

	rt "github.com/arnodel/golua/runtime"
)

// NewFile returns a new *File from an *os.File.
func NewFile(file *os.File, buffered bool) *File {
	f := &File{file: file}
	if buffered {
		f.reader = bufio.NewReader(file)
		f.writer = bufio.NewWriterSize(file, 65536)
	} else {
		f.reader = &nobufReader{file}
		f.writer = &nobufWriter{file}
	}
	currentFiles[f] = struct{}{}
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
	ff := NewFile(f, true)
	ff.temp = true
	return ff, nil
}

type bufReader interface {
	io.Reader
	Reset(r io.Reader)
	Buffered() int
	Discard(int) (int, error)
	ReadString(delim byte) (string, error)
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

func (u *nobufReader) Buffered() int {
	return 0
}

func (u *nobufReader) Discard(n int) (int, error) {
	if n > 0 {
		return 0, errors.New("nobufReader cannot discard")
	}
	return 0, nil
}

func (u *nobufReader) ReadString(delim byte) (string, error) {
	panic("unimplemented")
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
	temp   bool
}

var currentFiles = map[*File]struct{}{}

func cleanupCurrentFiles() {
	// We don't want to close the std files, that breaks testing.  In normal
	// operation it's the end of the program so that' OK too.
	for f := range currentFiles {
		switch f.file {
		case os.Stdout, os.Stderr:
			f.Flush()
		case os.Stdin:
			// Nothing to do?
		default:
			f.cleanup()
		}
	}
	currentFiles = map[*File]struct{}{}
}

func (f *File) release() {
	delete(currentFiles, f)
	f.cleanup()
}

func (f *File) cleanup() {
	_ = f.Close()
	if f.temp {
		_ = os.Remove(f.Name())
	}
}

// IsClosed returns true if the file is close.
func (f *File) IsClosed() bool {
	return f.closed
}

// Close attempts to close the file, returns an error if not successful.
func (f *File) Close() error {
	f.closed = true
	errFlush := f.writer.Flush()
	err := f.file.Close()
	if err == nil {
		return errFlush
	}
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
	s, err := f.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return rt.NilValue, err
	}
	l := len(s)
	if l == 0 {
		return rt.NilValue, err
	}
	if !withEnd && l > 0 && s[l-1] == '\n' {
		s = s[:l-1]
	}
	return rt.StringValue(s), err
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
	err = f.writer.Flush()
	if err != nil {
		return
	}
	switch whence {
	case io.SeekStart, io.SeekEnd:
		n, err = f.file.Seek(offset, whence)
		f.reader.Reset(f.file)
		f.writer.Reset(f.file)
	case io.SeekCurrent:
		var n0 int64
		n0, err = f.file.Seek(0, whence)
		bufCount := int64(f.reader.Buffered())
		n = n0 - bufCount + offset
		if err != nil {
			return
		}
		if offset < 0 || bufCount < offset {
			return f.Seek(n, io.SeekStart)
		}
		f.reader.Discard(int(offset))
	}
	return
}

// Name returns the file name.
func (f *File) Name() string {
	return f.file.Name()
}
