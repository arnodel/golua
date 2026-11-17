package runtime_test

import (
	"strings"
	"testing"

	"github.com/arnodel/golua/lib"
	rt "github.com/arnodel/golua/runtime"
)

// A coroutine has no goroutine until it needs one.  hasGoroutine reports
// whether the thread calling it has one.
func TestLightCoroutines(t *testing.T) {
	r := rt.New(nil)
	defer lib.LoadAll(r)()
	env := r.GlobalEnv()

	hasGoroutine := func(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
		return c.PushingNext1(t.Runtime, rt.BoolValue(t.HasGoroutine())), nil
	}
	// Both honest and liar are declared NoYieldAcross, but only liar calls
	// Lua.
	r.SetEnvGoFunc(env, "honest", hasGoroutine, 0, false, rt.NoYieldAcross)
	r.SetEnvGoFunc(env, "unknown", hasGoroutine, 0, false)
	r.SetEnvGoFunc(env, "liar", func(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
		res, err := rt.Call1(t, c.Arg(0))
		if err != nil {
			return nil, err
		}
		return c.PushingNext1(t.Runtime, res), nil
	}, 1, false, rt.NoYieldAcross)

	run := func(t *testing.T, src string) (rt.Value, error) {
		t.Helper()
		clos, err := r.CompileAndLoadLuaChunk("test", []byte(src), rt.TableValue(env))
		if err != nil {
			t.Fatal(err)
		}
		term := rt.NewTerminationWith(nil, 1, false)
		err = rt.Call(r.MainThread(), rt.FunctionValue(clos), nil, term)
		return term.Get(0), err
	}

	// Each chunk returns a string made of the answer to "has a goroutine?"
	// at several points of one coroutine's life.
	tests := []struct {
		name, src, want string
	}{
		{
			name: "yields and calls of declared functions keep the coroutine light",
			src: `local co = coroutine.wrap(function()
				local a = honest()
				coroutine.yield()
				local b = honest()
				return tostring(a) .. tostring(b)
			end)
			co() return co()`,
			want: "falsefalse",
		},
		{
			name: "a Go function not declared gives the coroutine a goroutine",
			src: `return coroutine.wrap(function()
				return tostring(honest()) .. tostring(unknown()) .. tostring(honest())
			end)()`,
			want: "falsetruetrue",
		},
		{
			name: "a pcall gives the coroutine a goroutine",
			src: `return coroutine.wrap(function()
				local a = honest()
				local _, b = pcall(honest)
				return tostring(a) .. tostring(b) .. tostring(honest())
			end)()`,
			want: "falsetruetrue",
		},
		{
			name: "a metamethod gives the coroutine a goroutine",
			src: `local mt = {__index = function() return 1 end}
			return coroutine.wrap(function()
				local a = honest()
				local _ = setmetatable({}, mt).x
				return tostring(a) .. tostring(honest())
			end)()`,
			want: "falsetrue",
		},
		{
			name: "a value to close gives the coroutine a goroutine",
			src: `return coroutine.wrap(function()
				local a = honest()
				local x <close> = setmetatable({}, {__close = function() end})
				return tostring(a) .. tostring(honest())
			end)()`,
			want: "falsetrue",
		},
		{
			name: "a nil to close does not",
			src: `return coroutine.wrap(function()
				local x <close> = nil
				return tostring(honest())
			end)()`,
			want: "false",
		},
		{
			name: "a debug hook gives the coroutine a goroutine from the start",
			src: `local co = coroutine.create(function() return honest() end)
			debug.sethook(co, function() end, "l")
			return tostring(select(2, coroutine.resume(co)))`,
			want: "true",
		},
		{
			name: "a debug hook set while it waits for another coroutine does too",
			src: `return coroutine.wrap(function()
				local a = honest()
				local me = coroutine.running()
				coroutine.wrap(function() debug.sethook(me, function() end, "c") end)()
				return tostring(a) .. tostring(honest())
			end)()`,
			want: "falsetrue",
		},
		{
			name: "a coroutine that has a goroutine keeps it",
			src: `local co = coroutine.wrap(function()
				pcall(honest)
				coroutine.yield()
				return honest()
			end)
			co() return tostring(co())`,
			want: "true",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := run(t, tt.src)
			if err != nil {
				t.Fatal(err)
			}
			if s, _ := got.TryString(); s != tt.want {
				t.Errorf("got %v, want %q", got, tt.want)
			}
		})
	}

	t.Run("a function declared NoYieldAcross that calls Lua", func(t *testing.T) {
		// The main thread has a goroutine: no problem there.
		got, err := run(t, `return liar(function() return 1 end)`)
		if err != nil || got != rt.IntValue(1) {
			t.Errorf("main thread: got %v, %v", got, err)
		}
		// A light coroutine cannot run the call.
		_, err = run(t, `return coroutine.wrap(function()
			return liar(function() return 1 end)
		end)()`)
		if err == nil || !strings.Contains(err.Error(), "declared NoYieldAcross") {
			t.Errorf("light coroutine: got error %v", err)
		}
	})
}
