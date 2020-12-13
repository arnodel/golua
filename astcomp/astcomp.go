package astcomp

import (
	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/ir"
)

// CompileLuaChunk compiles the given block statement to IR code and returns a
// slice or ir.Contant values and the index to the main code constant.
func CompileLuaChunk(source string, s ast.BlockStat) (uint, []ir.Constant) {
	kp := new(ir.ConstantPool)
	rootIrC := ir.NewCodeBuilder("<global chunk>", kp)
	rootIrC.DeclareLocal("_ENV", rootIrC.GetFreeRegister())
	irC := rootIrC.NewChild("<main chunk>")
	c := &compiler{CodeBuilder: irC}
	c.compileFunctionBody(ast.Function{
		ParList: ast.ParList{HasDots: true},
		Body:    s,
	})
	kidx, _ := irC.Close()
	return kidx, kp.Constants()
}

type compiler struct {
	*ir.CodeBuilder
}

func (c *compiler) NewChild(name string) *compiler {
	return &compiler{
		CodeBuilder: c.CodeBuilder.NewChild(name),
	}
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
