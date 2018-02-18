package ir

import (
	"fmt"
	"strings"

	"github.com/arnodel/golua/code"
	"github.com/arnodel/golua/ops"
)

type Instruction interface {
	fmt.Stringer
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

var codeBinOp = map[ops.Op]code.BinOp{}
var codeUnOp = map[ops.Op]code.UnOp{}

func codeReg(r Register) code.Reg {
	if r >= 0 {
		return code.MkRegister(uint8(r))
	}
	return code.MkUpvalue(uint8(1 - r))
}

func (c Combine) Compile(cc *code.Compiler) {
	opcode := code.MkType1(codeBinOp[c.Op], codeReg(c.Dst), codeReg(c.Lsrc), codeReg(c.Rsrc))
	cc.Emit(opcode)
}

func (c Combine) String() string {
	return fmt.Sprintf("%s := %s(%s, %s)", c.Dst, c.Op, c.Lsrc, c.Rsrc)
}

type Transform struct {
	Op  ops.Op
	Dst Register
	Src Register
}

func (t Transform) Compile(cc *code.Compiler) {
	opcode := code.MkType4a(code.Off, codeUnOp[t.Op], codeReg(t.Dst), codeReg(t.Src))
	cc.Emit(opcode)
}

func (t Transform) String() string {
	return fmt.Sprintf("%s := %s(%s)", t.Dst, t.Op, t.Src)
}

type LoadConst struct {
	Dst  Register
	Kidx uint
}

func (l LoadConst) Compile(cc *code.Compiler) {
	if l.Kidx > 0xffff {
		panic("Only 2^16 constants are supported in one compilation unit")
	}
	opcode := code.MkType3(code.Off, code.OpK, codeReg(l.Dst), code.Lit16(l.Kidx))
	cc.Emit(opcode)
}

func (l LoadConst) String() string {
	return fmt.Sprintf("%s := k%d", l.Dst, l.Kidx)
}

type Push struct {
	Cont Register
	Item Register
}

func (p Push) Compile(cc *code.Compiler) {
	opcode := code.MkType4a(code.On, code.OpId, codeReg(p.Cont), codeReg(p.Item))
	cc.Emit(opcode)
}

func (p Push) String() string {
	return fmt.Sprintf("push %s to %s", p.Item, p.Cont)
}

type PushCC struct {
	Cont Register
}

func (p PushCC) Compile(cc *code.Compiler) {
	opcode := code.MkType4b(code.On, code.OpCC, codeReg(p.Cont), code.Lit8(0))
	cc.Emit(opcode)
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

func (j Jump) Compile(cc *code.Compiler) {
	addr := cc.GetLabelAddr(code.Label(j.Label))
	opcode := code.MkType5(code.Off, code.OpJump, code.Reg(0), code.Lit16(addr))
	cc.Emit(opcode)
}

type JumpIf struct {
	Cond  Register
	Label Label
	Not   bool
}

func (j JumpIf) Compile(cc *code.Compiler) {
	addr := cc.GetLabelAddr(code.Label(j.Label))
	flag := code.Off
	if !j.Not {
		flag = code.On
	}
	opcode := code.MkType5(flag, code.OpJumpIf, codeReg(j.Cond), code.Lit16(addr))
	cc.Emit(opcode)
}

func (j JumpIf) String() string {
	return fmt.Sprintf("jump %s if %s is not %t", j.Label, j.Cond, j.Not)
}

type Call struct {
	Cont Register
}

func (c Call) Compile(cc *code.Compiler) {
	// TODO: tailcall
	opcode := code.MkType5(code.Off, code.OpCall, codeReg(c.Cont), code.Lit16(0))
	cc.Emit(opcode)
}

func (c Call) String() string {
	return fmt.Sprintf("call %s", c.Cont)
}

type MkClosure struct {
	Dst      Register
	Code     uint
	Upvalues []Register
}

func (m MkClosure) Compile(cc *code.Compiler) {
	if m.Code > 0xffff {
		panic("Only 2^16 constants supported")
	}
	opcode := code.MkType3(code.Off, code.OpClosureK, codeReg(m.Dst), code.Lit16(m.Code))
	cc.Emit(opcode)
	// Now add the upvalues
	upvals := m.Upvalues
	for len(upvals) >= 3 {
		opcode := code.MkType0(codeReg(upvals[0]), codeReg(upvals[1]), codeReg(upvals[2]))
		cc.Emit(opcode)
		upvals = upvals[3:]
	}
	switch len(upvals) {
	case 1:
		opcode := code.MkType0(codeReg(upvals[0]), code.Reg(0), code.Reg(0))
		cc.Emit(opcode)
	case 2:
		opcode := code.MkType0(codeReg(upvals[0]), codeReg(upvals[1]), code.Reg(0))
		cc.Emit(opcode)
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

func (m MkCont) Compile(cc *code.Compiler) {
	opcode := code.MkType4a(code.Off, code.OpCont, codeReg(m.Dst), codeReg(m.Closure))
	cc.Emit(opcode)
}

func (m MkCont) String() string {
	return fmt.Sprintf("%s := mkcont(%s)", m.Dst, m.Closure)
}

type MkCell struct {
	Dst Register
	Src Register
}

func (m MkCell) Compile(cc *code.Compiler) {
	// TODO: perhaps this instruction is not needed.
}

func (m MkCell) String() string {
	return fmt.Sprintf("%s := mkcell(%s)", m.Dst, m.Src)
}

type MkTable struct {
	Dst Register
}

func (m MkTable) Compile(cc *code.Compiler) {
	opcode := code.MkType4b(code.Off, code.OpTable, codeReg(m.Dst), code.Lit8(0))
	cc.Emit(opcode)
}

func (m MkTable) String() string {
	return fmt.Sprintf("%s := mktable()", m.Dst)
}

type Lookup struct {
	Dst   Register
	Table Register
	Index Register
}

func (s Lookup) Compile(cc *code.Compiler) {
	opcode := code.MkType2(code.Off, codeReg(s.Dst), codeReg(s.Table), codeReg(s.Index))
	cc.Emit(opcode)
}

func (s Lookup) String() string {
	return fmt.Sprintf("%s := %s[%s]", s.Dst, s.Table, s.Index)
}

type SetIndex struct {
	Table Register
	Index Register
	Src   Register
}

func (s SetIndex) Compile(cc *code.Compiler) {
	opcode := code.MkType2(code.On, codeReg(s.Src), codeReg(s.Table), codeReg(s.Index))
	cc.Emit(opcode)
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

func (r Receive) Compile(cc *code.Compiler) {
	dst := r.Dst
	for len(dst) >= 3 {
		opcode := code.MkType6(code.Off, 3, codeReg(dst[0]), codeReg(dst[1]), codeReg(dst[2]))
		cc.Emit(opcode)
		dst = dst[3:]
	}
	switch len(dst) {
	case 1:
		opcode := code.MkType6(code.Off, 1, codeReg(dst[0]), code.Reg(0), code.Reg(0))
		cc.Emit(opcode)
	case 2:
		opcode := code.MkType6(code.Off, 2, codeReg(dst[0]), codeReg(dst[1]), code.Reg(0))
		cc.Emit(opcode)
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

func (r ReceiveEtc) Compile(cc *code.Compiler) {
	dst := r.Dst
	for len(dst) >= 3 {
		opcode := code.MkType6(code.Off, 3, codeReg(dst[0]), codeReg(dst[1]), codeReg(dst[2]))
		cc.Emit(opcode)
		dst = dst[3:]
	}
	switch len(dst) {
	case 0:
		opcode := code.MkType6(code.On, 0, codeReg(r.Etc), code.Reg(0), code.Reg(0))
		cc.Emit(opcode)
	case 1:
		opcode := code.MkType6(code.Off, 1, codeReg(dst[0]), codeReg(r.Etc), code.Reg(0))
		cc.Emit(opcode)
	case 2:
		opcode := code.MkType6(code.Off, 2, codeReg(dst[0]), codeReg(dst[1]), codeReg(r.Etc))
		cc.Emit(opcode)
	}
}

type JumpIfForLoopDone struct {
	Label Label
	Var   Register
	Limit Register
	Step  Register
}

func (j JumpIfForLoopDone) Compile(cc *code.Compiler) {
	addr := cc.GetLabelAddr(code.Label(j.Label))
	opcode := code.MkType5(code.Off, code.OpJumpIfForLoopDone, code.Reg(0), code.Lit16(addr))
	cc.Emit(opcode)
}

func (j JumpIfForLoopDone) String() string {
	return fmt.Sprintf("jump %s if for loop done(%s, %s, %s)", j.Label, j.Var, j.Limit, j.Step)
}
