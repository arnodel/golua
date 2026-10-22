//go:build !noquotas

package oslib

import (
	rt "github.com/arnodel/golua/runtime"
	"io"
	"testing"
	"time"
)

func TestDateTableHeaderMemory(t *testing.T) {
	r := rt.New(io.Discard)
	defer r.Close(nil)
	th := r.MainThread()
	term := rt.NewTerminationWith(nil, 1, false)
	c := rt.NewGoCont(th, rt.NewGoFunction(date, "date", 2, false), term)
	c.Push(r, rt.StringValue("!*t"))
	c.Push(r, rt.IntValue(0))
	// Measure the existing field storage independently of the header charge.
	raw := rt.NewTable()
	r.PushContext(rt.RuntimeContextDef{HardLimits: rt.RuntimeResources{Memory: 1 << 20}})
	setTableFields(r, raw, time.Unix(0, 0).UTC())
	storage := r.UsedResources().Memory
	r.PopContext()
	r.PushContext(rt.RuntimeContextDef{HardLimits: rt.RuntimeResources{Memory: 1 << 20}})
	defer r.PopContext()
	if _, err := date(th, c); err != nil {
		t.Fatal(err)
	}
	if got, want := r.UsedResources().Memory, rt.TableHeaderSize+storage; got != want {
		t.Fatalf("charged %d bytes, want header plus fields %d", got, want)
	}
}
