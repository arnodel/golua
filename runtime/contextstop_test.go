//go:build !noquotas
// +build !noquotas

package runtime_test

import (
	"bytes"
	"testing"

	"github.com/arnodel/golua/lib"
	rt "github.com/arnodel/golua/runtime"
)

func newStopTestRuntime(t *testing.T) (*rt.Runtime, *rt.Thread) {
	t.Helper()
	var out bytes.Buffer
	r := rt.New(&out)
	t.Cleanup(lib.LoadAll(r))
	return r, r.MainThread()
}

// From #141: hard-stopping the parent context from a child used to leave the
// parent current, marked killed, and running without its limits.
func TestHardStopParentFromChild(t *testing.T) {
	r, th := newStopTestRuntime(t)
	loop, err := r.CompileAndLoadLuaChunk("loop", []byte(`local n = 0 for i = 1, 10000000 do n = n + 1 end return n`), rt.TableValue(r.GlobalEnv()))
	if err != nil {
		t.Fatal(err)
	}
	var ranInner, ranOuter bool
	var inner rt.RuntimeContext
	outer, outerErr := th.CallContext(rt.RuntimeContextDef{HardLimits: rt.RuntimeResources{Cpu: 1000000}}, func() error {
		// CallContext doesn't return here: popping the inner context kills
		// the outer one.
		th.CallContext(rt.RuntimeContextDef{}, func() error {
			inner = th.RuntimeContext()
			inner.Parent().SetStopLevel(rt.HardStop)
			ranInner = true
			return nil
		})
		ranOuter = true
		_, err := rt.Call1(th, rt.FunctionValue(loop))
		return err
	})
	if ranInner || ranOuter {
		t.Errorf("code ran after the stop: inner=%v outer=%v", ranInner, ranOuter)
	}
	if inner.Status() != rt.StatusKilled || outer.Status() != rt.StatusKilled || outerErr == nil {
		t.Errorf("inner=%v outer=%v err=%v, want both killed with an error", inner.Status(), outer.Status(), outerErr)
	}
	if th.RuntimeContext().Status() != rt.StatusLive {
		t.Errorf("the enclosing context should still be live")
	}
}

// Contexts between the current one and the stopped one are killed as control
// returns to them.
func TestHardStopAncestorKillsContextsInBetween(t *testing.T) {
	_, th := newStopTestRuntime(t)
	var ran []string
	outer, _ := th.CallContext(rt.RuntimeContextDef{}, func() error {
		outerCtx := th.RuntimeContext()
		th.CallContext(rt.RuntimeContextDef{}, func() error {
			th.CallContext(rt.RuntimeContextDef{}, func() error {
				outerCtx.SetStopLevel(rt.HardStop)
				ran = append(ran, "inner")
				return nil
			})
			ran = append(ran, "middle")
			return nil
		})
		ran = append(ran, "outer")
		return nil
	})
	if len(ran) > 0 {
		t.Errorf("code ran after the stop: %v", ran)
	}
	if outer.Status() != rt.StatusKilled {
		t.Errorf("outer status = %v, want killed", outer.Status())
	}
}

// A RuntimeContext keeps referring to the same context.
func TestRuntimeContextIdentity(t *testing.T) {
	r, th := newStopTestRuntime(t)
	loop, err := r.CompileAndLoadLuaChunk("loop", []byte(`for i = 1, 100 do end`), rt.TableValue(r.GlobalEnv()))
	if err != nil {
		t.Fatal(err)
	}
	var outerCtx rt.RuntimeContext
	th.CallContext(rt.RuntimeContextDef{HardLimits: rt.RuntimeResources{Cpu: 1000000}}, func() error {
		outerCtx = th.RuntimeContext()
		th.CallContext(rt.RuntimeContextDef{HardLimits: rt.RuntimeResources{Cpu: 1000}}, func() error {
			if got := outerCtx.HardLimits().Cpu; got != 1000000 {
				t.Errorf("outer CPU limit seen from inner context = %d, want 1000000", got)
			}
			if outerCtx.Status() != rt.StatusLive {
				t.Errorf("outer status = %v, want live", outerCtx.Status())
			}
			parent := th.RuntimeContext().Parent()
			if parent.HardLimits().Cpu != 1000000 {
				t.Errorf("Parent() CPU limit = %d, want 1000000", parent.HardLimits().Cpu)
			}
			return nil
		})
		_, err := rt.Call1(th, rt.FunctionValue(loop))
		return err
	})
	if outerCtx.Status() != rt.StatusDone {
		t.Errorf("outer status after it finished = %v, want done", outerCtx.Status())
	}
	if outerCtx.UsedResources().Cpu == 0 {
		t.Errorf("the finished context should report the resources it used")
	}
}

// A soft stop on an ancestor makes it due once it is current again.
func TestSoftStopAncestor(t *testing.T) {
	_, th := newStopTestRuntime(t)
	th.CallContext(rt.RuntimeContextDef{}, func() error {
		outerCtx := th.RuntimeContext()
		th.CallContext(rt.RuntimeContextDef{}, func() error {
			outerCtx.SetStopLevel(rt.SoftStop)
			if th.RuntimeContext().Due() {
				t.Errorf("the inner context should not be due")
			}
			return nil
		})
		if !th.RuntimeContext().Due() {
			t.Errorf("the outer context should be due")
		}
		return nil
	})
}
