package runtime_test

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arnodel/golua/lib"
	rt "github.com/arnodel/golua/runtime"
)

// runInterrupted runs src in a context with the interrupt installed and no
// other limit, triggers the interrupt from another goroutine after a short
// delay, and returns the context's error and how long the run took.
func runInterrupted(t *testing.T, src string) (rt.RuntimeContext, error, time.Duration) {
	ctx, err, elapsed, _ := runInterruptedOutput(t, src)
	return ctx, err, elapsed
}

func runInterruptedOutput(t *testing.T, src string) (rt.RuntimeContext, error, time.Duration, string) {
	t.Helper()
	var out bytes.Buffer
	r := rt.New(&out)
	defer lib.LoadAll(r)()
	chunk, err := r.CompileAndLoadLuaChunk("test", []byte(src), rt.TableValue(r.GlobalEnv()))
	if err != nil {
		t.Fatal(err)
	}
	intr := rt.NewInterrupt()
	timer := time.AfterFunc(50*time.Millisecond, func() { intr.Trigger("interrupted by test") })
	defer timer.Stop()
	start := time.Now()
	thread := r.MainThread()
	ctx, err := thread.CallContext(rt.RuntimeContextDef{Interrupt: intr}, func() error {
		_, err := rt.Call1(thread, rt.FunctionValue(chunk))
		return err
	})
	return ctx, err, time.Since(start), out.String()
}

func TestInterruptStopsAnInfiniteLoop(t *testing.T) {
	ctx, err, elapsed := runInterrupted(t, `while true do end`)
	if err == nil || !strings.Contains(err.Error(), "interrupted by test") {
		t.Fatalf("err = %v, want the interrupt reason", err)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("took %v to stop", elapsed)
	}
	if rt.QuotasAvailable && ctx.Status() != rt.StatusKilled {
		t.Fatalf("status = %v, want killed", ctx.Status())
	}
}

func TestInterruptPassesThroughPcall(t *testing.T) {
	// Each pcall swallows the termination of its own context; the interrupt
	// stays set, so each enclosing context terminates in turn.
	_, err, elapsed, out := runInterruptedOutput(t, `
		local function spin() while true do end end
		pcall(function()
			pcall(function()
				pcall(spin)
				print("level 3 returned")
				spin()
			end)
			print("level 2 returned")
			spin()
		end)
		print("level 1 returned")
		spin()`)
	if err == nil || !strings.Contains(err.Error(), "interrupted by test") {
		t.Fatalf("err = %v, want the interrupt reason", err)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("took %v to stop", elapsed)
	}
	if out != "" {
		t.Fatalf("code ran in an enclosing context after the interrupt: %q", out)
	}
}

func TestInterruptOnlyAffectsItsContext(t *testing.T) {
	// An already-triggered interrupt installed in an inner context ends that
	// context; the outer context carries on.
	var out bytes.Buffer
	r := rt.New(&out)
	defer lib.LoadAll(r)()
	thread := r.MainThread()
	intr := rt.NewInterrupt()
	intr.Trigger("stop inner")
	inner, innerErr := thread.CallContext(rt.RuntimeContextDef{Interrupt: intr}, func() error {
		chunk, _ := r.CompileAndLoadLuaChunk("inner", []byte(`while true do end`), rt.TableValue(r.GlobalEnv()))
		_, err := rt.Call1(thread, rt.FunctionValue(chunk))
		return err
	})
	if innerErr == nil || !strings.Contains(innerErr.Error(), "stop inner") {
		t.Fatalf("inner err = %v", innerErr)
	}
	if rt.QuotasAvailable && inner.Status() != rt.StatusKilled {
		t.Fatalf("inner status = %v", inner.Status())
	}
	chunk, _ := r.CompileAndLoadLuaChunk("outer", []byte(`local n = 0 for i = 1, 1000 do n = n + i end return n`), rt.TableValue(r.GlobalEnv()))
	v, err := rt.Call1(thread, rt.FunctionValue(chunk))
	if err != nil || v.AsInt() != 500500 {
		t.Fatalf("outer = %v, %v", v, err)
	}
}

func TestInterruptTriggerKeepsFirstReason(t *testing.T) {
	intr := rt.NewInterrupt()
	if _, ok := intr.Triggered(); ok {
		t.Fatal("new interrupt is triggered")
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			intr.Trigger("first")
		}()
	}
	wg.Wait()
	intr.Trigger("second")
	if reason, ok := intr.Triggered(); !ok || reason != "first" {
		t.Fatalf("Triggered() = %q, %v", reason, ok)
	}
}
