package astcomp

import (
	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/ir"
)

type Assign func(ir.Register)

type AssignCompiler struct {
	*Compiler
	assigns []Assign
}

var _ ast.VarProcessor = (*AssignCompiler)(nil)

// ProcessIndexExpVar compiles the expression as an L-value.
func (c *AssignCompiler) ProcessIndexExpVar(e ast.IndexExp) {
	tReg := c.GetFreeRegister()
	c.CompileExpInto(e.Coll, tReg)
	c.TakeRegister(tReg)
	iReg := c.GetFreeRegister()
	c.CompileExpInto(e.Idx, iReg)
	c.TakeRegister(iReg)
	c.assigns = append(c.assigns, func(src ir.Register) {
		c.ReleaseRegister(tReg)
		c.ReleaseRegister(iReg)
		c.EmitInstr(e, ir.SetIndex{
			Table: tReg,
			Index: iReg,
			Src:   src,
		})
	})
}

// ProcessNameVar compiles the expression as an L-value.
func (c *AssignCompiler) ProcessNameVar(n ast.Name) {
	reg, ok := c.GetRegister(ir.Name(n.Val))
	if ok {
		c.assigns = append(c.assigns, func(src ir.Register) {
			c.EmitMove(n, reg, src)
		})
	} else {
		c.ProcessIndexExpVar(globalVar(n))
	}
}

// CompileAssignments compiles a slice of ast.Var (L-values).
func (c *Compiler) CompileAssignments(lvals []ast.Var, dsts []ir.Register) {
	ac := AssignCompiler{Compiler: c, assigns: make([]Assign, 0, len(lvals))}
	for _, lval := range lvals {
		lval.ProcessVar(&ac)
	}
	// Compile the assignments
	for i, reg := range dsts {
		c.ReleaseRegister(reg)
		ac.assigns[i](reg)
	}
}
