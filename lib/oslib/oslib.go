package oslib

import (
	"os"
	"syscall"
	"time"

	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
	"github.com/arnodel/golua/safeio"
)

// LibLoader can load the os lib.
var LibLoader = packagelib.Loader{
	Load: load,
	Name: "os",
}

func load(r *rt.Runtime) rt.Value {
	pkg := rt.NewTable()

	rt.SolemnlyDeclareCompliance(
		rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyTimeSafe|rt.ComplyIoSafe,

		r.SetEnvGoFunc(pkg, "clock", clock, 0, false),
		r.SetEnvGoFunc(pkg, "time", timef, 1, false),
		r.SetEnvGoFunc(pkg, "setlocale", setlocale, 2, false),
		r.SetEnvGoFunc(pkg, "getenv", getenv, 1, false),
		r.SetEnvGoFunc(pkg, "tmpname", tmpname, 0, false),
		r.SetEnvGoFunc(pkg, "remove", remove, 1, false),
		r.SetEnvGoFunc(pkg, "rename", rename, 2, false),
	)

	return rt.TableValue(pkg)
}

func clock(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var rusage syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &rusage) // ignore errors
	time := float64(rusage.Utime.Sec+rusage.Stime.Sec) + float64(rusage.Utime.Usec+rusage.Stime.Usec)/1000000.0
	return c.PushingNext1(t.Runtime, rt.FloatValue(time)), nil
}

func timef(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if c.NArgs() == 0 {
		now := time.Now().UTC().Unix()
		return c.PushingNext1(t.Runtime, rt.IntValue(now)), nil
	}
	return nil, rt.NewErrorS("time(tbl) unimplemented")
}

func setlocale(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	locale, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	// Just pretend we can set the "C" locale and none other
	if locale != "C" {
		return c.PushingNext1(t.Runtime, rt.NilValue), nil
	}
	return c.PushingNext1(t.Runtime, rt.StringValue(locale)), nil
}

func getenv(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	name, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	val, ok := os.LookupEnv(name)
	valV := rt.NilValue
	if ok {
		t.RequireBytes(len(val))
		valV = rt.StringValue(val)
	}
	return c.PushingNext1(t.Runtime, valV), nil
}

func tmpname(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	f, ioErr := safeio.TempFile(t.Runtime, "", "")
	if ioErr != nil {
		return t.ProcessIoError(c.Next(), ioErr)
	}
	defer f.Close()
	name := f.Name()
	t.RequireBytes(len(name))
	return c.PushingNext1(t.Runtime, rt.StringValue(name)), nil
}

func remove(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	name, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	ioErr := safeio.RemoveFile(t.Runtime, name)
	if ioErr != nil {
		return t.ProcessIoError(c.Next(), ioErr)
	}
	return c.PushingNext1(t.Runtime, rt.BoolValue(true)), nil
}

func rename(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}
	oldName, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	newName, err := c.StringArg(1)
	if err != nil {
		return nil, err
	}
	ioErr := safeio.RenameFile(t.Runtime, oldName, newName)
	if ioErr != nil {
		return t.ProcessIoError(c.Next(), ioErr)
	}
	return c.PushingNext1(t.Runtime, rt.BoolValue(true)), nil
}
