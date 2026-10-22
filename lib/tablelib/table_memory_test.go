//go:build !noquotas

package tablelib

import (
	rt "github.com/arnodel/golua/runtime"
	"io"
	"testing"
)

func TestReturnedTableHeaderMemory(t *testing.T) {
	for _, name := range []string{"pack", "create", "create-hints"} {
		t.Run(name, func(t *testing.T) {
			r := rt.New(io.Discard)
			defer r.Close(nil)
			th := r.MainThread()
			term := rt.NewTerminationWith(nil, 1, false)
			fn := pack
			nargs := 0
			etc := true
			if name != "pack" {
				fn = create
				nargs = 2
				etc = false
			}
			c := rt.NewGoCont(th, rt.NewGoFunction(fn, name, nargs, etc), term)
			if name != "pack" {
				hint := int64(0)
				if name == "create-hints" {
					hint = 16
				}
				c.Push(r, rt.IntValue(hint))
				c.Push(r, rt.IntValue(hint))
			}
			r.PushContext(rt.RuntimeContextDef{HardLimits: rt.RuntimeResources{Memory: 1 << 20}})
			defer r.PopContext()
			if _, err := fn(th, c); err != nil {
				t.Fatal(err)
			}
			want := rt.TableHeaderSize
			if name == "pack" {
				raw := rt.NewTable()
				want += raw.Set(rt.StringValue("n"), rt.IntValue(0))
			}
			if got := r.UsedResources().Memory; got != want {
				t.Fatalf("charged %d bytes, want header plus storage %d", got, want)
			}
		})
	}
}
