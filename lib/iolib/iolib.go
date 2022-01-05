package iolib

import (
	"fmt"
	"io"
	"os"

	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
)

// BufferedStdFiles sets wether std files should be buffered
var BufferedStdFiles bool = true

type ioKeyType struct{}

var ioKey = rt.AsValue(ioKeyType{})

// LibLoader can load the io lib.
var LibLoader = packagelib.Loader{
	Load: load,
	Name: "io",
}

func load(r *rt.Runtime) (rt.Value, func()) {
	methods := rt.NewTable()

	rt.SolemnlyDeclareCompliance(
		rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyIoSafe,

		r.SetEnvGoFunc(methods, "read", fileread, 1, true),
		r.SetEnvGoFunc(methods, "lines", filelines, 1, true),
		r.SetEnvGoFunc(methods, "close", fileclose, 1, false),
		r.SetEnvGoFunc(methods, "flush", fileflush, 1, false),
		r.SetEnvGoFunc(methods, "seek", fileseek, 3, false),
		r.SetEnvGoFunc(methods, "setvbuf", filesetvbuf, 2, false),
		// TODO: setvbuf,
		r.SetEnvGoFunc(methods, "write", filewrite, 1, true),
	)

	meta := rt.NewTable()
	r.SetEnv(meta, "__name", rt.StringValue("file"))
	r.SetEnv(meta, "__index", rt.TableValue(methods))

	rt.SolemnlyDeclareCompliance(
		rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyTimeSafe|rt.ComplyTimeSafe|rt.ComplyIoSafe,

		r.SetEnvGoFunc(meta, "__tostring", tostring, 1, false),
	)

	var (
		stdoutOpts int
		stderrOpts int
		stdinOpts  int
	)
	if BufferedStdFiles {
		stdoutOpts = bufferedWrite
		stdinOpts = bufferedRead
	}

	stdoutFile := NewFile(os.Stdout, stdoutOpts)
	stderrFile := NewFile(os.Stderr, stderrOpts)
	// This is not a good pattern - it has to do for now.
	if r.Stdout == nil {
		r.Stdout = stdoutFile.writer
	}
	stdin := newFileUserData(NewFile(os.Stdin, stdinOpts), meta)
	stdout := newFileUserData(stdoutFile, meta)
	stderr := newFileUserData(stderrFile, meta) // I''m guessing, don't buffer stderr?

	r.SetRegistry(ioKey, rt.AsValue(&ioData{
		defaultOutput: stdout,
		defaultInput:  stdin,
		metatable:     meta,
	}))
	pkg := rt.NewTable()
	r.SetEnv(pkg, "stdin", rt.UserDataValue(stdin))
	r.SetEnv(pkg, "stdout", rt.UserDataValue(stdout))
	r.SetEnv(pkg, "stderr", rt.UserDataValue(stderr))

	rt.SolemnlyDeclareCompliance(
		rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyIoSafe,

		r.SetEnvGoFunc(pkg, "close", ioclose, 1, false),
		r.SetEnvGoFunc(pkg, "flush", ioflush, 0, false),
		r.SetEnvGoFunc(pkg, "input", input, 1, false),
		r.SetEnvGoFunc(pkg, "lines", iolines, 1, true),
		r.SetEnvGoFunc(pkg, "open", open, 2, false),
		r.SetEnvGoFunc(pkg, "output", output, 1, false),
		// TODO: popen,
		r.SetEnvGoFunc(pkg, "read", ioread, 0, true),
		r.SetEnvGoFunc(pkg, "tmpfile", tmpfile, 0, false),
		r.SetEnvGoFunc(pkg, "write", iowrite, 0, true),
	)

	rt.SolemnlyDeclareCompliance(
		rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyTimeSafe|rt.ComplyIoSafe,

		r.SetEnvGoFunc(pkg, "type", typef, 1, false),
	)

	// This function should make sure known buffers are flushed before quitting
	var cleanup = func() {
		getIoData(r).defaultOutputFile().Flush()
		stdoutFile.Flush()
		stderrFile.Flush()
	}

	return rt.TableValue(pkg), cleanup
}

type ioData struct {
	defaultOutput *rt.UserData
	defaultInput  *rt.UserData
	metatable     *rt.Table
}

func getIoData(r *rt.Runtime) *ioData {
	return r.Registry(ioKey).Interface().(*ioData)
}

func (d *ioData) defaultOutputFile() *File {
	return d.defaultOutput.Value().(*File)
}

func (d *ioData) defaultInputFile() *File {
	return d.defaultInput.Value().(*File)
}

