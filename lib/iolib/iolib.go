package iolib

import (
	"fmt"
	"io"
	"os"

	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
)

// Wether std files should be buffered
var BufferedStdFiles bool = true

type ioKeyType struct{}

var ioKey = rt.AsValue(ioKeyType{})

// LibLoader can load the io lib.
var LibLoader = packagelib.Loader{
	Load: load,
	Name: "io",
}

func load(r *rt.Runtime) rt.Value {
	methods := rt.NewTable()
	rt.SetEnvGoFunc(methods, "read", fileread, 1, true)
	rt.SetEnvGoFunc(methods, "lines", filelines, 1, true)
	rt.SetEnvGoFunc(methods, "close", fileclose, 1, false)
	rt.SetEnvGoFunc(methods, "flush", fileflush, 1, false)
	rt.SetEnvGoFunc(methods, "seek", fileseek, 3, false)
	// TODO: setvbuf
	rt.SetEnvGoFunc(methods, "write", filewrite, 1, true)

	meta := rt.NewTable()
	rt.SetEnv(meta, "__index", rt.TableValue(methods))
	rt.SetEnvGoFunc(meta, "__tostring", tostring, 1, false)

	stdoutFile := NewFile(os.Stdout, BufferedStdFiles)
	if r.Stdout == nil {
		r.Stdout = stdoutFile.writer
	}
	stdin := rt.NewUserData(NewFile(os.Stdin, BufferedStdFiles), meta)
	stdout := rt.NewUserData(stdoutFile, meta)
	stderr := rt.NewUserData(NewFile(os.Stderr, false), meta) // I''m guessing, don't buffer stderr?

	r.SetRegistry(ioKey, rt.AsValue(&ioData{
		defaultOutput: stdout,
		defaultInput:  stdin,
		metatable:     meta,
	}))
	pkg := rt.NewTable()
	rt.SetEnv(pkg, "stdin", rt.UserDataValue(stdin))
	rt.SetEnv(pkg, "stdout", rt.UserDataValue(stdout))
	rt.SetEnv(pkg, "stderr", rt.UserDataValue(stderr))
	rt.SetEnvGoFunc(pkg, "close", ioclose, 1, false)
	rt.SetEnvGoFunc(pkg, "flush", ioflush, 0, false)
	rt.SetEnvGoFunc(pkg, "input", input, 1, false)
	rt.SetEnvGoFunc(pkg, "lines", iolines, 1, true)
	rt.SetEnvGoFunc(pkg, "open", open, 2, false)
	rt.SetEnvGoFunc(pkg, "output", output, 1, false)
	// TODO: popen
	rt.SetEnvGoFunc(pkg, "read", ioread, 0, true)
	rt.SetEnvGoFunc(pkg, "tmpfile", tmpfile, 0, false)
	rt.SetEnvGoFunc(pkg, "type", typef, 1, false)
	rt.SetEnvGoFunc(pkg, "write", iowrite, 0, true)

	return rt.TableValue(pkg)
}

type ioData struct {
	defaultOutput *rt.UserData
	defaultInput  *rt.UserData
	metatable     *rt.Table
}

func getIoData(t *rt.Thread) *ioData {
	return t.Registry(ioKey).Interface().(*ioData)
}

func (d *ioData) defaultOutputFile() *File {
	return d.defaultOutput.Value().(*File)
}

func (d *ioData) defaultInputFile() *File {
	return d.defaultInput.Value().(*File)
}

func ioError(err error) *rt.Error {
	if err != nil {
		return rt.NewErrorE(err)
	}
	return nil
}

func pushIoResult(next rt.Cont, ioErr error) {
	if ioErr != nil {
		next.Push(rt.NilValue)
		// TODO: Why push a *rt.Error?
		next.Push(rt.AsValue(ioError(ioErr)))
	} else {
		next.Push(rt.BoolValue(true))
	}
}

func pushingNextIoResult(c *rt.GoCont, ioErr error) rt.Cont {
	next := c.Next()
	pushIoResult(next, ioErr)
	return next
}

func ioclose(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var f *File
	if c.NArgs() == 0 {
		f = getIoData(t).defaultOutputFile()
	} else {
		var err *rt.Error
		f, err = FileArg(c, 0)
		if err != nil {
			return nil, err.AddContext(c)
		}
	}
	return pushingNextIoResult(c, f.Close()), nil
}

func fileclose(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	return ioclose(t, c)
}

func ioflush(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var f *File
	if c.NArgs() == 0 {
		f = getIoData(t).defaultOutputFile()
	} else {
		var err *rt.Error
		f, err = FileArg(c, 0)
		if err != nil {
			return nil, err.AddContext(c)
		}
	}
	return pushingNextIoResult(c, f.Flush()), nil
}

func fileflush(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	return ioflush(t, c)
}

func errFileOrFilename() *rt.Error {
	return rt.NewErrorS("#1 must be a file or a filename")
}

