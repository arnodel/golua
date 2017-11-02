package golua

import (
	"os"
	"testing"

	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/lexer"
	"github.com/arnodel/golua/parser"
)

func Test1(t *testing.T) {
	testData := []string{
		`if x > 0 then x = x - 1 end`,
		`local function f(x, y)
  local z = x + y
  return z
end`,
		`for i = 1, 10 do f(i, i + i^2 - 3); end`,
		`a = {1, "ab'\"c", 4, x = 2, ['def"\n'] = 1.3}`,
	}
	w := ast.NewIndentWriter(os.Stdout)
	p := parser.NewParser()
	for _, src := range testData {
		s := lexer.NewLexer([]byte(src))
		tree, err := p.Parse(s)
		if err != nil {
			t.Error(err)
		} else {
			tree.(ast.Node).HWrite(w)
			w.Next()
		}
	}
}