func pushingNextIoResult(r *rt.Runtime, c *rt.GoCont, ioErr error) (rt.Cont, *rt.Error) {
	next := c.Next()
	if ioErr != nil {
		return r.ProcessIoError(next, ioErr)
	}
	r.Push1(next, rt.BoolValue(true))
	return next, nil
}

func ioclose(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var f *File
	if c.NArgs() == 0 {
		f = getIoData(t.Runtime).defaultOutputFile()
	} else {
		var err *rt.Error
		f, err = FileArg(c, 0)
		if err != nil {
			return nil, err
		}
	}
	return pushingNextIoResult(t.Runtime, c, f.Close())
}

func fileclose(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	return ioclose(t, c)
}

func ioflush(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var f *File
	if c.NArgs() == 0 {
		f = getIoData(t.Runtime).defaultOutputFile()
	} else {
		var err *rt.Error
		f, err = FileArg(c, 0)
		if err != nil {
			return nil, err
		}
	}
	return pushingNextIoResult(t.Runtime, c, f.Flush())
}

func fileflush(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	return ioflush(t, c)
}

func errFileOrFilename() *rt.Error {
	return rt.NewErrorS("#1 must be a file or a filename")
}

func input(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	ioData := getIoData(t.Runtime)
	if c.NArgs() == 0 {
		return c.PushingNext1(t.Runtime, rt.UserDataValue(ioData.defaultInput)), nil
	}
	var (
		fv  *rt.UserData
		arg = c.Arg(0)
	)
	switch arg.Type() {
	case rt.StringType:
		f, ioErr := OpenFile(t.Runtime, arg.AsString(), "r")
		if ioErr != nil {
			return nil, rt.NewErrorE(ioErr)
		}
		fv = newFileUserData(f, ioData.metatable)
	case rt.UserDataType:
		_, err := FileArg(c, 0)
		if err != nil {
			return nil, errFileOrFilename()
		}
		fv = arg.AsUserData()
	default:
		return nil, errFileOrFilename()
	}
	ioData.defaultInput = fv
	return c.PushingNext1(t.Runtime, rt.UserDataValue(fv)), nil
}

func output(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	ioData := getIoData(t.Runtime)
	if c.NArgs() == 0 {
		return c.PushingNext1(t.Runtime, rt.UserDataValue(ioData.defaultOutput)), nil
	}
	var (
		fv  *rt.UserData
		arg = c.Arg(0)
	)
	switch arg.Type() {
	case rt.StringType:
		f, ioErr := OpenFile(t.Runtime, arg.AsString(), "w")
		if ioErr != nil {
			return nil, rt.NewErrorE(ioErr)
		}
		fv = newFileUserData(f, ioData.metatable)
	case rt.UserDataType:
		_, err := FileArg(c, 0)
		if err != nil {
			return nil, errFileOrFilename()
		}
		fv = arg.AsUserData()
	default:
		return nil, errFileOrFilename()
	}
	// Make sure the current output is flushed
	ioData.defaultOutput.Value().(*File).Flush()
	ioData.defaultOutput = fv
	return c.PushingNext1(t.Runtime, rt.UserDataValue(fv)), nil
}

func iolines(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var (
		f         *File
		eofAction = closeAtEOF
	)
	if c.NArgs() == 0 || c.Arg(0) == rt.NilValue {
		f = getIoData(t.Runtime).defaultInputFile()
		eofAction = doNotCloseAtEOF
	} else {
		fname, err := c.StringArg(0)
		if err != nil {
			return nil, err
		}
		var ioErr error
		f, ioErr = OpenFile(t.Runtime, string(fname), "r")
		if ioErr != nil {
			return nil, rt.NewErrorE(ioErr)
		}
	}
	readers, fmtErr := getFormatReaders(c.Etc())
	if fmtErr != nil {
		return nil, rt.NewErrorE(fmtErr)
	}
	return c.PushingNext(t.Runtime, rt.FunctionValue(lines(t.Runtime, f, readers, eofAction))), nil
}

func filelines(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	f, err := FileArg(c, 0)
	if err != nil {
		return nil, err
	}
	readers, fmtErr := getFormatReaders(c.Etc())
	if fmtErr != nil {
		return nil, rt.NewErrorE(fmtErr)
	}

	return c.PushingNext(t.Runtime, rt.FunctionValue(lines(t.Runtime, f, readers, doNotCloseAtEOF))), nil
}

const (
	closeAtEOF      = 1
	doNotCloseAtEOF = 0
)

