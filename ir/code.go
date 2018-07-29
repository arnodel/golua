package ir

import "github.com/arnodel/golua/code"

type Constant interface {
	Compile(*ConstantCompiler) code.Constant
}

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

type Float float64

func (f Float) Compile(kc *ConstantCompiler) code.Constant {
	return code.Float(f)
}

type Int int64

func (n Int) Compile(kc *ConstantCompiler) code.Constant {
	return code.Int(n)
}

type Bool bool

func (b Bool) Compile(kc *ConstantCompiler) code.Constant {
	return code.Bool(b)
}

type String string

func (s String) Compile(kc *ConstantCompiler) code.Constant {
	return code.String(s)
}

type NilType struct{}

func (n NilType) Compile(kc *ConstantCompiler) code.Constant {
	return code.NilType{}
}

type Code struct {
	Instructions []Instruction
	Lines        []int
	Constants    []Constant
	RegCount     int16
	UpvalueCount int16
	LabelPos     map[int][]Label
	Name         string
}

func (c *Code) Compile(kc *ConstantCompiler) code.Constant {
	start := kc.Offset()
	for i, instr := range c.Instructions {
		for _, lbl := range c.LabelPos[i] {
			kc.EmitLabel(code.Label(lbl))
		}
		instr.Compile(InstrCompiler{c.Lines[i], kc})
	}
	end := kc.Offset()
	return code.Code{
		Name:         c.Name,
		StartOffset:  start,
		EndOffset:    end,
		UpvalueCount: c.UpvalueCount,
		RegCount:     c.RegCount,
	}
}

type InstrCompiler struct {
	line int
	*ConstantCompiler
}

func (ic InstrCompiler) Emit(opcode code.Opcode) {
	ic.Compiler.Emit(opcode, ic.line)
}

func (ic InstrCompiler) EmitJump(opcode code.Opcode, lbl code.Label) {
	ic.Compiler.EmitJump(opcode, lbl, ic.line)
}

type ConstantCompiler struct {
	*code.Compiler
	constants   []Constant
	constantMap map[uint]int
	compiled    []code.Constant
	queue       []uint
	offset      int
}

func (kc *ConstantCompiler) GetConstant(ki uint) Constant {
	return kc.constants[ki]
}

func (kc *ConstantCompiler) QueueConstant(ki uint) int {
	if cki, ok := kc.constantMap[ki]; ok {
		return cki
	}
	kc.constantMap[ki] = kc.offset
	kc.offset++
	kc.queue = append(kc.queue, ki)
	return kc.offset - 1
}

func (kc *ConstantCompiler) CompileQueue() *code.Unit {
	for kc.queue != nil {
		queue := kc.queue
		kc.queue = nil
		for _, ki := range queue {
			ck := kc.constants[ki].Compile(kc)
			if kc.constantMap[ki] != len(kc.compiled) {
				panic("Inconsistent constant indexes :(")
			}
			kc.compiled = append(kc.compiled, ck)
		}
	}
	return code.NewUnit(kc.Source(), kc.Code(), kc.Lines(), kc.compiled)
}
