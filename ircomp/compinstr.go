package ircomp

import (
	"fmt"

	"github.com/arnodel/golua/code"
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
)

type InstrCompiler interface {
	Emit(code.Opcode)
	EmitJump(code.Opcode, code.Label)
	QueueConstant(uint) int
}

type instrCompiler struct {
	line int
	*ConstantCompiler
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
	opcode := code.MkType1(codeOp, codeReg(c.Dst), codeReg(c.Lsrc), codeReg(c.Rsrc))
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
	opcode := code.MkType4a(code.Off, codeOp, codeReg(t.Dst), codeReg(t.Src))
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
	switch kk := k.(type) {
	case ir.Int:
		if 0 <= kk && kk <= math.MaxUint16 {
			opcode := code.MkType3(code.Off, code.OpInt16, codeReg(l.Dst), code.Lit16(kk))
			ic.Emit(opcode)
			return
		}
	}
	ckidx := ic.QueueConstant(l.Kidx)
	if ckidx > 0xffff {
		panic("Only 2^16 constants are supported in one compilation unit")
	}
	opcode := code.MkType3(code.Off, code.OpK, codeReg(l.Dst), code.Lit16(ckidx))
	ic.Emit(opcode)
}

// ProcessPushInstr compiles a Push instruction.
func (ic instrCompiler) ProcessPushInstr(p ir.Push) {
	op := code.OpId
	if p.Etc {
		op = code.OpEtcId
	}
	opcode := code.MkType4a(code.On, op, codeReg(p.Cont), codeReg(p.Item))
	ic.Emit(opcode)
}

// ProcessJumpInstr compiles a Jump instruction.
func (ic instrCompiler) ProcessJumpInstr(j ir.Jump) {
	opcode := code.MkType5(code.Off, code.OpJump, code.Reg(0), code.Lit16(0))
	ic.EmitJump(opcode, code.Label(j.Label))
}

// ProcessJumpIfInstr compiles a JumpIf instruction.
func (ic instrCompiler) ProcessJumpIfInstr(j ir.JumpIf) {
	flag := code.Off
	if !j.Not {
		flag = code.On
	}
	opcode := code.MkType5(flag, code.OpJumpIf, codeReg(j.Cond), code.Lit16(0))
	ic.EmitJump(opcode, code.Label(j.Label))

}

// ProcessCallInstr compiles a Call instruction.
func (ic instrCompiler) ProcessCallInstr(c ir.Call) {
	// TODO: tailcall
	opcode := code.MkType5(code.Off, code.OpCall, codeReg(c.Cont), code.Lit16(0))
	ic.Emit(opcode)
}

// ProcessMkClosureInstr compiles a MkClosure instruction.
func (ic instrCompiler) ProcessMkClosureInstr(m ir.MkClosure) {
	if m.Code > 0xffff {
		panic("Only 2^16 constants supported")
	}
	ckidx := ic.QueueConstant(m.Code)
	opcode := code.MkType3(code.Off, code.OpClosureK, codeReg(m.Dst), code.Lit16(ckidx))
	ic.Emit(opcode)
	// Now add the upvalues
	for _, upval := range m.Upvalues {
		ic.Emit(code.MkType4a(code.Off, code.OpUpvalue, codeReg(m.Dst), codeReg(upval)))
	}
}

// ProcessMkContInstr compiles a MkCont instruction.
func (ic instrCompiler) ProcessMkContInstr(m ir.MkCont) {
	op := code.OpCont
	if m.Tail {
		op = code.OpTailCont
	}
	opcode := code.MkType4a(code.Off, op, codeReg(m.Dst), codeReg(m.Closure))
	ic.Emit(opcode)
}

// ProcessClearRegInstr compiles a ClearReg instruction.
func (ic instrCompiler) ProcessClearRegInstr(i ir.ClearReg) {
	opcode := code.MkType4b(code.Off, code.OpClear, codeReg(i.Dst), code.Lit8(0))
	ic.Emit(opcode)
}

// ProcessMkTableInstr compiles a MkTable instruction.
func (ic instrCompiler) ProcessMkTableInstr(m ir.MkTable) {
	opcode := code.MkType4b(code.Off, code.OpTable, codeReg(m.Dst), code.Lit8(0))
	ic.Emit(opcode)
}

// ProcessLookupInstr compiles a Lookup instruction.
func (ic instrCompiler) ProcessLookupInstr(s ir.Lookup) {
	opcode := code.MkType2(code.Off, codeReg(s.Dst), codeReg(s.Table), codeReg(s.Index))
	ic.Emit(opcode)
}

// ProcessSetIndexInstr compiles a SetIndex instruction.
func (ic instrCompiler) ProcessSetIndexInstr(s ir.SetIndex) {
	opcode := code.MkType2(code.On, codeReg(s.Src), codeReg(s.Table), codeReg(s.Index))
	ic.Emit(opcode)
}

// ProcessReceiveInstr compiles a Receive instruction.
func (ic instrCompiler) ProcessReceiveInstr(r ir.Receive) {
	for _, reg := range r.Dst {
		ic.Emit(code.MkType0(code.Off, codeReg(reg)))
	}
}

// ProcessReceiveEtcInstr compiles a ReceiveEtc instruction.
func (ic instrCompiler) ProcessReceiveEtcInstr(r ir.ReceiveEtc) {
	for _, reg := range r.Dst {
		ic.Emit(code.MkType0(code.Off, codeReg(reg)))
	}
	ic.Emit(code.MkType0(code.On, codeReg(r.Etc)))
}

// ProcessEtcLookupInstr compiles a EtcLookup instruction.
func (ic instrCompiler) ProcessEtcLookupInstr(l ir.EtcLookup) {
	if l.Idx < 0 || l.Idx >= 256 {
		panic("Etc lookup index out of range")
	}
	ic.Emit(code.MkType6(code.Off, codeReg(l.Dst), codeReg(l.Etc), uint8(l.Idx)))
}

// ProcessFillTableInstr compiles a FillTable instruction.
func (ic instrCompiler) ProcessFillTableInstr(f ir.FillTable) {
	if f.Idx < 0 || f.Idx >= 256 {
		panic("Fill table index out of range")
	}
	ic.Emit(code.MkType6(code.On, codeReg(f.Dst), codeReg(f.Etc), uint8(f.Idx)))
}

func codeReg(r ir.Register) code.Reg {
	if r >= 0 {
		return code.MkRegister(uint8(r))
	}
	return code.MkUpvalue(uint8(-1 - r))
}
