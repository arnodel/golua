package base

import (
	"errors"
	"runtime"
	"runtime/debug"

	rt "github.com/arnodel/golua/runtime"
)

var gcPercent int
var gcRunning bool
var gcMode = "incremental" // Track GC mode for compatibility (Go's GC is always incremental)

// GC parameters for Lua compatibility (Go manages GC, so these are just tracked)
var gcParams = map[string]int64{
	"pause":    200, // Default pause value in Lua
	"stepmul":  200, // Default stepmul value in Lua
	"stepsize": 13,  // Default stepsize (log2 of 8KB)
}

func collectgarbage(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	opt := "collect"
	if c.NArgs() > 0 {
		optv, err := c.StringArg(0)
		if err != nil {
			return nil, err
		}
		opt = string(optv)
	}
	next := c.Next()
	switch opt {
	case "collect":
		t.CollectGarbage()
	case "step":
		// The Go runtime doesn't offer the ability to go gc steps.
		t.CollectGarbage()
		t.Push1(next, rt.BoolValue(true))
	case "stop":
		debug.SetGCPercent(-1)
		gcRunning = false
	case "restart":
		debug.SetGCPercent(gcPercent)
		gcRunning = gcPercent != -1
	case "isrunning":
		t.Push1(next, rt.BoolValue(gcRunning))
	case "setpause":
		// TODO: perhaps change gcPercent to reflect this?
	case "setstepmul":
		// TODO: perhaps change gcPercent to reflect this?
	case "count":
		stats := runtime.MemStats{}
		runtime.ReadMemStats(&stats)
		t.Push1(next, rt.FloatValue(float64(stats.Alloc)/1024.0))
	case "incremental", "generational":
		// Go's GC is always incremental, so these are no-ops.
		// Track and return the mode for Lua compatibility.
		prevMode := gcMode
		gcMode = opt
		t.Push1(next, rt.StringValue(prevMode))
	case "param":
		// Lua 5.5 "param" option: collectgarbage("param", name [, value])
		// Go manages GC, so we just track these values for compatibility.
		if c.NArgs() < 2 {
			return nil, errors.New("missing parameter name")
		}
		paramName, err := c.StringArg(1)
		if err != nil {
			return nil, err
		}
		name := string(paramName)
		if _, ok := gcParams[name]; !ok {
			return nil, errors.New("invalid parameter name")
		}
		prevVal := gcParams[name]
		if c.NArgs() >= 3 {
			newVal, err := c.IntArg(2)
			if err != nil {
				return nil, err
			}
			gcParams[name] = newVal
		}
		t.Push1(next, rt.IntValue(prevVal))
	default:
		return nil, errors.New("invalid option")
	}
	return next, nil
}

func init() {
	gcPercent = debug.SetGCPercent(-1)
	gcRunning = gcPercent != -1
	debug.SetGCPercent(gcPercent)
}
