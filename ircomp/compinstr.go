package ircomp

import (
	"fmt"
	"math"

	"github.com/arnodel/golua/code"
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
)

type instrCompiler struct {
	*ConstantCompiler
	*regAllocator
	line int
}

var _ ir.InstrProcessor = instrCompiler{}

func (ic instrCompiler) Emit(opcode code.Opcode) {
	ic.builder.Emit(opcode, ic.line)
}

func (ic instrCompiler) EmitJump(opcode code.Opcode, lbl code.Label) {
	ic.builder.EmitJump(opcode, lbl, ic.line)
}

// ProcessCombineInstr compiles a Combine instruction.
func (ic instrCompiler) ProcessCombineInstr(c ir.Combine) {
	codeOp, ok := codeBinOp[c.Op]
	if !ok {
		panic(fmt.Sprintf("Cannot compile %v: invalid op", c))
	}
	opcode := code.Combine(codeOp, ic.codeReg(c.Dst), ic.codeReg(c.Lsrc), ic.codeReg(c.Rsrc))
	ic.Emit(opcode)
}

var codeBinOp = map[ops.Op]code.BinOp{
	ops.OpLt:       code.OpLt,
	ops.OpLeq:      code.OpLeq,
	ops.OpEq:       code.OpEq,
	ops.OpBitOr:    code.OpBitOr,
	ops.OpBitXor:   code.OpBitXor,
	ops.OpBitAnd:   code.OpBitAnd,
	ops.OpShiftL:   code.OpShiftL,
	ops.OpShiftR:   code.OpShiftR,
	ops.OpConcat:   code.OpConcat,
	ops.OpAdd:      code.OpAdd,
	ops.OpSub:      code.OpSub,
	ops.OpMul:      code.OpMul,
	ops.OpDiv:      code.OpDiv,
	ops.OpFloorDiv: code.OpFloorDiv,
	ops.OpMod:      code.OpMod,
	ops.OpPow:      code.OpPow,
}

// ProcessTransformInstr compiles a Transform instruction.
func (ic instrCompiler) ProcessTransformInstr(t ir.Transform) {
	codeOp, ok := codeUnOp[t.Op]
	if !ok {
		panic(fmt.Sprintf("Cannot compile %v: invalid op", t))
	}
	opcode := code.Transform(codeOp, ic.codeReg(t.Dst), ic.codeReg(t.Src))
	ic.Emit(opcode)

}

var codeUnOp = map[ops.Op]code.UnOp{
	ops.OpNeg:      code.OpNeg,
	ops.OpNot:      code.OpNot,
	ops.OpLen:      code.OpLen,
	ops.OpBitNot:   code.OpBitNot,
	ops.OpId:       code.OpId,
	ops.OpToNumber: code.OpToNumber,
}

// ProcessLoadConstInstr compiles a LoadConst instruction.
func (ic instrCompiler) ProcessLoadConstInstr(l ir.LoadConst) {
	k := ic.GetConstant(l.Kidx)
	dst := ic.codeReg(l.Dst)
	var opcode code.Opcode
	var inlined bool
	// Short strings and small integers are inlined.
	switch kk := k.(type) {
	case ir.Int:
		opcode, inlined = code.LoadSmallInt(dst, int(kk))
	case ir.String:
		opcode, inlined = code.LoadShortString(dst, []byte(kk))
	case ir.Bool:
		opcode, inlined = code.LoadBool(dst, bool(kk)), true
	case ir.NilType:
		opcode, inlined = code.LoadNil(dst), true
	}
	if !inlined {
		ckidx := ic.QueueConstant(l.Kidx)
		opcode = code.LoadConst(dst, code.KIndexFromInt(ckidx))
	}
	ic.Emit(opcode)
}

// ProcessPushInstr compiles a Push instruction.
func (ic instrCompiler) ProcessPushInstr(p ir.Push) {
	var opcode code.Opcode
	if p.Etc {
		opcode = code.PushEtc(ic.codeReg(p.Cont), ic.codeReg(p.Item))
	} else {
		opcode = code.Push(ic.codeReg(p.Cont), ic.codeReg(p.Item))
	}
	ic.Emit(opcode)
}

// ProcessJumpInstr compiles a Jump instruction.
func (ic instrCompiler) ProcessJumpInstr(j ir.Jump) {
	// The offset will be computed later on - we don't know it at this stage.
	opcode := code.Jump(0)
	ic.EmitJump(opcode, code.Label(j.Label))
}

// ProcessJumpIfInstr compiles a JumpIf instruction.
func (ic instrCompiler) ProcessJumpIfInstr(j ir.JumpIf) {
	var opcode code.Opcode
	if j.Not {
		opcode = code.JumpIfNot(0, ic.codeReg(j.Cond))
	} else {
		opcode = code.JumpIf(0, ic.codeReg(j.Cond))
	}
	ic.EmitJump(opcode, code.Label(j.Label))
}

// ProcessCallInstr compiles a Call instruction.
func (ic instrCompiler) ProcessCallInstr(c ir.Call) {
	call := code.Call
	if c.Tail {
		call = code.TailCall
	}
	opcode := call(ic.codeReg(c.Cont))
	ic.Emit(opcode)
}

// ProcessMkClosureInstr compiles a MkClosure instruction.
func (ic instrCompiler) ProcessMkClosureInstr(m ir.MkClosure) {
	ckidx := ic.QueueConstant(m.Code)
	opcode := code.LoadClosure(ic.codeReg(m.Dst), code.KIndexFromInt(ckidx))
	ic.Emit(opcode)
	// Now add the upvalues
	for _, upval := range m.Upvalues {
		ic.Emit(code.Upval(ic.codeReg(m.Dst), ic.codeReg(upval)))
	}
}

