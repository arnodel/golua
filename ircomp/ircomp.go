package ircomp

import (
	"github.com/arnodel/golua/code"
	"github.com/arnodel/golua/ir"
)

type ConstantCompiler struct {
	*code.Compiler
	constants   []ir.Constant
	constantMap map[uint]int
	compiled    []code.Constant
	queue       []uint
	offset      int
}

var _ ir.ConstantProcessor = (*ConstantCompiler)(nil)

func NewConstantCompiler(constants []ir.Constant, cc *code.Compiler) *ConstantCompiler {
	kc := &ConstantCompiler{
		Compiler:    cc,
		constants:   constants,
		constantMap: make(map[uint]int),
	}
	return kc
}

// ProcessFloat compiles a Float.
func (kc *ConstantCompiler) ProcessFloat(k ir.Float) {
	kc.addCompiled(code.Float(k))
}

// ProcessInt compiles a Int.
func (kc *ConstantCompiler) ProcessInt(k ir.Int) {
	kc.addCompiled(code.Int(k))
}

// ProcessBool compiles a Bool.
func (kc *ConstantCompiler) ProcessBool(k ir.Bool) {
	kc.addCompiled(code.Bool(k))
}

// ProcessString compiles a String.
func (kc *ConstantCompiler) ProcessString(k ir.String) {
	kc.addCompiled(code.String(k))
}

// ProcessNil compiles a Nil.
func (kc *ConstantCompiler) ProcessNil(k ir.NilType) {
	kc.addCompiled(code.NilType{})
}

// ProcessCode compiles a Code.
func (kc *ConstantCompiler) ProcessCode(c ir.Code) {
	start := kc.Offset()
	for i, instr := range c.Instructions {
		for _, lbl := range c.LabelPos[i] {
			kc.EmitLabel(code.Label(lbl))
		}
		instr.ProcessInstr(instrCompiler{c.Lines[i], kc})
	}
	end := kc.Offset()
	kc.addCompiled(code.Code{
		Name:         c.Name,
		StartOffset:  start,
		EndOffset:    end,
		UpvalueCount: c.UpvalueCount,
		UpNames:      c.UpNames,
		RegCount:     c.RegCount,
	})
}

func (kc *ConstantCompiler) compileConstant(ki uint) {
	kc.constants[ki].ProcessConstant(kc)
}

func (kc *ConstantCompiler) addCompiled(ck code.Constant) {
	kc.compiled = append(kc.compiled, ck)

}

func (kc *ConstantCompiler) GetConstant(ki uint) ir.Constant {
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
			kc.compileConstant(ki)
			if kc.constantMap[ki] != len(kc.compiled)-1 {
				panic("Inconsistent constant indexes :(")
			}
		}
	}
	return code.NewUnit(kc.Source(), kc.Code(), kc.Lines(), kc.compiled)
}
