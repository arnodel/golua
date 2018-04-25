package ir

import (
	"fmt"
	"strings"

	"github.com/arnodel/golua/code"
	"github.com/arnodel/golua/ops"
)

type Instruction interface {
	fmt.Stringer
	Compile(*ConstantCompiler)
}

type Register int

func (r Register) String() string {
	if r >= 0 {
		return fmt.Sprintf("r%d", r)
	}
	return fmt.Sprintf("u%d", -r)
}

type Label uint

func (l Label) String() string {
	return fmt.Sprintf("L%d", l)
}

type Combine struct {
	Op   ops.Op
	Dst  Register
	Lsrc Register
	Rsrc Register
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

var codeUnOp = map[ops.Op]code.UnOp{
	ops.OpNeg:    code.OpNeg,
	ops.OpNot:    code.OpNot,
	ops.OpLen:    code.OpLen,
	ops.OpBitNot: code.OpBitNot,
	ops.OpId:     code.OpId,
}

func (c Combine) Compile(kc *ConstantCompiler) {
	codeOp, ok := codeBinOp[c.Op]
	if !ok {
		panic(fmt.Sprintf("Cannot compile %v: invalid op", c))
	}
	opcode := code.MkType1(codeOp, codeReg(c.Dst), codeReg(c.Lsrc), codeReg(c.Rsrc))
	kc.Emit(opcode)
}

func (c Combine) String() string {
	return fmt.Sprintf("%s := %s(%s, %s)", c.Dst, c.Op, c.Lsrc, c.Rsrc)
}

type Transform struct {
	Op  ops.Op
	Dst Register
	Src Register
}

func (t Transform) Compile(kc *ConstantCompiler) {
	codeOp, ok := codeUnOp[t.Op]
	if !ok {
		panic(fmt.Sprintf("Cannot compile %v: invalid op", t))
	}
	opcode := code.MkType4a(code.Off, codeOp, codeReg(t.Dst), codeReg(t.Src))
	kc.Emit(opcode)
}

func (t Transform) String() string {
	return fmt.Sprintf("%s := %s(%s)", t.Dst, t.Op, t.Src)
}

type LoadConst struct {
	Dst  Register
	Kidx uint
}

func (l LoadConst) Compile(kc *ConstantCompiler) {
	ckidx := kc.QueueConstant(l.Kidx)
	if ckidx > 0xffff {
		panic("Only 2^16 constants are supported in one compilation unit")
	}
	opcode := code.MkType3(code.Off, code.OpK, codeReg(l.Dst), code.Lit16(ckidx))
	kc.Emit(opcode)
}

func (l LoadConst) String() string {
	return fmt.Sprintf("%s := k%d", l.Dst, l.Kidx)
}

type Push struct {
	Cont Register
	Item Register
	Etc  bool
}

func (p Push) Compile(kc *ConstantCompiler) {
	op := code.OpId
	if p.Etc {
		op = code.OpEtcId
	}
	opcode := code.MkType4a(code.On, op, codeReg(p.Cont), codeReg(p.Item))
	kc.Emit(opcode)
}

func (p Push) String() string {
	return fmt.Sprintf("push %s to %s", p.Item, p.Cont)
}

type PushCC struct {
	Cont Register
}

func (p PushCC) Compile(kc *ConstantCompiler) {
	opcode := code.MkType4b(code.On, code.OpCC, codeReg(p.Cont), code.Lit8(0))
	kc.Emit(opcode)
}

func (p PushCC) String() string {
	return fmt.Sprintf("push cc to %s", p.Cont)
}

type Jump struct {
	Label Label
}

func (j Jump) String() string {
	return fmt.Sprintf("jump %s", j.Label)
}

func (j Jump) Compile(kc *ConstantCompiler) {
	opcode := code.MkType5(code.Off, code.OpJump, code.Reg(0), code.Lit16(0))
	kc.EmitJump(opcode, code.Label(j.Label))
}

type JumpIf struct {
	Cond  Register
	Label Label
	Not   bool
}

func (j JumpIf) Compile(kc *ConstantCompiler) {
	flag := code.Off
	if !j.Not {
		flag = code.On
	}
	opcode := code.MkType5(flag, code.OpJumpIf, codeReg(j.Cond), code.Lit16(0))
	kc.EmitJump(opcode, code.Label(j.Label))
}

func (j JumpIf) String() string {
	return fmt.Sprintf("jump %s if %s is not %t", j.Label, j.Cond, j.Not)
}

type Call struct {
	Cont Register
}

func (c Call) Compile(kc *ConstantCompiler) {
	// TODO: tailcall
	opcode := code.MkType5(code.Off, code.OpCall, codeReg(c.Cont), code.Lit16(0))
	kc.Emit(opcode)
}

func (c Call) String() string {
	return fmt.Sprintf("call %s", c.Cont)
}

type MkClosure struct {
	Dst      Register
	Code     uint
	Upvalues []Register
}

func (m MkClosure) Compile(kc *ConstantCompiler) {
	if m.Code > 0xffff {
		panic("Only 2^16 constants supported")
	}
	ckidx := kc.QueueConstant(m.Code)
	opcode := code.MkType3(code.Off, code.OpClosureK, codeReg(m.Dst), code.Lit16(ckidx))
	kc.Emit(opcode)
	// Now add the upvalues
	for _, upval := range m.Upvalues {
		kc.Emit(code.MkType4a(code.Off, code.OpUpvalue, codeReg(m.Dst), codeReg(upval)))
	}
}

func joinRegisters(regs []Register, sep string) string {
	us := []string{}
	for _, r := range regs {
		us = append(us, r.String())
	}
	return strings.Join(us, sep)
}

func (m MkClosure) String() string {
	return fmt.Sprintf("%s := mkclos(k%d; %s)", m.Dst, m.Code, joinRegisters(m.Upvalues, ", "))
}

type MkCont struct {
	Dst     Register
	Closure Register
}

func (m MkCont) Compile(kc *ConstantCompiler) {
	opcode := code.MkType4a(code.Off, code.OpCont, codeReg(m.Dst), codeReg(m.Closure))
	kc.Emit(opcode)
}

func (m MkCont) String() string {
	return fmt.Sprintf("%s := mkcont(%s)", m.Dst, m.Closure)
}

type MkCell struct {
	Dst Register
	Src Register
}

func (m MkCell) Compile(kc *ConstantCompiler) {
	// TODO: perhaps this instruction is not needed.
}

func (m MkCell) String() string {
	return fmt.Sprintf("%s := mkcell(%s)", m.Dst, m.Src)
}

type MkTable struct {
	Dst Register
}

func (m MkTable) Compile(kc *ConstantCompiler) {
	opcode := code.MkType4b(code.Off, code.OpTable, codeReg(m.Dst), code.Lit8(0))
	kc.Emit(opcode)
}

func (m MkTable) String() string {
	return fmt.Sprintf("%s := mktable()", m.Dst)
}

type Lookup struct {
	Dst   Register
	Table Register
	Index Register
}

func (s Lookup) Compile(kc *ConstantCompiler) {
	opcode := code.MkType2(code.Off, codeReg(s.Dst), codeReg(s.Table), codeReg(s.Index))
	kc.Emit(opcode)
}

func (s Lookup) String() string {
	return fmt.Sprintf("%s := %s[%s]", s.Dst, s.Table, s.Index)
}

type SetIndex struct {
	Table Register
	Index Register
	Src   Register
}

func (s SetIndex) Compile(kc *ConstantCompiler) {
	opcode := code.MkType2(code.On, codeReg(s.Src), codeReg(s.Table), codeReg(s.Index))
	kc.Emit(opcode)
}

func (s SetIndex) String() string {
	return fmt.Sprintf("%s[%s] := %s", s.Table, s.Index, s.Src)
}

// type Receiver interface {
// 	GetRegisters() []Register
// 	HasEtc() bool
// 	GetEtc() Register
// }

type Receive struct {
	Dst []Register
}

// func (r Receive) GetRegisters() []Register { return r.Dst }
// func (r Receive) HasEtc() bool             { return false }
// func (r Receive) GetEtc() Register         { return Register(0) }

func (r Receive) Compile(kc *ConstantCompiler) {
	for _, reg := range r.Dst {
		kc.Emit(code.MkType0(code.Off, codeReg(reg)))
	}
}

func (r Receive) String() string {
	return fmt.Sprintf("recv(%s)", joinRegisters(r.Dst, ", "))
}

type ReceiveEtc struct {
	Dst []Register
	Etc Register
}

// func (r ReceiveEtc) GetRegisters() []Register { return r.Dst }
// func (r ReceiveEtc) HasEtc() bool             { return true }
// func (r ReceiveEtc) GetEtc() Register         { return r.Etc }

func (r ReceiveEtc) String() string {
	return fmt.Sprintf("recv(%s, ...%s)", joinRegisters(r.Dst, ", "), r.Etc)
}

func (r ReceiveEtc) Compile(kc *ConstantCompiler) {
	for _, reg := range r.Dst {
		kc.Emit(code.MkType0(code.Off, codeReg(reg)))
	}
	kc.Emit(code.MkType0(code.On, codeReg(r.Etc)))
}

type JumpIfForLoopDone struct {
	Label Label
	Var   Register
	Limit Register
	Step  Register
}

func (j JumpIfForLoopDone) Compile(kc *ConstantCompiler) {
	opcode := code.MkType5(code.Off, code.OpJumpIfForLoopDone, code.Reg(0), code.Lit16(0))
	kc.EmitJump(opcode, code.Label(j.Label))
}

func (j JumpIfForLoopDone) String() string {
	return fmt.Sprintf("jump %s if for loop done(%s, %s, %s)", j.Label, j.Var, j.Limit, j.Step)
}

func codeReg(r Register) code.Reg {
	if r >= 0 {
		return code.MkRegister(uint8(r))
	}
	return code.MkUpvalue(uint8(-1 - r))
}
