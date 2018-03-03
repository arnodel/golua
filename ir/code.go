package ir

import "github.com/arnodel/golua/code"

type Constant interface{}

type ConstantPool struct {
	constants []Constant
}

func (c *ConstantPool) GetConstant(k Constant) uint {
	for i, kk := range c.constants {
		if k == kk {
			return uint(i)
		}
	}
	c.constants = append(c.constants, k)
	return uint(len(c.constants) - 1)
}

func (c *ConstantPool) Constants() []Constant {
	return c.constants
}

type Code struct {
	Instructions []Instruction
	Constants    []Constant
	RegCount     int
	UpvalueCount int
	LabelPos     map[int][]Label
}

type Float float64

type Int int64

type Bool bool

type String string

type NilType struct{}

func (c *Code) Compile(cc *code.Compiler) error {
	for i, instr := range c.Instructions {
		for _, lbl := range c.LabelPos[i] {
			cc.EmitLabel(code.Label(lbl))
		}
		instr.Compile(cc)
	}
	return nil
}
