package iolib

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strings"

	rt "github.com/arnodel/golua/runtime"
)

type ioKeyType struct{}

var ioKey = ioKeyType{}

func Load(r *rt.Runtime) {
	meta := rt.NewTable()
	rt.SetEnvGoFunc(meta, "read", fileread, 1, true)
	rt.SetEnvGoFunc(meta, "lines", filelines, 1, true)
	rt.SetEnvGoFunc(meta, "close", closef, 1, false)
	rt.SetEnvGoFunc(meta, "flush", flush, 1, false)
	// TODO: seek
	// TODO: setvbuf
	// TODO: write

	r.SetRegistry(ioKey, &ioData{
		defaultOutput: &File{file: os.Stdout},
		defaultInput:  &File{file: os.Stdin},
		metatable:     meta,
	})
	pkg := rt.NewTable()
	rt.SetEnv(r.GlobalEnv(), "io", pkg)
	rt.SetEnvGoFunc(pkg, "close", closef, 1, false)
	rt.SetEnvGoFunc(pkg, "flush", flush, 0, false)
	rt.SetEnvGoFunc(pkg, "input", input, 1, false)
	rt.SetEnvGoFunc(pkg, "lines", iolines, 1, true)
	rt.SetEnvGoFunc(pkg, "open", open, 2, false)
	rt.SetEnvGoFunc(pkg, "output", output, 1, false)
	// TODO: popen
	rt.SetEnvGoFunc(pkg, "read", ioread, 0, true)
	// TODO: tmpfile
	rt.SetEnvGoFunc(pkg, "type", typef, 1, false)
	// TODO: write
}

type ioData struct {
	defaultOutput *File
	defaultInput  *File
	metatable     *rt.Table
}

type File struct {
	file   *os.File
	closed bool
}

func (f *File) IsClosed() bool {
	return f.closed
}

func (f *File) Close() error {
	f.closed = true
	return f.file.Close()
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
		if b[0] == '\n' {
			if withEnd {
				buf.Write(b)
			}
			break
		}
	}
	return rt.String(buf.String()), nil
}

func lineReader(withEnd bool) formatReader {
	return func(f *File) (rt.Value, error) {
		return f.ReadLine(withEnd)
	}
}