func lines(r *rt.Runtime, f *File, readers []formatReader, flags int) *rt.GoFunction {
	if len(readers) == 0 {
		readers = []formatReader{lineReader(false)}
	}
	iterator := func(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
		next := c.Next()
		// if f.closed {
		// 	return next, nil
		// }
		err := read(r, f, readers, next)
		if err != nil {
			if err == io.EOF {
				if flags&closeAtEOF != 0 {
					if err := f.Close(); err != nil {
						return t.ProcessIoError(next, err)
					}
				}
				t.Push1(next, rt.NilValue)
				return next, nil
			}
			return nil, rt.NewErrorE(err)
		}
		return next, nil
	}
	iterGof := rt.NewGoFunction(iterator, "linesiterator", 0, false)
	iterGof.SolemnlyDeclareCompliance(rt.ComplyCpuSafe | rt.ComplyMemSafe | rt.ComplyIoSafe)
	return iterGof

}

func open(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	fname, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	mode := "r"
	if c.NArgs() >= 2 {
		mode, err = c.StringArg(1)
		if err != nil {
			return nil, err
		}
	}
	f, ioErr := OpenFile(t.Runtime, fname, mode)
	if ioErr != nil {
		return pushingNextIoResult(t.Runtime, c, ioErr)
	}
	u := newFileUserData(f, getIoData(t.Runtime).metatable)
	return c.PushingNext(t.Runtime, rt.UserDataValue(u)), nil
}

func typef(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
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
	return c.PushingNext(t.Runtime, val), nil
}

func iowrite(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	return write(t.Runtime, rt.UserDataValue(getIoData(t.Runtime).defaultOutput), c)
}

func filewrite(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	return write(t.Runtime, c.Arg(0), c)
}

func write(r *rt.Runtime, vf rt.Value, c *rt.GoCont) (rt.Cont, *rt.Error) {
	f, ok := ValueToFile(vf)
	if !ok {
		return nil, rt.NewErrorS("#1 must be a file")
	}
	if f.closed {
		return nil, rt.NewErrorE(errFileAlreadyClosed)
	}
	var err error
	for _, val := range c.Etc() {
		switch val.Type() {
		case rt.StringType:
		case rt.IntType:
		case rt.FloatType:
		default:
			return nil, rt.NewErrorS("argument must be a string or a number")
		}
		s, _ := val.ToString()
		if err = f.WriteString(s); err != nil {
			break
		}
	}
	next := c.Next()
	if err != nil {
		return r.ProcessIoError(next, err)
	} else {
		r.Push(next, vf)
	}
	return next, nil
}

func fileseek(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	f, err := FileArg(c, 0)
	if err != nil {
		return nil, err
	}
	whence := io.SeekCurrent
	offset := int64(0)
	nargs := c.NArgs()
	if nargs >= 2 {
		whenceName, err := c.StringArg(1)
		if err != nil {
			return nil, err
		}
		switch whenceName {
		case "cur":
			whence = io.SeekCurrent
		case "set":
			whence = io.SeekStart
		case "end":
			whence = io.SeekEnd
		default:
			return nil, rt.NewErrorS(`#1 must be "cur", "set" or "end"`)
		}
	}
	if nargs >= 3 {
		offsetI, err := c.IntArg(2)
		if err != nil {
			return nil, err
		}
		offset = int64(offsetI)
	}
	pos, ioErr := f.Seek(offset, whence)
	next := c.Next()
	if ioErr != nil {
		t.Push1(next, rt.NilValue)
		t.Push1(next, rt.StringValue(ioErr.Error()))
	} else {
		t.Push1(next, rt.IntValue(pos))
	}
	return next, nil
}

func filesetvbuf(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}
	f, err := FileArg(c, 0)
	if err != nil {
		return nil, err
	}
	mode, err := c.StringArg(1)
	if err != nil {
		return nil, err
	}
	var size int64
	if c.NArgs() > 2 {
		size, err = c.IntArg(2)
		if err != nil {
			return nil, err
		}
	}
	bufErr := f.SetWriteBuffer(mode, int(size))
	if bufErr != nil {
		return nil, rt.NewErrorE(bufErr)
	}
	return c.PushingNext1(t.Runtime, rt.BoolValue(true)), nil
}

func tmpfile(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	f, err := TempFile(t.Runtime)
	if err != nil {
		return nil, rt.NewErrorE(err)
	}
	fv := newFileUserData(f, getIoData(t.Runtime).metatable)
	return c.PushingNext(t.Runtime, rt.UserDataValue(fv)), nil
}

func tostring(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	f, err := FileArg(c, 0)
	if err != nil {
		return nil, err
	}
	var s string
	if f.closed {
		s = "file (closed)"
	} else {
		s = fmt.Sprintf("file (%q)", f.Name())
	}
	t.RequireBytes(len(s))
	return c.PushingNext(t.Runtime, rt.StringValue(s)), nil
}

func newFileUserData(f *File, meta *rt.Table) *rt.UserData {
	return rt.NewUserData(f, meta)
}
