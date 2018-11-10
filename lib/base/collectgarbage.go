package base

import (
	"runtime"
	"runtime/debug"

	rt "github.com/arnodel/golua/runtime"
)

var gcPercent int
var gcRunning bool

func collectgarbage(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	opt := "collect"
	if c.NArgs() > 0 {
		optv, err := c.StringArg(0)
		if err != nil {
			return nil, err.AddContext(c)
		}
		opt = string(optv)
	}
	next := c.Next()
	switch opt {
	case "collect":
		runtime.GC()
	case "step":
		// The Go runtime doesn't offer the ability to go gc steps.
		runtime.GC()
		next.Push(rt.Bool(true))
	case "stop":
		debug.SetGCPercent(-1)
		gcRunning = false
	case "restart":
		debug.SetGCPercent(gcPercent)
		gcRunning = gcPercent != -1
	case "isrunning":
		next.Push(rt.Bool(gcRunning))
	case "setpause":
		// TODO: perhaps change gcPercent to reflect this?
	case "setstepmul":
		// TODO: perhaps change gcPercent to reflect this?
	case "count":
		stats := runtime.MemStats{}
		runtime.ReadMemStats(&stats)
		next.Push(rt.Float(stats.Alloc / 1024.0))
	default:
		return nil, rt.NewErrorS("invalid option").AddContext(c)
	}
	return next, nil
}

func init() {
	gcPercent = debug.SetGCPercent(-1)
	gcRunning = gcPercent != -1
	debug.SetGCPercent(gcPercent)
}
