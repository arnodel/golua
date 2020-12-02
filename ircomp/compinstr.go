package ircomp

import (
	"fmt"

	"github.com/arnodel/golua/code"
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
)

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
	opcode := code.Combine(codeOp, codeReg(c.Dst), codeReg(c.Lsrc), codeReg(c.Rsrc))
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
	opcode := code.Transform(codeOp, codeReg(t.Dst), codeReg(t.Src))
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
	var opcode code.Opcode
	var inlined bool
	// Short strings and small integers are inlined.
	switch kk := k.(type) {
	case ir.Int:
		opcode, inlined = code.LoadSmallInt(codeReg(l.Dst), int(kk))
	case ir.String:
		opcode, inlined = code.LoadShortString(codeReg(l.Dst), []byte(kk))
	}
	if !inlined {
		ckidx := ic.QueueConstant(l.Kidx)
		opcode = code.LoadConst(codeReg(l.Dst), code.KIndexFromInt(ckidx))
	}
	ic.Emit(opcode)
}

// ProcessPushInstr compiles a Push instruction.
func (ic instrCompiler) ProcessPushInstr(p ir.Push) {
	var opcode code.Opcode
	if p.Etc {
		opcode = code.PushEtc(codeReg(p.Cont), codeReg(p.Item))
	} else {
		opcode = code.Push(codeReg(p.Cont), codeReg(p.Item))
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
		opcode = code.JumpIfNot(0, codeReg(j.Cond))
	} else {
		opcode = code.JumpIf(0, codeReg(j.Cond))
	}
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
	ckidx := ic.QueueConstant(m.Code)
	opcode := code.LoadClosure(codeReg(m.Dst), code.KIndexFromInt(ckidx))
	ic.Emit(opcode)
	// Now add the upvalues
	for _, upval := range m.Upvalues {
		ic.Emit(code.Upval(codeReg(m.Dst), codeReg(upval)))
	}
}

// ProcessMkContInstr compiles a MkCont instruction.
func (ic instrCompiler) ProcessMkContInstr(m ir.MkCont) {
	var opcode code.Opcode
	if m.Tail {
		opcode = code.TailCont(codeReg(m.Dst), codeReg(m.Closure))
	} else {
		opcode = code.Cont(codeReg(m.Dst), codeReg(m.Closure))
	}
	ic.Emit(opcode)
}

// ProcessClearRegInstr compiles a ClearReg instruction.
func (ic instrCompiler) ProcessClearRegInstr(i ir.ClearReg) {
	opcode := code.Clear(codeReg(i.Dst))
	ic.Emit(opcode)
}

// ProcessMkTableInstr compiles a MkTable instruction.
func (ic instrCompiler) ProcessMkTableInstr(m ir.MkTable) {
	opcode := code.LoadEmptyTable(codeReg(m.Dst))
	ic.Emit(opcode)
}

// ProcessLookupInstr compiles a Lookup instruction.
func (ic instrCompiler) ProcessLookupInstr(s ir.Lookup) {
	opcode := code.LoadLookup(codeReg(s.Dst), codeReg(s.Table), codeReg(s.Index))
	ic.Emit(opcode)
}

// ProcessSetIndexInstr compiles a SetIndex instruction.
func (ic instrCompiler) ProcessSetIndexInstr(s ir.SetIndex) {
	opcode := code.SetIndex(codeReg(s.Src), codeReg(s.Table), codeReg(s.Index))
	ic.Emit(opcode)
}

// ProcessReceiveInstr compiles a Receive instruction.
func (ic instrCompiler) ProcessReceiveInstr(r ir.Receive) {
	for _, reg := range r.Dst {
		ic.Emit(code.Receive(codeReg(reg)))
	}
}

// ProcessReceiveEtcInstr compiles a ReceiveEtc instruction.
func (ic instrCompiler) ProcessReceiveEtcInstr(r ir.ReceiveEtc) {
	for _, reg := range r.Dst {
		ic.Emit(code.Receive(codeReg(reg)))
	}
	ic.Emit(code.ReceiveEtc(codeReg(r.Etc)))
}

// ProcessEtcLookupInstr compiles a EtcLookup instruction.
func (ic instrCompiler) ProcessEtcLookupInstr(l ir.EtcLookup) {
	if l.Idx < 0 || l.Idx >= 256 {
		panic("Etc lookup index out of range")
	}
	ic.Emit(code.LoadEtcLookup(codeReg(l.Dst), codeReg(l.Etc), l.Idx))
}

// ProcessFillTableInstr compiles a FillTable instruction.
func (ic instrCompiler) ProcessFillTableInstr(f ir.FillTable) {
	if f.Idx < 0 || f.Idx >= 256 {
		panic("Fill table index out of range")
	}
	ic.Emit(code.FillTable(codeReg(f.Dst), codeReg(f.Etc), f.Idx))
}

func codeReg(r ir.Register) code.Reg {
	if r >= 0 {
		return code.MkRegister(uint8(r))
	}
	return code.MkUpvalue(uint8(-1 - r))
}
