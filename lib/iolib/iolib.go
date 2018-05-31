package iolib

import (
	"io"
	"os"

	rt "github.com/arnodel/golua/runtime"
)

type ioKeyType struct{}

var ioKey = ioKeyType{}

// Load io library
func Load(r *rt.Runtime) {
	methods := rt.NewTable()
	rt.SetEnvGoFunc(methods, "read", fileread, 1, true)
	rt.SetEnvGoFunc(methods, "lines", filelines, 1, true)
	rt.SetEnvGoFunc(methods, "close", closef, 1, false)
	rt.SetEnvGoFunc(methods, "flush", flush, 1, false)
	// TODO: seek
	// TODO: setvbuf
	rt.SetEnvGoFunc(methods, "write", filewrite, 1, true)

	meta := rt.NewTable()
	rt.SetEnv(meta, "__index", methods)

	r.SetRegistry(ioKey, &ioData{
		defaultOutput: rt.NewUserData(&File{file: os.Stdout}, meta),
		defaultInput:  rt.NewUserData(&File{file: os.Stdin}, meta),
		metatable:     methods,
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
	rt.SetEnvGoFunc(pkg, "write", iowrite, 0, true)
}

type ioData struct {
	defaultOutput rt.Value
	defaultInput  rt.Value
	metatable     *rt.Table
}

func getIoData(t *rt.Thread) *ioData {
	return t.Registry(ioKey).(*ioData)
}

func (d *ioData) defaultOutputFile() *File {
	f, _ := ValueToFile(d.defaultOutput)
	return f
}

func (d *ioData) defaultInputFile() *File {
	f, _ := ValueToFile(d.defaultInput)
	return f
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

func flush(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
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

var errFileOrFilename = rt.NewErrorS("#1 must be a file or a filename")

func input(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	ioData := getIoData(t)
	if c.NArgs() == 0 {
		return c.PushingNext(ioData.defaultInput), nil
	}
	var fv rt.Value
	switch x := c.Arg(0).(type) {
	case rt.String:
		f, ioErr := OpenFile(string(x), "r")
		if ioErr != nil {
			return nil, rt.NewErrorE(ioErr).AddContext(c)
		}
		fv = rt.NewUserData(f, ioData.metatable)
	case *rt.UserData:
		_, err := FileArg(c, 0)
		if err != nil {
			return nil, errFileOrFilename.AddContext(c)
		}
		fv = x
	default:
		return nil, errFileOrFilename.AddContext(c)
	}
	ioData.defaultInput = fv
	return c.Next(), nil
}

func output(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	ioData := getIoData(t)
	if c.NArgs() == 0 {
		return c.PushingNext(ioData.defaultOutput), nil
	}
	var fv rt.Value
	switch x := c.Arg(0).(type) {
	case rt.String:
		f, ioErr := OpenFile(string(x), "w")
		if ioErr != nil {
			return nil, rt.NewErrorE(ioErr).AddContext(c)
		}
		fv = rt.NewUserData(f, ioData.metatable)
	case *rt.UserData:
		_, err := FileArg(c, 0)
		if err != nil {
			return nil, errFileOrFilename.AddContext(c)
		}
		fv = x
	default:
		return nil, errFileOrFilename.AddContext(c)
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
	return write(getIoData(t).defaultOutput, c)
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
		switch val.(type) {
		case rt.String:
		case rt.Int:
		case rt.Float:
		default:
			return nil, rt.NewErrorS("argument must be a string or a number").AddContext(c)
		}
		s, _ := rt.AsString(val)
		if err = f.WriteString(string(s)); err != nil {
			break
		}
	}
	next := c.Next()
	if err != nil {
		next.Push(rt.String(err.Error()))
	} else {
		next.Push(vf)
	}
	return next, nil
}
