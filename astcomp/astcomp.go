package astcomp

import (
	"fmt"

	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/ir"
)

// CompileLuaChunk compiles the given block statement to IR code and returns a
// slice or ir.Contant values and the index to the main code constant.
func CompileLuaChunk(source string, s ast.BlockStat) (kidx uint, consts []ir.Constant, err error) {
	defer func() {
		if r := recover(); r != nil {
			compErr, ok := r.(Error)
			if !ok {
				panic(r)
			}
			err = compErr
		}
	}()
	kp := ir.NewConstantPool()
	rootIrC := ir.NewCodeBuilder("<global chunk>", kp)
	rootIrC.DeclareLocal("_ENV", rootIrC.GetFreeRegister())
	irC := rootIrC.NewChild("<main chunk>")
	c := &compiler{CodeBuilder: irC}
	c.compileFunctionBody(ast.Function{
		ParList: ast.ParList{HasDots: true},
		Body:    s,
	})
	kidx, _ = irC.Close()
	return kidx, kp.Constants(), nil
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

// Error that results from a valid AST which does not form a valid program.
type Error struct {
	Where   ast.Locator
	Message string
}

func (e Error) Error() string {
	loc := e.Where.Locate().StartPos()
	return fmt.Sprintf("%d:%d: %s", loc.Line, loc.Column, e.Message)
}

// The compiler uses other packages that may return non nil errors only if there
// is a bug in the compiler.  Such errors are wrapped in must() so that the bugs
// are not silently ignored.
type compilerBug struct {
	err error
}

func (b compilerBug) Error() string {
	return b.err.Error()
}

func must(err error) {
	if err != nil {
		panic(compilerBug{err: err})
	}
}