// ProcessMkContInstr compiles a MkCont instruction.
func (ic instrCompiler) ProcessMkContInstr(m ir.MkCont) {
	var opcode code.Opcode
	if m.Tail {
		opcode = code.TailCont(ic.codeReg(m.Dst), ic.codeReg(m.Closure))
	} else {
		opcode = code.Cont(ic.codeReg(m.Dst), ic.codeReg(m.Closure))
	}
	ic.Emit(opcode)
}

// ProcessClearRegInstr compiles a ClearReg instruction.
func (ic instrCompiler) ProcessClearRegInstr(i ir.ClearReg) {
	opcode := code.Clear(ic.codeReg(i.Dst))
	ic.Emit(opcode)
}

// ProcessMkTableInstr compiles a MkTable instruction.
func (ic instrCompiler) ProcessMkTableInstr(m ir.MkTable) {
	opcode := code.LoadEmptyTable(ic.codeReg(m.Dst))
	ic.Emit(opcode)
}

// ProcessLookupInstr compiles a Lookup instruction.
func (ic instrCompiler) ProcessLookupInstr(s ir.Lookup) {
	opcode := code.LoadLookup(ic.codeReg(s.Dst), ic.codeReg(s.Table), ic.codeReg(s.Index))
	ic.Emit(opcode)
}

// ProcessSetIndexInstr compiles a SetIndex instruction.
func (ic instrCompiler) ProcessSetIndexInstr(s ir.SetIndex) {
	opcode := code.SetIndex(ic.codeReg(s.Src), ic.codeReg(s.Table), ic.codeReg(s.Index))
	ic.Emit(opcode)
}

// ProcessReceiveInstr compiles a Receive instruction.
func (ic instrCompiler) ProcessReceiveInstr(r ir.Receive) {
	for _, reg := range r.Dst {
		ic.Emit(code.Receive(ic.codeReg(reg)))
	}
}

// ProcessReceiveEtcInstr compiles a ReceiveEtc instruction.
func (ic instrCompiler) ProcessReceiveEtcInstr(r ir.ReceiveEtc) {
	for _, reg := range r.Dst {
		ic.Emit(code.Receive(ic.codeReg(reg)))
	}
	ic.Emit(code.ReceiveEtc(ic.codeReg(r.Etc)))
}

// ProcessEtcLookupInstr compiles a EtcLookup instruction.
func (ic instrCompiler) ProcessEtcLookupInstr(l ir.EtcLookup) {
	if l.Idx < 0 || l.Idx >= 256 {
		panic("Etc lookup index out of range")
	}
	ic.Emit(code.LoadEtcLookup(ic.codeReg(l.Dst), ic.codeReg(l.Etc), l.Idx))
}

// ProcessFillTableInstr compiles a FillTable instruction.
func (ic instrCompiler) ProcessFillTableInstr(f ir.FillTable) {
	if f.Idx < 0 || f.Idx >= 256 {
		panic("Fill table index out of range")
	}
	ic.Emit(code.FillTable(ic.codeReg(f.Dst), ic.codeReg(f.Etc), f.Idx))
}

func (ic instrCompiler) ProcessTakeRegisterInstr(t ir.TakeRegister) {
	ic.takeRegister(t.Reg)

}

func (ic instrCompiler) ProcessReleaseRegisterInstr(r ir.ReleaseRegister) {
	ic.releaseRegister(r.Reg)
}

func (ic instrCompiler) ProcessDeclareLabelInstr(l ir.DeclareLabel) {
	ic.builder.EmitLabel(code.Label(l.Label))
}

type regAllocation struct {
	r    code.Reg
	done bool
}
type regAllocator struct {
	registers   []ir.RegData
	allocations []regAllocation
	regs        []int
	cells       []int
}

func (a *regAllocator) takeRegister(r ir.Register) int {
	cr := a.codeReg(r)
	var ref *int
	switch cr.RegType() {
	case code.ValueRegType:
		ref = &a.regs[cr.Idx()]
	case code.CellRegType:
		ref = &a.cells[cr.Idx()]
	}
	*ref++
	return *ref
}

func (a *regAllocator) releaseRegister(r ir.Register) int {
	cr := a.codeReg(r)
	var ref *int
	switch cr.RegType() {
	case code.ValueRegType:
		ref = &a.regs[cr.Idx()]
	case code.CellRegType:
		ref = &a.cells[cr.Idx()]
	}
	*ref--
	return *ref
}

func (a *regAllocator) codeReg(r ir.Register) code.Reg {
	rData := a.registers[r]
	alloc := a.allocations[r]
	if alloc.done {
		return alloc.r
	}
	var cr code.Reg
	var i uint8
	if rData.IsCell {
		a.cells, i = allocReg(a.cells)
		cr = code.CellReg(i)
	} else {
		a.regs, i = allocReg(a.regs)
		cr = code.ValueReg(i)
	}
	a.allocations[r] = regAllocation{
		done: true,
		r:    cr,
	}
	return cr
}

func allocReg(regs []int) ([]int, uint8) {
	for i, c := range regs {
		if c == 0 {
			return regs, uint8(i)
		}
	}
	if len(regs) == math.MaxUint8 {
		panic(newPanic("not enough registers"))
	}
	i := len(regs)
	return append(regs, 0), uint8(i)
}

type CompilationPanic struct {
	msg  string
	line int32
}

func (p *CompilationPanic) Error() string {
	if p.line > 0 {
		return fmt.Sprintf("%s (around line %d)", p.msg, p.line)
	}
	return p.msg
}

func newPanic(msg string) *CompilationPanic {
	return &CompilationPanic{msg: msg}
}
