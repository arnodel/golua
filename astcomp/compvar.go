package astcomp

import (
	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/ir"
)

type assignFunc func(ir.Register)

type assignCompiler struct {
	*compiler
	assigns []assignFunc
}

var _ ast.VarProcessor = (*assignCompiler)(nil)

// ProcessIndexExpVar compiles the expression as an L-value.
func (c *assignCompiler) ProcessIndexExpVar(e ast.IndexExp) {
	tReg := c.GetFreeRegister()
	c.compileExpInto(e.Coll, tReg)
	c.TakeRegister(tReg)
	iReg := c.GetFreeRegister()
	c.compileExpInto(e.Idx, iReg)
	c.TakeRegister(iReg)
	c.assigns = append(c.assigns, func(src ir.Register) {
		c.ReleaseRegister(tReg)
		c.ReleaseRegister(iReg)
		c.emitInstr(e, ir.SetIndex{
			Table: tReg,
			Index: iReg,
			Src:   src,
		})
	})
}

// ProcessNameVar compiles the expression as an L-value.
func (c *assignCompiler) ProcessNameVar(n ast.Name) {
	reg, ok := c.GetRegister(ir.Name(n.Val))
	if ok {
		c.assigns = append(c.assigns, func(src ir.Register) {
			c.emitMove(n, reg, src)
		})
	} else {
		c.ProcessIndexExpVar(globalVar(n))
	}
}

// compileAssignments compiles a slice of ast.Var (L-values).
func (c *compiler) compileAssignments(lvals []ast.Var, dsts []ir.Register) {
	ac := assignCompiler{compiler: c, assigns: make([]assignFunc, 0, len(lvals))}
	for _, lval := range lvals {
		lval.ProcessVar(&ac)
	}
	// Compile the assignments
	for i, reg := range dsts {
		c.ReleaseRegister(reg)
		ac.assigns[i](reg)
	}
}
