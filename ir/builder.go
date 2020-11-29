package ir

import (
	"fmt"

	"github.com/arnodel/golua/ops"
)

type Name string

const regHasUpvalue uint = 1

type CodeBuilder struct {
	chunkName    string
	registers    []int
	context      LexicalContext
	parent       *CodeBuilder
	upvalues     []Register
	upnames      []string
	code         []Instruction
	lines        []int
	constantPool *ConstantPool
	labels       map[Label]int
	labelPos     map[int][]Label
}

func NewCodeBuilder(chunkName string, constantPool *ConstantPool) *CodeBuilder {
	return &CodeBuilder{
		chunkName:    chunkName,
		context:      LexicalContext{}.PushNew(),
		labels:       make(map[Label]int),
		labelPos:     make(map[int][]Label),
		constantPool: constantPool,
	}
}

func (c *CodeBuilder) NewChild(chunkName string) *CodeBuilder {
	return &CodeBuilder{
		chunkName:    chunkName,
		context:      LexicalContext{}.PushNew(),
		labels:       make(map[Label]int),
		labelPos:     make(map[int][]Label),
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
	for i, instr := range c.code {
		for _, lbl := range c.labelPos[i] {
			fmt.Printf("%s:\n", lbl)
		}
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
	c.labels[lbl] = -1
	return lbl
}

func (c *CodeBuilder) EmitLabel(lbl Label) {
	pos := len(c.code)
	c.labels[lbl] = pos
	c.labelPos[pos] = append(c.labelPos[pos], lbl)
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
		c.upvalues = append(c.upvalues, reg)
		c.upnames = append(c.upnames, string(name))
		reg = Register(-len(c.upvalues))
		c.context.AddToRoot(name, reg)
	}
	return
}

func (c *CodeBuilder) GetFreeRegister() Register {
	var reg Register
	for i, n := range c.registers {
		if n == 0 {
			reg = Register(i)
			goto FoundLbl
		}
	}
	c.registers = append(c.registers, 0)
	reg = Register(len(c.registers) - 1)
FoundLbl:
	// fmt.Printf("Get Free Reg %s\n", reg)
	return reg
}

func (c *CodeBuilder) TakeRegister(reg Register) {
	if int(reg) >= 0 {
		c.registers[reg]++
		// fmt.Printf("Take Reg %s %d\n", reg, c.registers[reg])
	}
}

func (c *CodeBuilder) ReleaseRegister(reg Register) {
	if int(reg) < 0 {
		return
	}
	if c.registers[reg] == 0 {
		panic("Register cannot be released")
	}
	c.registers[reg]--
	// fmt.Printf("Release Reg %s %d\n", reg, c.registers[reg])
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
		RegCount:     int16(len(c.registers)),
		UpvalueCount: int16(len(c.upvalues)),
		UpNames:      c.upnames,
		LabelPos:     c.labelPos,
		Name:         c.chunkName,
	}
}

func EmitConstant(c *CodeBuilder, k Constant, reg Register, line int) {
	c.Emit(LoadConst{Dst: reg, Kidx: c.getConstant(k)}, line)
}

func EmitMoveNoLine(c *CodeBuilder, dst Register, src Register) {
	if dst != src {
		c.EmitNoLine(Transform{Op: ops.OpId, Dst: dst, Src: src})
	}
}

func EmitMove(c *CodeBuilder, dst Register, src Register, line int) {
	if dst != src {
		c.Emit(Transform{Op: ops.OpId, Dst: dst, Src: src}, line)
	}
}
