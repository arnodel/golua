package runtime_test

import (
	"testing"

	"github.com/arnodel/golua/lib"
	rt "github.com/arnodel/golua/runtime"
)

// A coroutine has no goroutine until it needs one.  The Lua function light
// tells whether the coroutine calling it has none.
func TestLightCoroutines(t *testing.T) {
	r := rt.New(nil)
	defer lib.LoadAll(r)()
	env := r.GlobalEnv()

	hasGoroutine := func(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
		return c.PushingNext1(t.Runtime, rt.BoolValue(t.HasGoroutine())), nil
	}
	// light and callback are declared NoYieldAcross, unknown is not.
	// callback calls the function it is given.
	r.SetEnvGoFunc(env, "light", func(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
		return c.PushingNext1(t.Runtime, rt.BoolValue(!t.HasGoroutine())), nil
	}, 0, false, rt.NoYieldAcross)
	r.SetEnvGoFunc(env, "unknown", hasGoroutine, 0, false)
	r.SetEnvGoFunc(env, "callback", func(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
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

	// Each chunk returns what light() said at several points of one
	// coroutine's life, as a string of y and n.
	tests := []struct {
		name, src, want string
	}{
		{
			name: "yields and library functions keep the coroutine light",
			src: `local function yn(b) return b and "y" or "n" end
			local co = coroutine.wrap(function()
				local s = yn(light())
				coroutine.yield()
				local t = {}
				for i, v in ipairs{3, 1, 2} do table.insert(t, math.floor(v)) end
				table.sort(t)
				s = s .. string.format("%s", yn(light())) .. tostring(#t):sub(1, 0)
				for k in pairs(t) do end
				return s .. yn(light())
			end)
			co() return co()`,
			want: "yyy",
		},
		{
			name: "a Go function not declared gives the coroutine a goroutine",
			src: `local function yn(b) return b and "y" or "n" end
			return coroutine.wrap(function()
				local a = light()
				local b = not unknown()
				return yn(a) .. yn(b) .. yn(light())
			end)()`,
			want: "ynn",
		},
		{
			name: "a pcall gives the coroutine a goroutine",
			src: `local function yn(b) return b and "y" or "n" end
			return coroutine.wrap(function()
				local a = light()
				local _, b = pcall(light)
				return yn(a) .. yn(b) .. yn(light())
			end)()`,
			want: "ynn",
		},
		{
			name: "a metamethod called by an opcode gives the coroutine a goroutine",
			src: `local function yn(b) return b and "y" or "n" end
			local mt = {__index = function() return 1 end}
			return coroutine.wrap(function()
				local a = light()
				local _ = setmetatable({}, mt).x
				return yn(a) .. yn(light())
			end)()`,
			want: "yn",
		},
		{
			name: "a value to close gives the coroutine a goroutine",
			src: `local function yn(b) return b and "y" or "n" end
			return coroutine.wrap(function()
				local a = light()
				local x <close> = setmetatable({}, {__close = function() end})
				return yn(a) .. yn(light())
			end)()`,
			want: "yn",
		},
		{
			name: "a nil to close does not",
			src: `local function yn(b) return b and "y" or "n" end
			return coroutine.wrap(function()
				local x <close> = nil
				return yn(light())
			end)()`,
			want: "y",
		},
		{
			name: "a debug hook does not: it cannot yield",
			src: `local function yn(b) return b and "y" or "n" end
			local calls = 0
			local co = coroutine.create(function()
				local function f() end
				f()
				local a = light()
				debug.sethook()
				return yn(a) .. yn(light()) .. yn(calls > 0)
			end)
			debug.sethook(co, function() calls = calls + 1 end, "c")
			return select(2, coroutine.resume(co))`,
			want: "yyy",
		},
		{
			name: "a coroutine that has a goroutine keeps it",
			src: `local function yn(b) return b and "y" or "n" end
			local co = coroutine.wrap(function()
				pcall(light)
				coroutine.yield()
				return yn(light())
			end)
			co() return co()`,
			want: "n",
		},
		{
			name: "Lua code called by a declared function does anything without a goroutine",
			src: `local function yn(b) return b and "y" or "n" end
			return coroutine.wrap(function()
				local closed = false
				local s = callback(function()
					-- None of these can be yielded across here.
					local x <close> = setmetatable({}, {__close = function() closed = true end})
					local _ = setmetatable({}, {__index = function() return 1 end}).x
					local ok, a = pcall(light)
					return yn(ok and a) .. yn(not unknown())
				end)
				return s .. yn(closed) .. yn(light())
			end)()`,
			want: "yyyy",
		},
		{
			name: "and cannot yield",
			src: `local function yn(b) return b and "y" or "n" end
			return coroutine.wrap(function()
				local ok, msg = pcall(callback, coroutine.yield)
				return yn(ok) .. yn(msg:find("attempt to yield across a Go-call boundary", 1, true) ~= nil)
			end)()`,
			want: "ny",
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
}
