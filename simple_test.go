package golua

import (
	"fmt"
	"os"
	"testing"

	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/lexer"
	"github.com/arnodel/golua/parser"
)

func Test1(t *testing.T) {
	testData := []string{
		`local x, y = 2, 3; local z = x + 2*y`,
		`local x = 0; if x > 0 then x = x - 1  else x = x + 1 end`,
		`local x; while x > 0 do x = x - 1 end x = 10`,
		`local x = 0; repeat x = x + 1 until x == 10`,
		`local function f(x, y)
  local z = x + y
  return z
end`,
		`for i = 1, 10 do f(i, i + i^2 - 3); end`,
		`local f, s; for i, j in f, s do print(i, j); end`,
		`a = {1, "ab'\"c", 4, x = 2, ['def"\n'] = 1.3}`,
	}
	w := ast.NewIndentWriter(os.Stdout)
	p := parser.NewParser()
	rc := ir.NewCompiler()
	envReg := rc.GetFreeRegister()
	rc.DeclareLocal(ir.Name("_ENV"), envReg)
	for _, src := range testData {
		s := lexer.NewLexer([]byte(src))
		tree, err := p.Parse(s)
		if err != nil {
			t.Error(err)
		} else {
			fmt.Println("\n\n---------")
			fmt.Printf("%s\n", src)
			tree.(ast.Node).HWrite(w)
			w.Next()
			c := rc.NewChild()
			tree.(ast.Stat).CompileStat(c)
			fmt.Printf("%+v\n", c)
			c.Dump()
		}
	}
}