func input(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	ioData := getIoData(t)
	if c.NArgs() == 0 {
		return c.PushingNext(rt.UserDataValue(ioData.defaultInput)), nil
	}
	var (
		fv  *rt.UserData
		arg = c.Arg(0)
	)
	switch arg.Type() {
	case rt.StringType:
		f, ioErr := OpenFile(arg.AsString(), "r")
		if ioErr != nil {
			return nil, rt.NewErrorE(ioErr).AddContext(c)
		}
		fv = rt.NewUserData(f, ioData.metatable)
	case rt.UserDataType:
		_, err := FileArg(c, 0)
		if err != nil {
			return nil, errFileOrFilename().AddContext(c)
		}
		fv = arg.AsUserData()
	default:
		return nil, errFileOrFilename().AddContext(c)
	}
	ioData.defaultInput = fv
	return c.Next(), nil
}

func output(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	ioData := getIoData(t)
	if c.NArgs() == 0 {
		return c.PushingNext(rt.UserDataValue(ioData.defaultOutput)), nil
	}
	var (
		fv  *rt.UserData
		arg = c.Arg(0)
	)
	switch arg.Type() {
	case rt.StringType:
		f, ioErr := OpenFile(arg.AsString(), "w")
		if ioErr != nil {
			return nil, rt.NewErrorE(ioErr).AddContext(c)
		}
		fv = rt.NewUserData(f, ioData.metatable)
	case rt.UserDataType:
		_, err := FileArg(c, 0)
		if err != nil {
			return nil, errFileOrFilename().AddContext(c)
		}
		fv = arg.AsUserData()
	default:
		return nil, errFileOrFilename().AddContext(c)
	}
	getIoData(t).defaultOutput = fv
	return c.Next(), nil
}

func iolines(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var f *File
	if c.NArgs() == 0 {
		f = getIoData(t).defaultInputFile()
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
	return c.PushingNext(rt.FunctionValue(lines(f, readers, closeAtEOF))), nil
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

	return c.PushingNext(rt.FunctionValue(lines(f, readers, doNotCloseAtEOF))), nil
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
	mode := "r"
	if c.NArgs() >= 2 {
		mode, err = c.StringArg(1)
		if err != nil {
			return nil, err.AddContext(c)
		}
	}
	f, ioErr := OpenFile(fname, mode)
	if ioErr != nil {
		return nil, rt.NewErrorE(ioErr).AddContext(c)
	}
	u := rt.NewUserData(f, getIoData(t).metatable)
	return c.PushingNext(rt.UserDataValue(u)), nil
}

func typef(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	f, err := FileArg(c, 0)
	var val rt.Value
	if err != nil {
		val = rt.NilValue
	} else if f.IsClosed() {
		val = rt.StringValue("closed file")
	} else {
		val = rt.StringValue("file")
	}
	return c.PushingNext(val), nil
}

func iowrite(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	return write(rt.UserDataValue(getIoData(t).defaultOutput), c)
}

func filewrite(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	return write(c.Arg(0), c)
}

func write(vf rt.Value, c *rt.GoCont) (rt.Cont, *rt.Error) {
	f, ok := ValueToFile(vf)
	if !ok {
		return nil, rt.NewErrorS("#1 must be a file").AddContext(c)
	}
	var err error
	for _, val := range c.Etc() {
		switch val.Type() {
		case rt.StringType:
		case rt.IntType:
		case rt.FloatType:
		default:
			return nil, rt.NewErrorS("argument must be a string or a number").AddContext(c)
		}
		s, _ := rt.ToString(val)
		if err = f.WriteString(s); err != nil {
			break
		}
	}
	next := c.Next()
	if err != nil {
		next.Push(rt.StringValue(err.Error()))
	} else {
		next.Push(vf)
	}
	return next, nil
}

func fileseek(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	f, err := FileArg(c, 0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	whence := io.SeekCurrent
	offset := int64(0)
	nargs := c.NArgs()
	if nargs >= 2 {
		whenceName, err := c.StringArg(1)
		if err != nil {
			return nil, err.AddContext(c)
		}
		switch whenceName {
		case "cur":
			whence = io.SeekCurrent
		case "set":
			whence = io.SeekStart
		case "end":
			whence = io.SeekEnd
		default:
			return nil, rt.NewErrorS(`#1 must be "cur", "set" or "end"`).AddContext(c)
		}
	}
	if nargs >= 3 {
		offsetI, err := c.IntArg(2)
		if err != nil {
			return nil, err.AddContext(c)
		}
		offset = int64(offsetI)
	}
	pos, ioErr := f.Seek(offset, whence)
	next := c.Next()
	if ioErr != nil {
		next.Push(rt.NilValue)
		next.Push(rt.StringValue(ioErr.Error()))
	} else {
		next.Push(rt.IntValue(pos))
	}
	return next, nil
}

func tmpfile(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	f, err := TempFile()
	if err != nil {
		return nil, rt.NewErrorE(err).AddContext(c)
	}
	fv := rt.NewUserData(f, getIoData(t).metatable)
	return c.PushingNext(rt.UserDataValue(fv)), nil
}

func tostring(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	f, err := FileArg(c, 0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	s := rt.StringValue(fmt.Sprintf("file(%q)", f.Name()))
	return c.PushingNext(s), nil
}
