package ir

import (
	"fmt"

	"github.com/arnodel/golua/code"
	"github.com/arnodel/golua/ops"
)

type Name string

type lexicalMap struct {
	reg   map[Name]Register
	label map[Name]Label
}

type LexicalContext []lexicalMap

func (c LexicalContext) GetRegister(name Name) (reg Register, ok bool) {
	for i := len(c) - 1; i >= 0; i-- {
		reg, ok = c[i].reg[name]
		if ok {
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
		c[0].reg[name] = reg
	}
	return
}

func (c LexicalContext) AddToTop(name Name, reg Register) (ok bool) {
	ok = len(c) > 0
	if ok {
		c[len(c)-1].reg[name] = reg
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
		reg:   make(map[Name]Register),
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
		for name, reg := range ns.reg {
			fmt.Printf("  %s: %s\n", name, reg)
		}
		// TODO: dump labels
	}
}

type Compiler struct {
	registers    []int
	context      LexicalContext
	parent       *Compiler
	upvalues     []Register
	code         []Instruction
	constantPool *ConstantPool
	labels       map[Label]int
	labelPos     map[int][]Label
}

func NewCompiler() *Compiler {
	return &Compiler{
		context:      LexicalContext{}.PushNew(),
		labels:       make(map[Label]int),
		labelPos:     make(map[int][]Label),
		constantPool: new(ConstantPool),
	}
}

func (c *Compiler) NewChild() *Compiler {
	return &Compiler{
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

func (c *Compiler) GetRegister(name Name) (reg Register, ok bool) {
	reg, ok = c.context.GetRegister(name)
	// fmt.Println("GET", name, reg, ok)
	if ok || c.parent == nil {
		return
	}
	reg, ok = c.parent.GetRegister(name)
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
	for _, reg := range top.reg {
		c.ReleaseRegister(reg)
	}
}

func (c *Compiler) DeclareLocal(name Name, reg Register) {
	// fmt.Printf("Declare %s %s\n", name, reg)

	c.TakeRegister(reg)
	c.context.AddToTop(name, reg)
}

func (c *Compiler) Emit(instr Instruction) {
	// fmt.Printf("Emit %s\n", instr)
	c.code = append(c.code, instr)
}

func (c *Compiler) GetConstant(k Constant) uint {
	return c.constantPool.GetConstant(k)
}

func (c *Compiler) GetCode() *Code {
	return &Code{
		Instructions: c.code,
		Constants:    c.constantPool.Constants(),
		RegCount:     len(c.registers),
		UpvalueCount: len(c.upvalues),
		LabelPos:     c.labelPos,
	}
}

func EmitConstant(c *Compiler, k Constant, reg Register) {
	c.Emit(LoadConst{Dst: reg, Kidx: c.GetConstant(k)})
}

func EmitMove(c *Compiler, dst Register, src Register) {
	if dst != src {
		c.Emit(Transform{Op: ops.OpId, Dst: dst, Src: src})
	}
}

func (c *Compiler) NewConstantCompiler() *ConstantCompiler {
	ki := c.GetConstant(c.GetCode())
	kc := &ConstantCompiler{
		Compiler:    code.NewCompiler(),
		constants:   c.constantPool.Constants(),
		constantMap: make(map[uint]int),
	}
	kc.QueueConstant(ki)
	return kc
}