func (f *File) Read(n int) (rt.Value, error) {
	b := make([]byte, n)
	n, err := f.file.Read(b)
	if err == nil || err == io.EOF && n > 0 {
		return rt.String(b), nil
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

func FileArg(c *rt.GoCont, n int) (*File, *rt.Error) {
	u, ok := c.Arg(n).(*rt.UserData)
	if ok {
		if f, ok := u.Value().(*File); ok {
			return f, nil
		}
	}
	return nil, rt.NewErrorF("#%d must be a file")
}

func getIoData(t *rt.Thread) *ioData {
	return t.Registry(ioKey).(*ioData)
}

func ioError(err error) *rt.Error {
	if err != nil {
		return rt.NewErrorE(err)
	}
	return nil
}

func pushIoResult(next rt.Cont, ioErr error) {
	if ioErr != nil {
		next.Push(nil)
		next.Push(ioError(ioErr))
	} else {
		next.Push(rt.Bool(true))
	}
}

func pushingNextIoResult(c *rt.GoCont, ioErr error) rt.Cont {
	next := c.Next()
	pushIoResult(next, ioErr)
	return next
}

func closef(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var f *File
	if c.NArgs() == 0 {
		f = getIoData(t).defaultOutput
	} else {
		var err *rt.Error
		f, err = FileArg(c, 0)
		if err != nil {
			return nil, err.AddContext(c)
		}
	}
	return pushingNextIoResult(c, f.Close()), nil
}

func flush(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var f *File
	if c.NArgs() == 0 {
		f = getIoData(t).defaultOutput
	} else {
		var err *rt.Error
		f, err = FileArg(c, 0)
		if err != nil {
			return nil, err.AddContext(c)
		}
	}
	return pushingNextIoResult(c, f.Flush()), nil
}

var errFileOrFilename = rt.NewErrorS("#1 must be a file or a filename")

func input(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var f *File
	if c.NArgs() == 0 {
		f = getIoData(t).defaultInput
		return c.PushingNext(f), nil
	}
	switch x := c.Arg(0).(type) {
	case rt.String:
		var ioErr error
		f, ioErr = OpenFile(string(x), "r")
		if ioErr != nil {
			return nil, rt.NewErrorE(ioErr).AddContext(c)
		}
	case *rt.UserData:
		var err *rt.Error
		f, err = FileArg(c, 0)
		if err != nil {
			return nil, errFileOrFilename.AddContext(c)
		}
	default:
		return nil, errFileOrFilename.AddContext(c)
	}
	getIoData(t).defaultInput = f
	return c.Next(), nil
}

func output(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var f *File
	if c.NArgs() == 0 {
		f = getIoData(t).defaultOutput
		return c.PushingNext(f), nil
	}
	switch x := c.Arg(0).(type) {
	case rt.String:
		var ioErr error
		f, ioErr = OpenFile(string(x), "w")
		if ioErr != nil {
			return nil, rt.NewErrorE(ioErr).AddContext(c)
		}
	case *rt.UserData:
		var err *rt.Error
		f, err = FileArg(c, 0)
		if err != nil {
			return nil, errFileOrFilename.AddContext(c)
		}
	default:
		return nil, errFileOrFilename.AddContext(c)
	}
	getIoData(t).defaultOutput = f
	return c.Next(), nil
}

func iolines(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var f *File
	if c.NArgs() == 0 {
		f = getIoData(t).defaultInput
	} else {
		fname, err := c.StringArg(0)
		if err != nil {
			return nil, err.AddContext(c)
		}
		var ioErr error
		f, ioErr = OpenFile(string(fname), "r")
		if ioErr != nil {
			return nil, rt.NewErrorE(ioErr).AddContext(c)
		}
	}
	readers, fmtErr := getFormatReaders(c.Etc())
	if fmtErr != nil {
		return nil, rt.NewErrorE(fmtErr).AddContext(c)
	}
	return c.PushingNext(lines(f, readers, closeAtEOF)), nil
}

func filelines(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	f, err := FileArg(c, 0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	readers, fmtErr := getFormatReaders(c.Etc())
	if fmtErr != nil {
		return nil, rt.NewErrorE(fmtErr).AddContext(c)
	}

	return c.PushingNext(lines(f, readers, doNotCloseAtEOF)), nil
}

const (
	closeAtEOF      = 1
	doNotCloseAtEOF = 0
)

func lines(f *File, readers []formatReader, flags int) *rt.GoFunction {
	if len(readers) == 0 {
		readers = []formatReader{lineReader(false)}
	}
	iterator := func(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
		next := c.Next()
		err := read(f, readers, next)
		if err != nil {
			if err == io.EOF && flags&closeAtEOF != 0 {
				f.Close()
			}
			return nil, err.AddContext(c)
		}
		return next, nil
	}
	return rt.NewGoFunction(iterator, "linesiterator", 0, false)
}

func open(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	fname, err := c.StringArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	mode := rt.String("r")
	if c.NArgs() >= 2 {
		mode, err = c.StringArg(1)
		if err != nil {
			return nil, err.AddContext(c)
		}
	}
	f, ioErr := OpenFile(string(fname), string(mode))
	if ioErr != nil {
		return nil, rt.NewErrorE(ioErr).AddContext(c)
	}
	return c.PushingNext(rt.NewUserData(f, getIoData(t).metatable)), nil
}

func typef(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	f, err := FileArg(c, 0)
	var val rt.Value
	if err != nil {
		val = nil
	} else if f.IsClosed() {
		val = rt.String("closed file")
	} else {
		val = rt.String("file")
	}
	return c.PushingNext(val), nil
}

func iowrite(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	f := 
}

func write(f *File, values []rt.Value) *rt.Error {
	for _, val := range values {
		switch val.(type) {
		case rt.String:
		case rt.Int:
		case rt.Float:
		default:
			return rt.NewErrorS("argument must be a string or a number")
		}
		s, _ := rt.AsString(val)
		if err := f.WriteString(string(s)); err != nil {
			return rt.NewErrorE(err)
		}
	}
	return nil
}
