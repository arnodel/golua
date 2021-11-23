package oslib

import (
	"syscall"
	"time"

	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
)

// LibLoader can load the os lib.
var LibLoader = packagelib.Loader{
	Load: load,
	Name: "os",
}

func load(r *rt.Runtime) rt.Value {
	pkg := rt.NewTable()

	rt.SolemnlyDeclareSafetyFlags(
		rt.RCS_CpuSafe|rt.RCS_MemSafe|rt.RCS_IOSafe,

		r.SetEnvGoFunc(pkg, "clock", clock, 0, false),
		r.SetEnvGoFunc(pkg, "time", timef, 1, false),
		r.SetEnvGoFunc(pkg, "setlocale", setlocale, 2, false),
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
		return nil, err.AddContext(c)
	}
	locale, err := c.StringArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	// Just pretend we can set the "C" locale and none other
	if locale != "C" {
		return c.PushingNext1(t.Runtime, rt.NilValue), nil
	}
	return c.PushingNext1(t.Runtime, rt.StringValue(locale)), nil
}
