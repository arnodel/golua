//go:build !noquotas

package debuglib

import (
	rt "github.com/arnodel/golua/runtime"
	"io"
	"testing"
	"unsafe"
)

func TestGetinfoTableHeaderMemory(t *testing.T) {
	r := rt.New(io.Discard)
	defer r.Close(nil)
	th := r.MainThread()
	term := rt.NewTerminationWith(nil, 1, false)
	c := rt.NewGoCont(th, rt.NewGoFunction(getinfo, "getinfo", 3, false), term)
	target := rt.NewGoFunction(func(_ *rt.Thread, _ *rt.GoCont) (rt.Cont, error) { return nil, nil }, "target", 0, false)
	c.Push(r, rt.FunctionValue(target))
	raw := rt.NewTable()
	storage := raw.Set(rt.StringValue("name"), rt.StringValue("target")) + raw.Set(rt.StringValue("currentline"), rt.IntValue(0)) + raw.Set(rt.StringValue("source"), rt.StringValue("[Go]"))
	r.PushContext(rt.RuntimeContextDef{HardLimits: rt.RuntimeResources{Memory: 1 << 20}})
	defer r.PopContext()
	if _, err := getinfo(th, c); err != nil {
		t.Fatal(err)
	}
	if got, want := r.UsedResources().Memory, rt.TableHeaderSize+storage+uint64(unsafe.Sizeof(rt.GoCont{})); got != want {
		t.Fatalf("charged %d bytes, want header plus fields and continuation %d", got, want)
	}
}
