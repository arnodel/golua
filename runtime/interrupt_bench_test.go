package runtime_test

import (
	"bytes"
	"testing"

	"github.com/arnodel/golua/lib"
	rt "github.com/arnodel/golua/runtime"
)

// runLoop runs a 2M-iteration Lua loop, b.N times, in a context built from
// def.
func runLoop(b *testing.B, def rt.RuntimeContextDef) {
	var out bytes.Buffer
	r := rt.New(&out)
	defer lib.LoadAll(r)()
	chunk, err := r.CompileAndLoadLuaChunk("b",
		[]byte(`local s = 0 for i = 1, 2000000 do s = s + i % 7 end return s`),
		rt.TableValue(r.GlobalEnv()))
	if err != nil {
		b.Fatal(err)
	}
	th := r.MainThread()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := th.CallContext(def, func() error {
			_, err := rt.Call1(th, rt.FunctionValue(chunk))
			return err
		}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoopNoLimit(b *testing.B) { runLoop(b, rt.RuntimeContextDef{}) }

// An Interrupt that is never triggered must cost (almost) nothing.
func BenchmarkLoopInterrupt(b *testing.B) {
	runLoop(b, rt.RuntimeContextDef{Interrupt: rt.NewInterrupt()})
}

// The CPU limit is large enough never to be reached.
func BenchmarkLoopCPULimit(b *testing.B) {
	runLoop(b, rt.RuntimeContextDef{HardLimits: rt.RuntimeResources{Cpu: 1 << 40}})
}

func BenchmarkLoopCPULimitInterrupt(b *testing.B) {
	runLoop(b, rt.RuntimeContextDef{
		HardLimits: rt.RuntimeResources{Cpu: 1 << 40},
		Interrupt:  rt.NewInterrupt(),
	})
}
