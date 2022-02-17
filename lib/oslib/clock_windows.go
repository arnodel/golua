//go:build windows
// +build windows

package oslib

import (
	"time"

	rt "github.com/arnodel/golua/runtime"
)

var startTime time.Time

func clock(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	// No syscall.Getrusage on windows.  As a fallback return clock time since
	// starting the program.
	time := float64(time.Now().Sub(startTime).Microseconds()) / 1e6
	return c.PushingNext1(t.Runtime, rt.FloatValue(time)), nil
}

func init() {
	startTime = time.Now()
}
