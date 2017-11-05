package ast

import (
	"fmt"

	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
)

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

func (c LexicalContext) Pop() (LexicalContext, map[Name]ir.Register) {
	if len(c) == 0 {
		return c, nil
	}
	return c[:len(c)-1], c[len(c)-1]
}

func (c LexicalContext) Top() map[Name]ir.Register {
	if len(c) > 0 {
		return c[len(c)-1]
	}
	return nil
}

func (c LexicalContext) Dump() {
	for i, ns := range c {
		fmt.Printf("NS %d:\n", i)
		for name, reg := range ns {
			fmt.Printf("  %s: %s\n", name, reg)
		}
	}
}

type Compiler struct {
	registers []int
	context   LexicalContext
	parent    *Compiler
	upvalues  []ir.Register
	code      []ir.Instruction
	constants []ir.Constant
	labels    map[ir.Label]int
	labelPos  map[int][]ir.Label
}

func NewCompiler(parent *Compiler) *Compiler {
	return &Compiler{
		context:  LexicalContext{}.PushNew(),
		parent:   parent,
		labels:   make(map[ir.Label]int),
		labelPos: make(map[int][]ir.Label),
	}
}

func (c *Compiler) Dump() {
	fmt.Println("--context")
	c.context.Dump()
	fmt.Println("--constants")
	for i, k := range c.constants {
		fmt.Printf("k%d: %s\n", i, k)
	}
	fmt.Println("--code")
	for i, instr := range c.code {
		for _, lbl := range c.labelPos[i] {
			fmt.Printf("%s:\n", lbl)
		}
		fmt.Println(instr)
	}
}

func (c *Compiler) GetNewLabel() ir.Label {
	lbl := ir.Label(len(c.labels))
	c.labels[lbl] = -1
	return lbl
}

func (c *Compiler) EmitLabel(lbl ir.Label) {
	pos := len(c.code)
	c.labels[lbl] = pos
	c.labelPos[pos] = append(c.labelPos[pos], lbl)
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

func (c *Compiler) GetFreeRegister() ir.Register {
	var reg ir.Register
	for i, n := range c.registers {
		if n == 0 {
			reg = ir.Register(i)
			goto FoundLbl
		}
	}
	c.registers = append(c.registers, 0)
	reg = ir.Register(len(c.registers) - 1)
FoundLbl:
	// fmt.Printf("Get Free Reg %s\n", reg)
	return reg
}

func (c *Compiler) TakeRegister(reg ir.Register) {
	if int(reg) >= 0 {
		c.registers[reg]++
		// fmt.Printf("Take Reg %s %d\n", reg, c.registers[reg])
	}
}

func (c *Compiler) ReleaseRegister(reg ir.Register) {
	if int(reg) < 0 {
		return
	}
	if c.registers[reg] == 0 {
		panic("Register cannot be released")
	}
	c.registers[reg]--
	// fmt.Printf("Release Reg %s %d\n", reg, c.registers[reg])
}

func (c *Compiler) PushContext() {
	c.context = c.context.PushNew()
}

func (c *Compiler) PopContext() {
	context, top := c.context.Pop()
	if top == nil {
		panic("Cannot pop empty context")
	}
	c.context = context
	for _, reg := range top {
		c.ReleaseRegister(reg)
	}
}

func (c *Compiler) DeclareLocal(name Name, reg ir.Register) {
	// fmt.Printf("Declare %s %s\n", name, reg)
	c.TakeRegister(reg)
	c.context.AddToTop(name, reg)
}

func (c *Compiler) Emit(instr ir.Instruction) {
	// fmt.Printf("Emit %s\n", instr)
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

func EmitConstant(c *Compiler, k ir.Constant, reg ir.Register) {
	c.Emit(ir.LoadConst{Dst: reg, Kidx: c.GetConstant(k)})
}

func EmitMove(c *Compiler, dst ir.Register, src ir.Register) {
	if dst != src {
		c.Emit(ir.Transform{Op: ops.OpId, Dst: dst, Src: src})
	}
}
