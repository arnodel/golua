package ircomp

import (
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
		upvalueOf:   map[ir.Register]int{},
		atEntry:     true,
		boxUsed:     boxUses(c),
	}
	// Register 0 holds the caller's continuation and must get slot 0.  The
	// registers that receive an upvalue some opcode uses as a box come next,
	// taken over the whole function: the frame fills them when it is made.
	// A fixed upvalue that is only read needs no register for its box.
	regAllocator.takeRegister(0)
	for i, r := range c.UpvalueDests {
		regAllocator.upvalueOf[r] = i
		if regAllocator.boxUsed[r] {
			regAllocator.takeRegister(r)
		}
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
	// The upvalues some opcode uses as a box go in their registers when the
	// frame is made; a fixed one that is only read is there as its contents.
	for i, r := range c.UpvalueDests {
		if regAllocator.boxUsed[r] {
			regAllocator.entryLoads = append(regAllocator.entryLoads,
				code.EntryLoad{Up: uint8(i), Reg: regAllocator.codeReg(r).Idx(), Box: true})
		}
	}
	kc.addCompiled(code.Code{
		Name:         c.Name,
		StartOffset:  start,
		EndOffset:    end,
		UpvalueCount: int16(len(c.UpvalueDests)),
		UpNames:      c.UpNames,
		RegCount:     int16(len(regAllocator.regs)),
		EntryLoads:   regAllocator.entryLoads,
	})
}

// boxUses returns the registers some instruction of c uses as a box: the
// box of a Load or Store, an upvalue a MkClosure captures.  A Load of a
// fixed upvalue at the function's entry (before anything but the
// parameters' recv) is not one: the frame does that load (see
// instrCompiler.ProcessLoadInstr).
func boxUses(c ir.Code) map[ir.Register]bool {
	upvalue := map[ir.Register]bool{}
	for _, r := range c.UpvalueDests {
		upvalue[r] = true
	}
	used := map[ir.Register]bool{}
	atEntry := true
	for _, instr := range c.Instructions {
		switch x := instr.(type) {
		case ir.TakeRegister, ir.ReleaseRegister, ir.Receive, ir.ReceiveEtc:
			continue
		case ir.Load:
			if atEntry && upvalue[x.Box] && c.Registers[x.Box].IsFixed {
				continue
			}
			used[x.Box] = true
		case ir.Store:
			used[x.Box] = true
		case ir.MkClosure:
			for _, u := range x.Upvalues {
				used[u] = true
			}
		}
		atEntry = false
	}
	return used
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
