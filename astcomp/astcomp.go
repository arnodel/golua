package astcomp

import (
	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/ir"
)

type Compiler struct {
	ir.Builder
}

// Names of various labels and registers used during compilation.
const (
	breakLblName    = ir.Name("<break>")
	ellipsisRegName = ir.Name("...")
	callerRegName   = ir.Name("<caller>")
	loopFRegName    = ir.Name("<f>")
	loopSRegName    = ir.Name("<s>")
	loopVarRegName  = ir.Name("<var>")
)

// TODO: a better interface
func CompileLuaChunk(source string, s ast.BlockStat) *ir.Compiler {
	rootIrC := ir.NewCompiler(source, "<global chunk>")
	rootIrC.DeclareLocal("_ENV", rootIrC.GetFreeRegister())
	irC := rootIrC.NewChild("<main chunk>")
	c := &Compiler{Builder: irC}
	c.compileFunctionBody(ast.Function{
		ParList: ast.ParList{HasDots: true},
		Body:    s,
	})
	return irC
}

func (c *Compiler) NewChild(name string) *Compiler {
	return &Compiler{
		Builder: c.Builder.NewChild(name),
	}
}
