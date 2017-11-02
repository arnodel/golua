package ast

import "github.com/arnodel/golua/ir"

type LexicalContext []map[Name]ir.Register

func (c LexicalContext) GetRegister(name Name) (reg ir.Register, ok bool) {
	for i := len(c) - 1; i >= 0; i-- {
		reg, ok = c[i][name]
		if ok {
			break
		}
	}
	return
}

func (c LexicalContext) AddToRoot(name Name, reg ir.Register) (ok bool) {
	ok = len(c) > 0
	if ok {
		c[0][name] = reg
	}
	return
}

func (c LexicalContext) AddToTop(name Name, reg ir.Register) (ok bool) {
	ok = len(c) > 0
	if ok {
		c[len(c)-1][name] = reg
	}
	return
}

func (c LexicalContext) PushNew() LexicalContext {
	return append(c, make(map[Name]ir.Register))
}

func (c LexicalContext) Pop() LexicalContext {
	if len(c) == 0 {
		return c
	}
	return c[:len(c)-1]
}

type Compiler struct {
	regCount  int
	context   LexicalContext
	parent    *Compiler
	upvalues  []ir.Register
	code      []ir.Instruction
	constants []ir.Constant
}

func NewCompiler(parent *Compiler) *Compiler {
	return &Compiler{
		context: LexicalContext{}.PushNew(),
		parent:  parent,
	}
}

func (c *Compiler) GetRegister(name Name) (reg ir.Register, ok bool) {
	reg, ok = c.context.GetRegister(name)
	if ok || c.parent == nil {
		return
	}
	reg, ok = c.parent.GetRegister(name)
	if ok {
		c.upvalues = append(c.upvalues, reg)
		reg = ir.Register(-len(c.upvalues))
		c.context.AddToRoot(name, reg)
	}
	return
}

func (c *Compiler) NewRegister() ir.Register {
	reg := ir.Register(c.regCount)
	c.regCount++
	return reg
}

func (c *Compiler) PushContext() {
	c.context = c.context.PushNew()
}

func (c *Compiler) PopContext() {
	c.context = c.context.Pop()
}

func (c *Compiler) DeclareLocal(name Name) ir.Register {
	reg := c.NewRegister()
	c.context.AddToTop(name, reg)
	return reg
}

func (c *Compiler) Emit(instr ir.Instruction) {
	c.code = append(c.code, instr)
}

func (c *Compiler) GetConstant(k ir.Constant) uint {
	for i, kk := range c.constants {
		if k == kk {
			return uint(i)
		}
	}
	c.constants = append(c.constants, k)
	return uint(len(c.constants) - 1)
}

func EmitConstant(c *Compiler, k ir.Constant) ir.Register {
	reg := c.NewRegister()
	c.Emit(ir.LoadConst{Dst: reg, Kidx: c.GetConstant(k)})
	return reg
}
