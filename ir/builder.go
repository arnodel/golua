package ir

import (
	"fmt"

	"github.com/arnodel/golua/ops"
)

type Name string

type RegData struct {
	IsCell   bool
	refCount int
}

const regHasUpvalue uint = 1

type CodeBuilder struct {
	chunkName    string
	registers    []RegData
	context      LexicalContext
	parent       *CodeBuilder
	upvalues     []Register
	upvalueDests []Register
	upnames      []string
	code         []Instruction
	lines        []int
	labels       []bool
	constantPool *ConstantPool
}

func NewCodeBuilder(chunkName string, constantPool *ConstantPool) *CodeBuilder {
	return &CodeBuilder{
		chunkName:    chunkName,
		context:      LexicalContext{}.PushNew(),
		constantPool: constantPool,
	}
}

func (c *CodeBuilder) NewChild(chunkName string) *CodeBuilder {
	return &CodeBuilder{
		chunkName:    chunkName,
		context:      LexicalContext{}.PushNew(),
		constantPool: c.constantPool,
		parent:       c,
	}
}

func (c *CodeBuilder) Dump() {
	fmt.Println("--context")
	c.context.Dump()
	fmt.Println("--constants")
	for i, k := range c.constantPool.Constants() {
		fmt.Printf("k%d: %s\n", i, k)
	}
	fmt.Println("--code")
	for _, instr := range c.code {
		fmt.Println(instr)
	}
}

func (c *CodeBuilder) DeclareGotoLabel(name Name) Label {
	lbl := c.GetNewLabel()
	c.context.AddLabel(name, lbl)
	return lbl
}

func (c *CodeBuilder) getGotoLabel(name Name) (Label, bool) {
	return c.context.GetLabel(name)
}

func (c *CodeBuilder) EmitGotoLabel(name Name) {
	label, ok := c.getGotoLabel(name)
	if !ok {
		panic("Cannot emit undeclared label")
	}
	c.EmitLabel(label)
}

func (c *CodeBuilder) GetNewLabel() Label {
	lbl := Label(len(c.labels))
	c.labels = append(c.labels, false)
	return lbl
}

func (c *CodeBuilder) EmitLabel(lbl Label) {
	if c.labels[lbl] {
		panic(fmt.Sprintf("Label %s emitted twice", lbl))
	}
	c.labels[lbl] = true
	c.EmitNoLine(DeclareLabel{Label: lbl})
}

func (c *CodeBuilder) GetRegister(name Name) (Register, bool) {
	return c.getRegister(name, 0)
}

func (c *CodeBuilder) getRegister(name Name, tags uint) (reg Register, ok bool) {
	reg, ok = c.context.GetRegister(name, tags)
	if ok || c.parent == nil {
		return
	}
	reg, ok = c.parent.getRegister(name, regHasUpvalue)
	if ok {
		c.parent.registers[reg].IsCell = true
		c.upvalues = append(c.upvalues, reg)
		c.upnames = append(c.upnames, string(name))
		reg = c.GetFreeRegister()
		c.upvalueDests = append(c.upvalueDests, reg)
		c.registers[reg].IsCell = true
		c.context.AddToRoot(name, reg)
	}
	return
}

func (c *CodeBuilder) GetFreeRegister() Register {
	reg := Register(len(c.registers))
	c.registers = append(c.registers, RegData{})
	return reg
}

func (c *CodeBuilder) TakeRegister(reg Register) {
	c.registers[reg].refCount++
	c.EmitNoLine(TakeRegister{Reg: reg})
}

func (c *CodeBuilder) ReleaseRegister(reg Register) {
	if c.registers[reg].refCount == 0 {
		panic("cannot release register")
	}
	c.registers[reg].refCount--
	c.EmitNoLine(ReleaseRegister{Reg: reg})
}

func (c *CodeBuilder) PushContext() {
	// fmt.Println("PUSH")
	c.context = c.context.PushNew()
}

func (c *CodeBuilder) PopContext() {
	// fmt.Println("POP")
	context, top := c.context.Pop()
	if top.reg == nil {
		panic("Cannot pop empty context")
	}
	c.context = context
	c.emitClearReg(top)
	for _, tr := range top.reg {
		c.ReleaseRegister(tr.reg)
	}
}

func (c *CodeBuilder) emitClearReg(m lexicalMap) {
	for _, tr := range m.reg {
		if tr.tags&regHasUpvalue != 0 && tr.reg >= 0 {
			c.EmitNoLine(ClearReg{Dst: tr.reg})
		}
	}
}

func (c *CodeBuilder) EmitJump(lblName Name, line int) {
	lc := c.context
	var top lexicalMap
	for len(lc) > 0 {
		lc, top = lc.Pop()
		if lbl, ok := top.label[lblName]; ok {
			c.Emit(Jump{Label: lbl}, line)
			return
		}
		c.emitClearReg(top)
	}
	panic("Undefined label for jump")
}

func (c *CodeBuilder) DeclareLocal(name Name, reg Register) {
	// fmt.Printf("Declare %s %s\n", name, reg)

	c.TakeRegister(reg)
	c.context.AddToTop(name, reg)
}

func (c *CodeBuilder) EmitNoLine(instr Instruction) {
	// fmt.Printf("Emit %s\n", instr)
	c.Emit(instr, 0)
}

func (c *CodeBuilder) Emit(instr Instruction, line int) {
	c.code = append(c.code, instr)
	c.lines = append(c.lines, line)
}

func (c *CodeBuilder) Close() (uint, []Register) {
	return c.getConstant(c.getCode()), c.upvalues
}

func (c *CodeBuilder) getConstant(k Constant) uint {
	return c.constantPool.GetConstant(k)
}

func (c *CodeBuilder) getCode() *Code {
	return &Code{
		Instructions: c.code,
		Lines:        c.lines,
		Constants:    c.constantPool.Constants(),
		Registers:    c.registers,
		UpvalueDests: c.upvalueDests,
		UpNames:      c.upnames,
		Name:         c.chunkName,
	}
}

func EmitConstant(c *CodeBuilder, k Constant, reg Register, line int) {
	c.Emit(LoadConst{Dst: reg, Kidx: c.getConstant(k)}, line)
}

func EmitMoveNoLine(c *CodeBuilder, dst Register, src Register) {
	EmitMove(c, dst, src, 0)
}

func EmitMove(c *CodeBuilder, dst Register, src Register, line int) {
	if dst != src {
		c.Emit(Transform{Op: ops.OpId, Dst: dst, Src: src}, line)
	}
}
