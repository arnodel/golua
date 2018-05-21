package ir

import (
	"fmt"

	"github.com/arnodel/golua/code"
	"github.com/arnodel/golua/ops"
)

type Name string

type taggedReg struct {
	reg  Register
	tags uint
}

const regHasUpvalue uint = 1

type lexicalMap struct {
	reg   map[Name]taggedReg
	label map[Name]Label
}

type LexicalContext []lexicalMap

func (c LexicalContext) GetRegister(name Name, tags uint) (reg Register, ok bool) {
	for i := len(c) - 1; i >= 0; i-- {
		var tr taggedReg
		tr, ok = c[i].reg[name]
		if ok {
			reg = tr.reg
			if tags != 0 {
				tr.tags |= tags
				c[i].reg[name] = tr
			}
			break
		}
	}
	return
}

func (c LexicalContext) GetLabel(name Name) (label Label, ok bool) {
	for i := len(c) - 1; i >= 0; i-- {
		label, ok = c[i].label[name]
		if ok {
			break
		}
	}
	return
}

func (c LexicalContext) AddToRoot(name Name, reg Register) (ok bool) {
	ok = len(c) > 0
	if ok {
		c[0].reg[name] = taggedReg{reg, 0}
	}
	return
}

func (c LexicalContext) AddToTop(name Name, reg Register) (ok bool) {
	ok = len(c) > 0
	if ok {
		c[len(c)-1].reg[name] = taggedReg{reg, 0}
	}
	return
}

func (c LexicalContext) AddLabel(name Name, label Label) (ok bool) {
	ok = len(c) > 0
	if ok {
		c[len(c)-1].label[name] = label
	}
	return
}

func (c LexicalContext) PushNew() LexicalContext {
	return append(c, lexicalMap{
		reg:   make(map[Name]taggedReg),
		label: make(map[Name]Label),
	})
}

func (c LexicalContext) Pop() (LexicalContext, lexicalMap) {
	if len(c) == 0 {
		return c, lexicalMap{}
	}
	return c[:len(c)-1], c[len(c)-1]
}

func (c LexicalContext) Top() lexicalMap {
	if len(c) > 0 {
		return c[len(c)-1]
	}
	return lexicalMap{}
}

func (c LexicalContext) Dump() {
	for i, ns := range c {
		fmt.Printf("NS %d:\n", i)
		for name, tr := range ns.reg {
			fmt.Printf("  %s: %s\n", name, tr.reg)
		}
		// TODO: dump labels
	}
}

type Compiler struct {
	source       string
	registers    []int
	context      LexicalContext
	parent       *Compiler
	upvalues     []Register
	code         []Instruction
	lines        []int
	constantPool *ConstantPool
	labels       map[Label]int
	labelPos     map[int][]Label
}

func NewCompiler(source string) *Compiler {
	return &Compiler{
		source:       source,
		context:      LexicalContext{}.PushNew(),
		labels:       make(map[Label]int),
		labelPos:     make(map[int][]Label),
		constantPool: new(ConstantPool),
	}
}

func (c *Compiler) NewChild() *Compiler {
	return &Compiler{
		source:       c.source,
		context:      LexicalContext{}.PushNew(),
		labels:       make(map[Label]int),
		labelPos:     make(map[int][]Label),
		constantPool: c.constantPool,
		parent:       c,
	}
}

func (c *Compiler) Upvalues() []Register {
	return c.upvalues
}

func (c *Compiler) Dump() {
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

func (c *Compiler) DeclareGotoLabel(name Name) Label {
	lbl := c.GetNewLabel()
	c.context.AddLabel(name, lbl)
	return lbl
}

func (c *Compiler) GetGotoLabel(name Name) (Label, bool) {
	return c.context.GetLabel(name)
}

func (c *Compiler) EmitGotoLabel(name Name) {
	label, ok := c.GetGotoLabel(name)
	if !ok {
		panic("Cannot emit undeclared label")
	}
	c.EmitLabel(label)
}

func (c *Compiler) GetNewLabel() Label {
	lbl := Label(len(c.labels))
	c.labels[lbl] = -1
	return lbl
}

func (c *Compiler) EmitLabel(lbl Label) {
	pos := len(c.code)
	c.labels[lbl] = pos
	c.labelPos[pos] = append(c.labelPos[pos], lbl)
}

func (c *Compiler) GetRegister(name Name) (Register, bool) {
	return c.getRegister(name, 0)
}

func (c *Compiler) getRegister(name Name, tags uint) (reg Register, ok bool) {
	reg, ok = c.context.GetRegister(name, tags)
	if ok || c.parent == nil {
		return
	}
	reg, ok = c.parent.getRegister(name, regHasUpvalue)
	if ok {
		c.upvalues = append(c.upvalues, reg)
		reg = Register(-len(c.upvalues))
		c.context.AddToRoot(name, reg)
	}
	return
}

func (c *Compiler) GetFreeRegister() Register {
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

func (c *Compiler) TakeRegister(reg Register) {
	if int(reg) >= 0 {
		c.registers[reg]++
		// fmt.Printf("Take Reg %s %d\n", reg, c.registers[reg])
	}
}

func (c *Compiler) ReleaseRegister(reg Register) {
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
	// fmt.Println("PUSH")
	c.context = c.context.PushNew()
}

func (c *Compiler) PopContext() {
	// fmt.Println("POP")
	context, top := c.context.Pop()
	if top.reg == nil {
		panic("Cannot pop empty context")
	}
	c.context = context
	for _, tr := range top.reg {
		c.ReleaseRegister(tr.reg)
		if tr.tags&regHasUpvalue != 0 && tr.reg >= 0 {
			c.EmitNoLine(ClearReg{Dst: tr.reg})
		}
	}
}

func (c *Compiler) DeclareLocal(name Name, reg Register) {
	// fmt.Printf("Declare %s %s\n", name, reg)

	c.TakeRegister(reg)
	c.context.AddToTop(name, reg)
}

func (c *Compiler) EmitNoLine(instr Instruction) {
	// fmt.Printf("Emit %s\n", instr)
	c.Emit(instr, 0)
}

func (c *Compiler) Emit(instr Instruction, line int) {
	c.code = append(c.code, instr)
	c.lines = append(c.lines, line)
}

func (c *Compiler) GetConstant(k Constant) uint {
	return c.constantPool.GetConstant(k)
}

func (c *Compiler) GetCode() *Code {
	return &Code{
		Instructions: c.code,
		Lines:        c.lines,
		Constants:    c.constantPool.Constants(),
		RegCount:     len(c.registers),
		UpvalueCount: len(c.upvalues),
		LabelPos:     c.labelPos,
	}
}

func EmitConstant(c *Compiler, k Constant, reg Register, line int) {
	c.Emit(LoadConst{Dst: reg, Kidx: c.GetConstant(k)}, line)
}

func EmitMoveNoLine(c *Compiler, dst Register, src Register) {
	if dst != src {
		c.EmitNoLine(Transform{Op: ops.OpId, Dst: dst, Src: src})
	}
}

func EmitMove(c *Compiler, dst Register, src Register, line int) {
	if dst != src {
		c.Emit(Transform{Op: ops.OpId, Dst: dst, Src: src}, line)
	}
}

func (c *Compiler) NewConstantCompiler() *ConstantCompiler {
	ki := c.GetConstant(c.GetCode())
	kc := &ConstantCompiler{
		Compiler:    code.NewCompiler(c.source),
		constants:   c.constantPool.Constants(),
		constantMap: make(map[uint]int),
	}
	kc.QueueConstant(ki)
	return kc
}
