package ircomp

import (
	"fmt"
	"math"

	"github.com/arnodel/golua/code"
	"github.com/arnodel/golua/ir"
)

type ConstantCompiler struct {
	builder       *code.Builder
	constants     []ir.Constant
	constantMap   map[uint]int
	compiledCount int
	queue         []uint
	offset        int
}

var _ ir.ConstantProcessor = (*ConstantCompiler)(nil)

func NewConstantCompiler(constants []ir.Constant, cc *code.Builder) *ConstantCompiler {
	kc := &ConstantCompiler{
		builder:     cc,
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

// ProcessCode compiles a Code.  Register lifetimes are the code generator's
// business: it computes them first, from liveness, whatever markers the code
// may carry.
func (kc *ConstantCompiler) ProcessCode(c ir.Code) {
	c = computeLifetimes(c)
	start := kc.builder.Offset()
	regAllocator := &regAllocator{
		registers:   c.Registers,
		allocations: make([]regAllocation, len(c.Registers)),
	}
	// First allocate all the registers that will take upvalues.
	for _, r := range c.UpvalueDests {
		regAllocator.takeRegister(r)
	}
	// Then the preloaded registers, in r1 onwards: r0 is left for the first
	// register allocated, which receives the caller.  They are never
	// released.
	var template []code.Constant
	regAllocator.regs = []int{0}
	for r, rData := range c.Registers {
		if rData.Preloaded {
			template = append(template, templateConstant(kc.constants[rData.Kidx]))
			regAllocator.allocations[r] = regAllocation{done: true, r: code.ValueReg(uint8(len(regAllocator.regs)))}
			regAllocator.regs = append(regAllocator.regs, 1)
		}
	}
	if len(regAllocator.regs) > math.MaxUint8 {
		panic(notEnoughRegisters())
	}
	ic := instrCompiler{
		ConstantCompiler: kc,
		regAllocator:     regAllocator,
	}
	for i, instr := range c.Instructions {
		ic.line = c.Lines[i]
		instr.ProcessInstr(ic)
	}
	end := kc.builder.Offset()
	kc.addCompiled(code.Code{
		Name:         c.Name,
		StartOffset:  start,
		EndOffset:    end,
		UpvalueCount: int16(len(c.UpvalueDests)),
		CellCount:    int16(len(regAllocator.cells)),
		UpNames:      c.UpNames,
		RegCount:     int16(len(regAllocator.regs)),
		Template:     template,
	})
}

// templateConstant returns the compiled form of a constant held in a
// preloaded register.
func templateConstant(k ir.Constant) code.Constant {
	switch kk := k.(type) {
	case ir.Int:
		return code.Int(kk)
	case ir.Float:
		return code.Float(kk)
	case ir.String:
		return code.String(kk)
	case ir.Bool:
		return code.Bool(kk)
	case ir.NilType:
		return code.NilType{}
	}
	panic(fmt.Sprintf("cannot preload %T constant", k))
}

func (kc *ConstantCompiler) compileConstant(ki uint) {
	kc.constants[ki].ProcessConstant(kc)
}

func (kc *ConstantCompiler) addCompiled(ck code.Constant) {
	kc.builder.AddConstant(ck)
	kc.compiledCount++
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

func (kc *ConstantCompiler) CompileQueue() (unit *code.Unit, err error) {
	defer func() {
		if r := recover(); r != nil {
			cp, ok := r.(*CompilationPanic)
			if !ok {
				panic(r)
			}
			// Try to recover where we were, this is not always accurate but a
			// refactor would be required to get accurate info
			//
			// TODO: make more accurate
			unit := kc.builder.GetUnit()
			for i := len(unit.Lines) - 1; i >= 0; i-- {
				if line := unit.Lines[i]; line > 0 {
					cp.line = line
					break
				}
			}
			err = cp
		}
	}()
	for kc.queue != nil {
		queue := kc.queue
		kc.queue = nil
		for _, ki := range queue {
			kc.compileConstant(ki)
			if kc.constantMap[ki] != kc.compiledCount-1 {
				panic("Inconsistent constant indexes :(")
			}
		}
	}
	return kc.builder.GetUnit(), nil
}
