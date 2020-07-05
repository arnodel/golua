package ir

import (
	"fmt"
	"strings"

	"github.com/arnodel/golua/code"
	"github.com/arnodel/golua/ops"
)

type Instruction interface {
	fmt.Stringer
	Compile(InstrCompiler)
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

// Combine applies the binary operator Op to Lsrc and Rsrc and stores the result
// in Dst.
type Combine struct {
	Op   ops.Op   // Operator to apply to Lsrc and Rsrc
	Dst  Register // Destination register
	Lsrc Register // Left operand register
	Rsrc Register // Right operand register
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
	ops.OpNeg:      code.OpNeg,
	ops.OpNot:      code.OpNot,
	ops.OpLen:      code.OpLen,
	ops.OpBitNot:   code.OpBitNot,
	ops.OpId:       code.OpId,
	ops.OpToNumber: code.OpToNumber,
}

func (c Combine) Compile(kc InstrCompiler) {
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

// Transform applies a unary operator Op to Src and stores the result in Dst.
type Transform struct {
	Op  ops.Op   // Operator to apply to Src
	Dst Register // Destination register
	Src Register // Operand register
}

func (t Transform) Compile(kc InstrCompiler) {
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

// LoadConst loads a constant into a register.
type LoadConst struct {
	Dst  Register // Destination register
	Kidx uint     // Index of the constant to load
}

func (l LoadConst) Compile(kc InstrCompiler) {
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

// Push pushes the contents of a register into a continuation.
type Push struct {
	Cont Register // Destination register (should contain a continuation)
	Item Register // Register containing item to push
	Etc  bool     // True if the Item is an etc value.
}

func (p Push) Compile(kc InstrCompiler) {
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

// Jump jumps to the givel label.
type Jump struct {
	Label Label
}

func (j Jump) String() string {
	return fmt.Sprintf("jump %s", j.Label)
}

func (j Jump) Compile(kc InstrCompiler) {
	opcode := code.MkType5(code.Off, code.OpJump, code.Reg(0), code.Lit16(0))
	kc.EmitJump(opcode, code.Label(j.Label))
}

// JumpIf jumps to the given label if the boolean value in Cond is different from Not.
type JumpIf struct {
	Cond  Register
	Label Label
	Not   bool
}

func (j JumpIf) Compile(kc InstrCompiler) {
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

// Call moves execution to the given continuation
type Call struct {
	Cont Register
}

func (c Call) Compile(kc InstrCompiler) {
	// TODO: tailcall
	opcode := code.MkType5(code.Off, code.OpCall, codeReg(c.Cont), code.Lit16(0))
	kc.Emit(opcode)
}

func (c Call) String() string {
	return fmt.Sprintf("call %s", c.Cont)
}

// MkClosure creates a new closure with the given code and upvalues and puts it in Dst.
type MkClosure struct {
	Dst      Register
	Code     uint
	Upvalues []Register
}

func (m MkClosure) Compile(kc InstrCompiler) {
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

// MkCont creates a new continuation for the given closure and puts it in Dst.
type MkCont struct {
	Dst     Register
	Closure Register
	Tail    bool
}

func (m MkCont) Compile(kc InstrCompiler) {
	op := code.OpCont
	if m.Tail {
		op = code.OpTailCont
	}
	opcode := code.MkType4a(code.Off, op, codeReg(m.Dst), codeReg(m.Closure))
	kc.Emit(opcode)
}

func (m MkCont) String() string {
	return fmt.Sprintf("%s := mkcont(%s)", m.Dst, m.Closure)
}

// ClearReg resets the given register to nil (if it contained a cell, this cell
// is removed).
type ClearReg struct {
	Dst Register
}

func (i ClearReg) Compile(kc InstrCompiler) {
	opcode := code.MkType4b(code.Off, code.OpClear, codeReg(i.Dst), code.Lit8(0))
	kc.Emit(opcode)
}

func (i ClearReg) String() string {
	return fmt.Sprintf("clrreg(%s)", i.Dst)
}

// MkTable creates a new empty table and puts it i Dst.
type MkTable struct {
	Dst Register
}

func (m MkTable) Compile(kc InstrCompiler) {
	opcode := code.MkType4b(code.Off, code.OpTable, codeReg(m.Dst), code.Lit8(0))
	kc.Emit(opcode)
}

func (m MkTable) String() string {
	return fmt.Sprintf("%s := mktable()", m.Dst)
}

// Lookup finds the value associated with the key Index in Table and puts it in
// Dst.
type Lookup struct {
	Dst   Register
	Table Register
	Index Register
}

func (s Lookup) Compile(kc InstrCompiler) {
	opcode := code.MkType2(code.Off, codeReg(s.Dst), codeReg(s.Table), codeReg(s.Index))
	kc.Emit(opcode)
}

func (s Lookup) String() string {
	return fmt.Sprintf("%s := %s[%s]", s.Dst, s.Table, s.Index)
}

// SetIndex associates Index with Src in the table Table.
type SetIndex struct {
	Table Register
	Index Register
	Src   Register
}

func (s SetIndex) Compile(kc InstrCompiler) {
	opcode := code.MkType2(code.On, codeReg(s.Src), codeReg(s.Table), codeReg(s.Index))
	kc.Emit(opcode)
}

func (s SetIndex) String() string {
	return fmt.Sprintf("%s[%s] := %s", s.Table, s.Index, s.Src)
}

// Receive will put the result of pushes in the given registers.
type Receive struct {
	Dst []Register
}

func (r Receive) Compile(kc InstrCompiler) {
	for _, reg := range r.Dst {
		kc.Emit(code.MkType0(code.Off, codeReg(reg)))
	}
}

func (r Receive) String() string {
	return fmt.Sprintf("recv(%s)", joinRegisters(r.Dst, ", "))
}

// ReceiveEtc will put the result of pushes into the given registers.  Extra
// pushes will be accumulated into the Etc register.
type ReceiveEtc struct {
	Dst []Register
	Etc Register
}

func (r ReceiveEtc) String() string {
	return fmt.Sprintf("recv(%s, ...%s)", joinRegisters(r.Dst, ", "), r.Etc)
}

func (r ReceiveEtc) Compile(kc InstrCompiler) {
	for _, reg := range r.Dst {
		kc.Emit(code.MkType0(code.Off, codeReg(reg)))
	}
	kc.Emit(code.MkType0(code.On, codeReg(r.Etc)))
}

// EtcLookup finds the value at index Idx in the Etc register and puts it in
// Dst.
type EtcLookup struct {
	Etc Register
	Dst Register
	Idx int
}

func (l EtcLookup) String() string {
	return fmt.Sprintf("%s := %s[%d]", l.Dst, l.Etc, l.Idx)
}

func (l EtcLookup) Compile(kc InstrCompiler) {
	if l.Idx < 0 || l.Idx >= 256 {
		panic("Etc lookup index out of range")
	}
	kc.Emit(code.MkType6(code.Off, codeReg(l.Dst), codeReg(l.Etc), uint8(l.Idx)))
}

// FillTable fills Dst (which must contain a table) with the contents of Etc
// (which must be an etc value) starting from the given index.
type FillTable struct {
	Etc Register
	Dst Register
	Idx int
}

func (f FillTable) String() string {
	return fmt.Sprintf("fill %s with %s from %d", f.Dst, f.Etc, f.Idx)
}

func (f FillTable) Compile(kc InstrCompiler) {
	if f.Idx < 0 || f.Idx >= 256 {
		panic("Fill table index out of range")
	}
	kc.Emit(code.MkType6(code.On, codeReg(f.Dst), codeReg(f.Etc), uint8(f.Idx)))
}

func codeReg(r Register) code.Reg {
	if r >= 0 {
		return code.MkRegister(uint8(r))
	}
	return code.MkUpvalue(uint8(-1 - r))
}
