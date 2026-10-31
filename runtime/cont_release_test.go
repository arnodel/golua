//go:build !noquotas
// +build !noquotas

package runtime

import (
	"io"
	"testing"
)

// TestGoContReleaseOnSuccess verifies that a GoCont's args, etc, and struct
// are all released back to their pools and memory accounting is balanced after
// a successful call.
func TestGoContReleaseOnSuccess(t *testing.T) {
	r := New(io.Discard)
	mt := r.MainThread()

	// Register a Go function with 2 named args and variadic.
	env := r.GlobalEnv()
	SolemnlyDeclareCompliance(
		ComplyCpuSafe|ComplyMemSafe,
		r.SetEnvGoFunc(env, "testfn", func(t *Thread, c *GoCont) (Cont, error) {
			return c.PushingNext1(t.Runtime, IntValue(42)), nil
		}, 2, true),
	)

	ctx, err := mt.CallContext(RuntimeContextDef{
		HardLimits: RuntimeResources{Cpu: 10000, Memory: 100000},
	}, func() error {
		term := NewTerminationWith(nil, 1, false)
		return Call(mt, env.Get(StringValue("testfn")), []Value{IntValue(1), IntValue(2), IntValue(3)}, term)
	})
	if err != nil {
		t.Fatal(err)
	}

	mem := ctx.UsedResources().Memory
	if mem != 0 {
		t.Errorf("memory accounting not balanced after successful GoCont call: used=%d", mem)
	}
}

// TestLuaContReleaseOnSuccess verifies that a LuaCont's registers, cells, and
// struct are released after a successful Lua function call.
func TestLuaContReleaseOnSuccess(t *testing.T) {
	r := New(io.Discard)
	mt := r.MainThread()

	// Compile outside the context so compilation artifacts don't count.
	chunk, err := r.CompileAndLoadLuaChunk("test", []byte(`
		local function f(a, b, c) return a + b + c end
		return f(1, 2, 3)
	`), TableValue(r.GlobalEnv()))
	if err != nil {
		t.Fatal(err)
	}

	ctx, err := mt.CallContext(RuntimeContextDef{
		HardLimits: RuntimeResources{Cpu: 10000, Memory: 100000},
	}, func() error {
		term := NewTerminationWith(nil, 1, false)
		return Call(mt, FunctionValue(chunk), nil, term)
	})
	if err != nil {
		t.Fatal(err)
	}

	mem := ctx.UsedResources().Memory
	if mem != 0 {
		t.Errorf("memory accounting not balanced after successful Lua call: used=%d", mem)
	}
}

// TestLuaContReleaseOnError verifies that a LuaCont's resources are released
// when a Lua function errors (going through the RunContinuation error path).
// We run twice and verify memory doesn't grow, allowing a small fixed residual
// from error handling itself (e.g. the error value).
func TestLuaContReleaseOnError(t *testing.T) {
	r := New(io.Discard)
	mt := r.MainThread()

	// Compile outside the context so compilation artifacts don't count.
	chunk, err := r.CompileAndLoadLuaChunk("test", []byte(`
		local function f(a, b, c) error("intentional") end
		return f(1, 2, 3)
	`), TableValue(r.GlobalEnv()))
	if err != nil {
		t.Fatal(err)
	}

	// Run twice: if continuations leak, memory grows each iteration.
	var memFirst, memSecond uint64
	for i := 0; i < 2; i++ {
		ctx, err := mt.CallContext(RuntimeContextDef{
			HardLimits: RuntimeResources{Cpu: 10000, Memory: 100000},
		}, func() error {
			term := NewTerminationWith(nil, 1, false)
			return Call(mt, FunctionValue(chunk), nil, term)
		})
		if err == nil {
			t.Fatal("expected error from Lua error()")
		}
		mem := ctx.UsedResources().Memory
		if i == 0 {
			memFirst = mem
		} else {
			memSecond = mem
		}
	}

	if memSecond > memFirst {
		t.Errorf("memory grows across erroring Lua calls: first=%d, second=%d", memFirst, memSecond)
	}
}
