package code

import (
	"fmt"
	"strconv"
)

type Label uint
type Addr int

type Constant interface {
	ShortString() string
}

type Code struct {
	Name                   string
	StartOffset, EndOffset uint
	UpvalueCount           int
	RegCount               int
}

func (c Code) ShortString() string {
	return "some code"
}

type Float float64

func (f Float) ShortString() string {
	return strconv.FormatFloat(float64(f), 'g', -1, 64)
}

type Int int64

func (i Int) ShortString() string {
	return strconv.FormatInt(int64(i), 10)
}

type Bool bool

func (b Bool) ShortString() string {
	return strconv.FormatBool(bool(b))
}

type String string

func (s String) ShortString() string {
	return strconv.Quote(string(s))
}

type NilType struct{}

func (n NilType) ShortString() string {
	return "nil"
}

type Compiler struct {
	source   string
	lines    []int
	code     []Opcode
	jumpTo   map[Label]int
	jumpFrom map[Label][]int
}

func NewCompiler(source string) *Compiler {
	return &Compiler{
		source:   source,
		jumpTo:   make(map[Label]int),
		jumpFrom: make(map[Label][]int),
	}
}

func (c *Compiler) Emit(opcode Opcode, line int) {
	c.code = append(c.code, opcode)
	c.lines = append(c.lines, line)
}

func (c *Compiler) EmitJump(opcode Opcode, lbl Label, line int) {
	jumpToAddr, ok := c.jumpTo[lbl]
	addr := len(c.code)
	if ok {
		opcode |= Opcode(Lit16(jumpToAddr - addr).ToN())
	} else {
		c.jumpFrom[lbl] = append(c.jumpFrom[lbl], addr)
	}
	c.Emit(opcode, line)
}

func (c *Compiler) EmitLabel(lbl Label) {
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

func (c *Compiler) Offset() uint {
	if len(c.jumpFrom) > 0 {
		fmt.Printf("to: %v\n", c.jumpTo)
		fmt.Printf("from: %v\n", c.jumpFrom)
		panic("Illegal offset")
	}
	c.jumpTo = make(map[Label]int)
	return uint(len(c.code))
}

func (c *Compiler) Code() []Opcode {
	return c.code
}

func (c *Compiler) Lines() []int {
	return c.lines
}

func (c *Compiler) Source() string {
	return c.source
}
