package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/arnodel/golua/lib/packagelib"
	"github.com/arnodel/golua/parsing"

	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/code"
	"github.com/arnodel/golua/lib/base"
	"github.com/arnodel/golua/lib/coroutine"
	"github.com/arnodel/golua/runtime"
	"github.com/arnodel/golua/scanner"
)

func Test1(t *testing.T) {
	testData := []string{
		`
local function ipairs_iterator(t, n)
    if n < #t then
        return n + 1, t[n + 1]
    end
end

local function ipairs(t)
    return ipairs_iterator, t, 0
end

local t = {5, 4, 3}
local s = 0
print(t[1])
--> =5

for i, v in ipairs(t) do
    s = s + v
end
print(s)
--> =6
`,
		// 		`print("hello," .. " world!", 1 + 2 * 3, {}, 2 == 2.0)`,
		// 		`
		// local function max(x, y)
		//   if x > y then
		//     return x
		//   end
		//   return y
		// end
		// print(max(2, 3) == max(3, 2))`,
		// 		`
		// local function sum(n)
		//     local s = 0
		//     for i = 1,n do
		//         s = s + i
		//     end
		//     return s
		// end
		// print(sum(10))`,
		// 		`
		// local function fac(n)
		//   if n == 0 then
		//     return 1
		//   end
		//   return n * fac(n-1)
		// end
		// print(fac(10))`,
		// 		`
		// local function twice(f)
		//   return function(x)
		//     return f(f(x))
		//   end
		// end
		// local function square(x)
		//   return x*x
		// end
		// print(twice(square)(2))`,
		// 		`
		// local ok, res = pcall(type)
		// print(ok, res)`,
		// 		`
		// print(pcall(type))`,
		// 		`
		// local function p(...)
		//   print(">>>", ...)
		// end
		// p(1, 2, 3)`,
		// 		`
		// local function g(x)
		//   error(x .. "ld!")
		// end
		// local function f(x)
		//     g(x .. ", wor")
		// end
		// print(pcall(f, "hello"))`,
		// 		`
		// local function f(x) return x end
		// coroutine.create(f)`,
		// 		`local x, y = 2, 3; local z = x + 2*y`,
		// 		`local x = 0; if x > 0 then x = x - 1  else x = x + 1 end`,
		// 		`local x; while x > 0 do x = x - 1 end x = 10`,
		// 		`local x = 0; repeat x = x + 1 until x == 10`,
		// 		`local function f(x, y)
		//   local z = x + y
		//   return z
		// end`,
		// 		`for i = 1, 10 do f(i, i + i^2 - 3) end`,
		// 		`local f, s; for i, j in f, s do print(i, j) end`,
		// 		`a = {1, "ab'\"c", 4, x = 2, ['def"\n'] = 1.3, z={tt=1, [u]=2}}`,
		// 		`a = a + 1; return a`,
		// 		`local x = 1; return function(i) x = x + i; return x; end`,
		// 		`local i = 0; while 1 do if i > 5 then break end end`,
		// 		`local a = {}`,
		// 		`print([[foo bar]])`,
		// 		`f(x)(y)`,
		// 		`::foo:: local x = 1; goto foo`,
	}
	w := ast.NewIndentWriter(os.Stdout)
	for i, src := range testData {
		t.Run(fmt.Sprintf("t%d", i), func(t *testing.T) {
			s := scanner.New("test", []byte(src))
			tree, err := parsing.ParseChunk(s.Scan)
			if err != nil {
				t.Error(err)
			} else {
				fmt.Println("\n\n---------")
				fmt.Printf("%s\n", src)
				tree.HWrite(w)
				w.Next()
				c := tree.CompileChunk("test")
				fmt.Printf("%+v\n", c)
				c.Dump()
				kc := c.NewConstantCompiler()
				unit := kc.CompileQueue()
				fmt.Println("\n=========")
				dis := code.NewUnitDisassembler(unit)
				dis.Disassemble(os.Stdout)
				r := runtime.New(os.Stdout)
				base.Load(r)
				packagelib.LibLoader.Run(r)
				coroutine.LibLoader.Run(r)
				t := runtime.NewThread(r)
				clos := runtime.LoadLuaUnit(unit, r.GlobalEnv())

				err := runtime.Call(t, clos, nil, runtime.NewTerminationWith(0, false))
				fmt.Println(err)
			}
		})
	}
}
