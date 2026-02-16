package astcomp

import (
	"fmt"

	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/ir"
)

type assignFunc func(ir.Register)

type assignCompiler struct {
	*compiler
	assigns         []assignFunc
	checkNotDefined bool // when true, IndexExp assignments emit CheckNotDefined
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
	checkNotDefined := c.checkNotDefined
	c.assigns = append(c.assigns, func(src ir.Register) {
		c.ReleaseRegister(tReg)
		c.ReleaseRegister(iReg)
		c.emitInstr(e, ir.SetIndex{
			Table:           tReg,
			Index:           iReg,
			Src:             src,
			CheckNotDefined: checkNotDefined,
		})
	})
}

// ProcessNameVar compiles the expression as an L-value.
func (c *assignCompiler) ProcessNameVar(n ast.Name) {
	reg, ok := c.GetRegister(ir.Name(n.Val))
	if ok {
		if c.IsConstantReg(reg) {
			panic(Error{
				Where:   n,
				Message: fmt.Sprintf("attempt to reassign constant variable '%s'", n.Val),
			})
		}
		c.assigns = append(c.assigns, func(src ir.Register) {
			c.emitMove(n, reg, src)
		})
	} else {
		// This is a global variable - check if write is authorized
		declType := c.GetGlobalDeclType(ir.Name(n.Val)).StripLegacy()
		switch declType {
		case ir.NoDeclaredGlobal:
			panic(Error{
				Where:   n,
				Message: fmt.Sprintf("attempt to assign to undeclared global variable '%s'", n.Val),
			})
		case ir.ConstGlobal:
			panic(Error{
				Where:   n,
				Message: fmt.Sprintf("attempt to assign to const global variable '%s'", n.Val),
			})
		}
		c.ProcessIndexExpVar(globalVar(n))
	}
}

// compileAssignments compiles a slice of ast.Var (L-values).
func (c *compiler) compileAssignments(lvals []ast.Var, dsts []ir.Register) {
	c.compileAssignmentsWithCheck(lvals, dsts, false)
}

// compileDefineAssignments is like compileAssignments but emits a
// CheckNotDefined before each IndexExp assignment, implementing Lua 5.5's
// "global 'name' already defined" runtime check.
func (c *compiler) compileDefineAssignments(lvals []ast.Var, dsts []ir.Register) {
	c.compileAssignmentsWithCheck(lvals, dsts, true)
}

func (c *compiler) compileAssignmentsWithCheck(lvals []ast.Var, dsts []ir.Register, checkNotDefined bool) {
	ac := assignCompiler{compiler: c, assigns: make([]assignFunc, 0, len(lvals)), checkNotDefined: checkNotDefined}
	for _, lval := range lvals {
		lval.ProcessVar(&ac)
	}
	// Compile the assignments
	for i, reg := range dsts {
		c.ReleaseRegister(reg)
		ac.assigns[i](reg)
	}
}
