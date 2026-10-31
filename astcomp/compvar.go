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
	checkNotDefined bool                 // when true, IndexExp assignments emit CheckNotDefined
	assigned        map[ir.Register]bool // local registers this statement assigns to
}

var _ ast.VarProcessor = (*assignCompiler)(nil)

// localOperand returns the register of e if e is a local variable whose
// register can be read directly when the store happens: a non-cell local that
// this statement does not assign to.  Nothing can change such a register
// between now and the store, so there is no need to copy it.  With q, i and j
// locals in r3, r4 and r5, "q[i], q[j] = q[j], q[i]" compiles to
//
//	r7 <- r3[r5]; r8 <- r3[r4]; r3[r4] <- r7; r3[r5] <- r8
//
// where before the table and the keys were copied first:
//
//	r7 <- r3[r5]; r8 <- r3[r4]; r9 <- r3; r10 <- r4; r11 <- r3; r12 <- r5;
//	r9[r10] <- r7; r11[r12] <- r8
//
// A cell could be changed by a function called while compiling the other
// l-values, so it is still copied.  So is a local assigned by this statement,
// which must keep its old value for the store: in "i, a[i] = i+1, 20" the
// store goes to the old a[i], so i is copied (r9 <- r4 below) while a is not:
//
//	r7 <- 1; r7 <- r4 + r7; r8 <- 20; r9 <- r4; r4 <- r7; r6[r9] <- r8
func (c *assignCompiler) localOperand(e ast.ExpNode) (ir.Register, bool) {
	n, ok := e.(ast.Name)
	if !ok {
		return 0, false
	}
	reg, ok := c.GetRegister(ir.Name(n.Val))
	if !ok || c.IsCellReg(reg) || c.assigned[reg] {
		return 0, false
	}
	return reg, true
}

// operandReg returns a register holding the value of e for a store, and
// whether that register was taken and must be released after the store.
func (c *assignCompiler) operandReg(e ast.ExpNode) (reg ir.Register, taken bool) {
	if reg, ok := c.localOperand(e); ok {
		return reg, false
	}
	reg = c.GetFreeRegister()
	c.compileExpInto(e, reg)
	c.TakeRegister(reg)
	return reg, true
}

// ProcessIndexExpVar compiles the expression as an L-value.
func (c *assignCompiler) ProcessIndexExpVar(e ast.IndexExp) {
	tReg, tTaken := c.operandReg(e.Coll)
	if s, ok := e.Idx.(ast.String); ok && !c.checkNotDefined {
		// A constant key (e.g. a field name) needs no register.
		kidx := c.GetConstantIndex(ir.String(s.Val))
		c.assigns = append(c.assigns, func(src ir.Register) {
			if tTaken {
				c.ReleaseRegister(tReg)
			}
			c.emitInstr(e, ir.SetIndexK{
				Table: tReg,
				Kidx:  kidx,
				Src:   src,
			})
		})
		return
	}
	iReg, iTaken := c.operandReg(e.Idx)
	checkNotDefined := c.checkNotDefined
	c.assigns = append(c.assigns, func(src ir.Register) {
		if tTaken {
			c.ReleaseRegister(tReg)
		}
		if iTaken {
			c.ReleaseRegister(iReg)
		}
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
	// The locals assigned by this statement must be read before they are
	// assigned, see localOperand.
	for _, lval := range lvals {
		if n, ok := lval.(ast.Name); ok {
			if reg, ok := c.GetRegister(ir.Name(n.Val)); ok {
				if ac.assigned == nil {
					ac.assigned = map[ir.Register]bool{}
				}
				ac.assigned[reg] = true
			}
		}
	}
	for _, lval := range lvals {
		lval.ProcessVar(&ac)
	}
	// Compile the assignments
	for i, reg := range dsts {
		c.ReleaseRegister(reg)
		ac.assigns[i](reg)
	}
}
