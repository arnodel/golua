package iolib

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"

	rt "github.com/arnodel/golua/runtime"
	"github.com/arnodel/golua/safeio"
	"github.com/arnodel/golua/scanner"
	"github.com/arnodel/golua/token"
)

const (
	bufferedRead int = 1 << iota
	bufferedWrite
	notClosable
	tempFile
)

var (
	errCloseStandardFile = errors.New("cannot close standard file")
	errFileAlreadyClosed = errors.New("file already closed")
	errInvalidBufferMode = errors.New("invalid buffer mode")
	errInvalidBufferSize = errors.New("invalid buffer size")
)

// A File wraps an os.File for manipulation by iolib.
type File struct {
	file   *os.File
	status fileStatus
	reader bufReader
	writer bufWriter
	cmd *exec.Cmd
}

type fileStatus int

const (
	statusClosed = 1 << iota
	statusTemp
	statusNotClosable
)

// NewFile returns a new *File from an *os.File.
func NewFile(file *os.File, options int) *File {
	f := &File{file: file}
	// TODO: find out if there is mileage in having unbuffered readers.
	if true || options&bufferedRead != 0 {
		f.reader = bufio.NewReader(file)
	} else {
		f.reader = &nobufReader{file}
	}
	if options&bufferedWrite != 0 {
		f.writer = bufio.NewWriterSize(file, 65536)
	} else {
		f.writer = &nobufWriter{file}
	}
	if options&tempFile != 0 {
		f.status |= statusTemp
	}
	if options&notClosable != 0 {
		f.status |= statusNotClosable
	}
	runtime.SetFinalizer(f, (*File).cleanup)
	return f
}

// OpenFile opens a file with the given name in the given lua mode.
func OpenFile(r *rt.Runtime, name, mode string) (*File, error) {
	var flag, options int
	switch strings.TrimSuffix(mode, "b") {
	case "r":
		flag = os.O_RDONLY
		options = bufferedRead
	case "w":
		flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		options = bufferedWrite
	case "a":
		flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
		options = bufferedWrite
	case "r+":
		flag = os.O_RDWR
		options = bufferedRead | bufferedWrite
	case "w+":
		flag = os.O_RDWR | os.O_CREATE | os.O_TRUNC
		options = bufferedRead | bufferedWrite
	case "a+":
		flag = os.O_RDWR | os.O_CREATE | os.O_APPEND
		options = bufferedRead | bufferedWrite
	default:
		return nil, errors.New("invalid mode")
	}
	f, err := safeio.OpenFile(r, name, flag, 0666)
	if err != nil {
		return nil, err
	}
	return NewFile(f, options), nil
}

// TempFile tries to make a temporary file, and if successful schedules the file
// to be removed when the process dies.
func TempFile(r *rt.Runtime) (*File, error) {
	f, err := safeio.TempFile(r, "", "golua")
	if err != nil {
		return nil, err
	}
	ff := NewFile(f, bufferedRead|bufferedWrite|tempFile)
	return ff, nil
}

// FileArg turns a continuation argument into a *File.
func FileArg(c *rt.GoCont, n int) (*File, error) {
	f, ok := ValueToFile(c.Arg(n))
	if ok {
		return f, nil
	}
	return nil, fmt.Errorf("#%d must be a file", n+1)
}

// ValueToFile turns a lua value to a *File if possible.
func ValueToFile(v rt.Value) (*File, bool) {
	u, ok := v.TryUserData()
	if ok {
		return u.Value().(*File), true
	}
	return nil, false
}

// IsClosed returns true if the file is closed.
func (f *File) IsClosed() bool {
	return f.status&statusClosed != 0
}

// IsTemp returns true if the file is temporary.
func (f *File) IsTemp() bool {
	return f.status&statusTemp != 0
}

func (f *File) IsClosable() bool {
	return f.status&statusNotClosable == 0
}

// Close attempts to close the file, returns an error if not successful.
func (f *File) Close() error {
	if !f.IsClosable() {
		// Lua doesn't return a Lua error, so wrap this in a PathError
		return &fs.PathError{
			Op:   "close",
			Path: f.file.Name(),
			Err:  errCloseStandardFile,
		}
	}
	if f.IsClosed() {
		// Also this is undocumented, in this case an error is returned
		return errFileAlreadyClosed
	}
	f.status |= statusClosed
	errFlush := f.writer.Flush()

	var err error
	if f.file != nil {
		err = f.file.Close()
	} else {
		if readcloser, ok := f.reader.(io.Closer); ok {
			err = readcloser.Close()
			if err != nil {
				return err
			}
		}
		if writecloser, ok := f.writer.(io.Closer); ok {
			err = writecloser.Close()
		}
	}

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
		l--
		if l > 1 && s[l-1] == '\r' {
			l--
		}
		s = s[:l]
	}
	return rt.StringValue(s), nil
}

// Read return a lua string made of up to n bytes.
func (f *File) Read(n int) (rt.Value, error) {
	if n == 0 {
		// Special case when n = 0: we try to peek 1 byte ahead to decide
		// whether it's the end of the file or not.
		_, err := f.reader.Peek(1)
		switch err {
		case nil:
			return rt.StringValue(""), nil
		case io.EOF:
			return rt.NilValue, nil
		default:
			return rt.NilValue, err
		}
	}
	b := make([]byte, n)
	n, err := io.ReadFull(f.reader, b)
	if err == nil || err == io.ErrUnexpectedEOF {
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
	const maxSize = 64
	bytes, err := f.reader.Peek(maxSize) // Should be enough for any number
	if err != nil && (err != io.EOF || len(bytes) == 0) {
		return rt.NilValue, err
	}
	scan := scanner.New("", bytes, scanner.ForNumber())
	tok := scan.Scan()
	_, _ = f.reader.Discard(len(tok.Lit))
	if tok.Type == token.INVALID || len(tok.Lit) == maxSize {
		return rt.NilValue, nil
	}
	n, x, tp := rt.StringToNumber(string(tok.Lit))
	switch tp {
	case rt.IsInt:
		return rt.IntValue(n), nil
	case rt.IsFloat:
		return rt.FloatValue(x), nil
	default:
		return rt.NilValue, nil
	}
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

func (f *File) SetWriteBuffer(mode string, size int) error {
	if size < 0 {
		return errInvalidBufferSize
	}
	f.Flush()
	switch mode {
	case "no":
		f.writer = &nobufWriter{f.file}
	case "full":
		if size == 0 {
			size = 65536
		}
		f.writer = bufio.NewWriterSize(f.file, size)
	case "line":
		if size == 0 {
			size = 65536
		}
		f.writer = linebufWriter{bufio.NewWriterSize(f.file, size)}
		// TODO
	default:
		return errInvalidBufferMode
	}
	return nil
}

// Name returns the file name.
func (f *File) Name() string {
	return f.file.Name()
}

// Best effort to flush and close files when they are no longer accessible.
func (f *File) cleanup() {
	if !f.IsClosed() {
		f.Close()
	}
	if f.IsTemp() {
		_ = os.Remove(f.Name())
	}
}
