package runtime

import (
	"errors"
	"io"
	"testing"
)

// Benchmark helpers

// benchLua compiles a Lua chunk and runs it b.N times, reporting allocations.
func benchLua(b *testing.B, r *Runtime, src string) {
	b.Helper()
	chunk, err := r.CompileAndLoadLuaChunk("bench", []byte(src), TableValue(r.GlobalEnv()))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		term := NewTerminationWith(nil, 1, false)
		if err := Call(r.MainThread(), FunctionValue(chunk), nil, term); err != nil {
			b.Fatal(err)
		}
	}
}

func newBenchRuntime() *Runtime {
	return New(io.Discard)
}

// BenchmarkPoolTailRecursion exercises contpool heavily: each tail call
// releases the current continuation back to the pool and the next call
// reacquires it immediately. Regpool benefits from reusing same-sized
// register arrays.
func BenchmarkPoolTailRecursion(b *testing.B) {
	benchLua(b, newBenchRuntime(), `
		local function sum(n, acc)
			if n <= 0 then return acc end
			return sum(n-1, acc+n)
		end
		return sum(1000000, 0)
	`)
}

// BenchmarkPoolDeepRecursion creates many live continuations simultaneously
// (non-tail recursive fibonacci). Contpool helps less here since
// continuations can't be recycled until unwinding. Regpool still helps
// by reusing register arrays on the way back up.
func BenchmarkPoolDeepRecursion(b *testing.B) {
	benchLua(b, newBenchRuntime(), `
		local function fib(n)
			if n <= 1 then return n end
			return fib(n-1) + fib(n-2)
		end
		return fib(25)
	`)
}

// BenchmarkPoolLoopWithCalls tests steady-state pool reuse: a tight loop
// calling a function repeatedly. Each iteration allocates and releases
// one continuation and one register set.
func BenchmarkPoolLoopWithCalls(b *testing.B) {
	benchLua(b, newBenchRuntime(), `
		local function f(x) return x + 1 end
		local s = 0
		for i = 1, 500000 do s = f(s) end
		return s
	`)
}

// BenchmarkPoolGoFunctionCalls tests GoCont pool and argsPool reuse by
// repeatedly calling a Go-implemented function from Lua.
func BenchmarkPoolGoFunctionCalls(b *testing.B) {
	r := newBenchRuntime()
	// Register a simple Go function as a global
	env := r.GlobalEnv()
	r.SetEnvGoFunc(env, "addone", func(t *Thread, c *GoCont) (Cont, error) {
		x, err := c.IntArg(0)
		if err != nil {
			return nil, err
		}
		return c.PushingNext1(t.Runtime, IntValue(x+1)), nil
	}, 1, false)

	benchLua(b, r, `
		local s = 0
		for i = 1, 500000 do s = addone(s) end
		return s
	`)
}

// BenchmarkPoolManyLocals tests regpool with large register arrays.
// A function with 20 locals is called repeatedly in a loop.
func BenchmarkPoolManyLocals(b *testing.B) {
	benchLua(b, newBenchRuntime(), `
		local function f()
			local a,b,c,d,e,f,g,h,i,j = 1,2,3,4,5,6,7,8,9,10
			local k,l,m,n,o,p,q,r,s,t = 11,12,13,14,15,16,17,18,19,20
			return a+b+c+d+e+f+g+h+i+j+k+l+m+n+o+p+q+r+s+t
		end
		local s = 0
		for i = 1, 200000 do s = s + f() end
		return s
	`)
}

// BenchmarkPoolCoroutines tests pool behaviour across coroutine boundaries.
// Creates and runs many short-lived coroutines.
func BenchmarkPoolCoroutines(b *testing.B) {
	r := newBenchRuntime()
	// Register coroutine.wrap and coroutine.yield as globals
	env := r.GlobalEnv()
	coroutineTable := NewTable()
	r.SetEnvGoFunc(coroutineTable, "wrap", goWrap, 1, false)
	r.SetTable(env, StringValue("coroutine"), TableValue(coroutineTable))

	benchLua(b, r, `
		local wrap = coroutine.wrap
		local s = 0
		for i = 1, 100000 do
			local co = wrap(function(x) return x + 1 end)
			s = s + co(i)
		end
		return s
	`)
}

// goWrap implements coroutine.wrap for the benchmark (minimal version).
func goWrap(t *Thread, c *GoCont) (Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	f, ok := c.Arg(0).TryCallable()
	if !ok {
		return nil, errors.New("argument must be callable")
	}
	coro := NewThread(t.Runtime)
	coro.Start(f)
	wrapped := NewGoFunction(func(t2 *Thread, c2 *GoCont) (Cont, error) {
		res, err := coro.Resume(t2, c2.Etc())
		if err != nil {
			return nil, err
		}
		return c2.PushingNext(t2.Runtime, res...), nil
	}, "wrapped_coroutine", 0, true)
	return c.PushingNext1(t.Runtime, FunctionValue(wrapped)), nil
}
