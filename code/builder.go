package code

import (
	"fmt"
	"strconv"
)

type Label uint
type Addr int

type Constant interface {
	ShortString(d *UnitDisassembler) string
}

type Code struct {
	Name                   string
	StartOffset, EndOffset uint
	UpvalueCount           int16
	RegCount               int16
	UpNames                []string
}

func (c Code) ShortString(d *UnitDisassembler) string {
	lbl := d.GetLabel(int(c.StartOffset))
	name := c.Name
	if name == "" {
		name = "<" + lbl + ">"
	}
	d.SetSpan(name, int(c.StartOffset), int(c.EndOffset-1))
	return fmt.Sprintf("function %s %s=%d - %d", name, lbl, c.StartOffset, c.EndOffset-1)
}

type Float float64

func (f Float) ShortString(d *UnitDisassembler) string {
	return strconv.FormatFloat(float64(f), 'g', -1, 64)
}

type Int int64

func (i Int) ShortString(d *UnitDisassembler) string {
	return strconv.FormatInt(int64(i), 10)
}

type Bool bool

func (b Bool) ShortString(d *UnitDisassembler) string {
	return strconv.FormatBool(bool(b))
}

type String string

func (s String) ShortString(d *UnitDisassembler) string {
	return strconv.Quote(string(s))
}

type NilType struct{}

func (n NilType) ShortString(d *UnitDisassembler) string {
	return "nil"
}

type Builder struct {
	source    string
	lines     []int32
	code      []Opcode
	jumpTo    map[Label]int
	jumpFrom  map[Label][]int
	constants []Constant
}

func NewBuilder(source string) *Builder {
	return &Builder{
		source:   source,
		jumpTo:   make(map[Label]int),
		jumpFrom: make(map[Label][]int),
	}
}

func (c *Builder) Emit(opcode Opcode, line int) {
	c.code = append(c.code, opcode)
	c.lines = append(c.lines, int32(line))
}

func (c *Builder) EmitJump(opcode Opcode, lbl Label, line int) {
	jumpToAddr, ok := c.jumpTo[lbl]
	addr := len(c.code)
	if ok {
		opcode |= Opcode(Lit16(jumpToAddr - addr).ToN())
	} else {
		c.jumpFrom[lbl] = append(c.jumpFrom[lbl], addr)
	}
	c.Emit(opcode, line)
}

func (c *Builder) EmitLabel(lbl Label) {
	if _, ok := c.jumpTo[lbl]; ok {
		panic("Label already emitted")
	}
	addr := len(c.code)
	c.jumpTo[lbl] = addr
	for _, jumpFromAddr := range c.jumpFrom[lbl] {
		c.code[jumpFromAddr] |= Opcode(Lit16(addr - jumpFromAddr).ToN())
	}
	delete(c.jumpFrom, lbl)
}

func (c *Builder) Offset() uint {
	if len(c.jumpFrom) > 0 {
		fmt.Printf("to: %v\n", c.jumpTo)
		fmt.Printf("from: %v\n", c.jumpFrom)
		panic("Illegal offset")
	}
	c.jumpTo = make(map[Label]int)
	return uint(len(c.code))
}

func (c *Builder) AddConstant(k Constant) {
	c.constants = append(c.constants, k)
}

func (c *Builder) GetUnit() *Unit {
	return NewUnit(c.source, c.code, c.lines, c.constants)
}
