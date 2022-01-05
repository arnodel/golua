package oslib

import (
	"os"
	"syscall"
	"time"

	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
	"github.com/arnodel/golua/safeio"
	"github.com/arnodel/strftime"
)

// LibLoader can load the os lib.
var LibLoader = packagelib.Loader{
	Load: load,
	Name: "os",
}

func load(r *rt.Runtime) (rt.Value, func()) {
	pkg := rt.NewTable()

	rt.SolemnlyDeclareCompliance(
		rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyTimeSafe|rt.ComplyIoSafe,

		r.SetEnvGoFunc(pkg, "clock", clock, 0, false),
		r.SetEnvGoFunc(pkg, "date", date, 2, false),
		r.SetEnvGoFunc(pkg, "difftime", difftime, 2, false),
		r.SetEnvGoFunc(pkg, "time", timef, 1, false),
		r.SetEnvGoFunc(pkg, "getenv", getenv, 1, false),
		r.SetEnvGoFunc(pkg, "tmpname", tmpname, 0, false),
		r.SetEnvGoFunc(pkg, "remove", remove, 1, false),
		r.SetEnvGoFunc(pkg, "rename", rename, 2, false),
	)
	// These functions are not safe - I don't know what compliance category to
	// put them in.
	r.SetEnvGoFunc(pkg, "setlocale", setlocale, 2, false)
	r.SetEnvGoFunc(pkg, "exit", exit, 2, false)
	return rt.TableValue(pkg), nil
}

func clock(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var rusage syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &rusage) // ignore errors
	time := float64(rusage.Utime.Sec+rusage.Stime.Sec) + float64(rusage.Utime.Usec+rusage.Stime.Usec)/1000000.0
	return c.PushingNext1(t.Runtime, rt.FloatValue(time)), nil
}

func date(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var (
		err    *rt.Error
		utc    bool
		now    time.Time
		format string
		date   rt.Value
	)
	if err = c.Check1Arg(); err != nil {
		return nil, err
	}
	format, err = c.StringArg(0)
	if err != nil {
		return nil, err
	}

	// If format starts with "!" it means UTC
	if len(format) > 0 && format[0] == '!' {
		utc = true
		format = format[1:]
	}

	// Get the time value
	if c.NArgs() > 1 {
		var t int64
		t, err = c.IntArg(1)
		if err != nil {
			return nil, err
		}
		now = time.Unix(t, 0)
	} else {
		now = time.Now()
	}
	if utc {
		now = now.UTC()
	}
	switch format {
	case "*t":
		{
			tbl := rt.NewTable()
			setTableFields(t.Runtime, tbl, now)
			date = rt.TableValue(tbl)
		}
	default:
		{
			dateStr, fmtErr := strftime.StrictFormat(format, now)
			if fmtErr != nil {
				return nil, rt.NewErrorE(fmtErr)
			}
			date = rt.StringValue(dateStr)
		}
	}
	return c.PushingNext1(t.Runtime, date), nil
}

func difftime(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}
	t2, err := c.IntArg(0)
	if err != nil {
		return nil, err
	}
	t1, err := c.IntArg(1)
	if err != nil {
		return nil, err
	}
	return c.PushingNext1(t.Runtime, rt.IntValue(t2-t1)), nil
}

func exit(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var (
		code  = 0 // 0 for success, 1 for failure
		close = false
	)
	if c.NArgs() > 0 {
		if !rt.Truth(c.Arg(0)) {
			code = 1
		}
	}
	if c.NArgs() > 1 {
		close = rt.Truth(c.Arg(1))
	}
	if close {
		// TODO: "close" the runtime, i.e. cleanup.
		_ = close
	}
	os.Exit(code)
	return nil, nil
}

func timef(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if c.NArgs() == 0 {
		now := time.Now().Unix()
		return c.PushingNext1(t.Runtime, rt.IntValue(now)), nil
	}
	tbl, err := c.TableArg(0)
	if err != nil {
		return nil, err
	}
	var fieldErr *rt.Error
	var getField = func(dest *int, name string, required bool) bool {
		if fieldErr != nil {
			return false
		}
		var val rt.Value
		val, fieldErr = rt.Index(t, rt.TableValue(tbl), rt.StringValue(name))
		if fieldErr != nil {
			return false
		}
		if val == rt.NilValue {
			if required {
				fieldErr = rt.NewErrorF("required field '%s' missing", name)
				return false
			}
			return true
		}
		iVal, ok := val.TryInt()
		if !ok {
			fieldErr = rt.NewErrorF("field '%s' is not an integer", name)
			return false
		}
		*dest = int(iVal)
		return true
	}
	var (
		year, month, day int
		hour, min, sec   = 12, 0, 0
	)
	ok := getField(&year, "year", true) &&
		getField(&month, "month", true) &&
		getField(&day, "day", true) &&
		getField(&hour, "hour", false) &&
		getField(&min, "min", false) &&
		getField(&sec, "sec", false)
	if !ok {
		return nil, fieldErr
	}
	// TODO: deal with DST - I have no idea how to do that.

	date := time.Date(year, time.Month(month), day, hour, min, sec, 0, time.Local)
	setTableFields(t.Runtime, tbl, date)
	return c.PushingNext1(t.Runtime, rt.IntValue(date.Unix())), nil
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

//
// Utils
//

func setTableFields(r *rt.Runtime, tbl *rt.Table, now time.Time) {
	r.SetEnv(tbl, "year", rt.IntValue(int64(now.Year())))
	r.SetEnv(tbl, "month", rt.IntValue(int64(now.Month())))
	r.SetEnv(tbl, "day", rt.IntValue(int64(now.Day())))
	r.SetEnv(tbl, "hour", rt.IntValue(int64(now.Hour())))
	r.SetEnv(tbl, "min", rt.IntValue(int64(now.Minute())))
	r.SetEnv(tbl, "sec", rt.IntValue(int64(now.Second())))
	// Weeks start on Sunday according to Lua!
	wday := now.Weekday() + 1
	if wday == 8 {
		wday = 1
	}
	r.SetEnv(tbl, "wday", rt.IntValue(int64(wday)))
	r.SetEnv(tbl, "yday", rt.IntValue(int64(now.YearDay())))
	r.SetEnv(tbl, "isdst", rt.BoolValue(now.IsDST()))

}
