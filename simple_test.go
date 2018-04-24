package golua

import (
	"fmt"
	"os"
	"testing"

	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/code"
	"github.com/arnodel/golua/lexer"
	"github.com/arnodel/golua/parser"
	"github.com/arnodel/golua/runtime"
)

func Test1(t *testing.T) {
	testData := []string{
		`print("hello," .. " world!", 1 + 2 * 3)`,
		`
local function max(x, y)
  if x > y then
    return x
  end
  return y
end
print(max(2, 3) == max(3, 2))`,
		`
local function sum(n)
    local s = 0
    for i = 1,n do
        s = s + i
    end
    return s
end
print(sum(10))`,
		`
local function fac(n)
  if n == 0 then
    return 1
  end
  return n * fac(n-1)
end
print(fac(10))`,
		`
local function twice(f)
  return function(x)
    return f(f(x))
  end
end
local function square(x)
  return x*x
end
print(twice(square)(2))`,
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
	p := parser.NewParser()
	for i, src := range testData {
		t.Run(fmt.Sprintf("t%d", i), func(t *testing.T) {
			s := lexer.NewLexer([]byte(src))
			tree, err := p.Parse(s)
			if err != nil {
				t.Error(err)
			} else {
				fmt.Println("\n\n---------")
				fmt.Printf("%s\n", src)
				tree.(ast.Node).HWrite(w)
				w.Next()
				c := tree.(ast.BlockStat).CompileChunk()
				fmt.Printf("%+v\n", c)
				c.Dump()
				kc := c.NewConstantCompiler()
				unit := kc.CompileQueue()
				fmt.Println("\n=========")
				dis := code.NewUnitDisassembler(unit)
				dis.Disassemble(os.Stdout)
				env := runtime.NewTable()
				env.Set(runtime.String("print"), runtime.GoFunction(print))
				t := runtime.NewThread(env)
				clos := runtime.LoadLuaUnit(t, unit)

				err := runtime.Call(t, clos, nil, runtime.NewTerminationWith(0, false))
				fmt.Println(err)
			}
		})
	}
}

func print(t *runtime.Thread, args []runtime.Value, next runtime.Continuation) error {
	for i, v := range args {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Print(v)
	}
	fmt.Print("\n")
	return nil
}
