package ir

import (
	"fmt"
	"strings"

	"github.com/arnodel/golua/ops"
)

// Instruction is the interface that all ir instruction types must implement.
type Instruction interface {
	fmt.Stringer
	ProcessInstr(InstrProcessor)
}

// SetRegInstruction is the interface that instructions that set the value of a
// register must implement.
type SetRegInstruction interface {
	DestReg() Register
	WithDestReg(Register) Instruction
}

// An InstrProcessor can process any instruction.
type InstrProcessor interface {

	// Real instructions
	ProcessCombineInstr(Combine)
	ProcessTransformInstr(Transform)
	ProcessLoadConstInstr(LoadConst)
	ProcessPushInstr(Push)
	ProcessJumpInstr(Jump)
	ProcessJumpIfInstr(JumpIf)
	ProcessCallInstr(Call)
	ProcessMkClosureInstr(MkClosure)
	ProcessMkContInstr(MkCont)
	ProcessClearRegInstr(ClearReg)
	ProcessMkTableInstr(MkTable)
	ProcessLookupInstr(Lookup)
	ProcessSetIndexInstr(SetIndex)
	ProcessReceiveInstr(Receive)
	ProcessReceiveEtcInstr(ReceiveEtc)
	ProcessEtcLookupInstr(EtcLookup)
	ProcessFillTableInstr(FillTable)

	// These are hints that a register is needed or no longer needed.
	ProcessTakeRegisterInstr(TakeRegister)
	ProcessReleaseRegisterInstr(ReleaseRegister)

	// A label (for jumping to)
	ProcessDeclareLabelInstr(DeclareLabel)
}

// A Register is an IR register.  The number of IR registers is not bounded
// (other than the bounds of the underling type).
type Register uint

func (r Register) String() string {
	return fmt.Sprintf("r%d", r)
}

// A Label is a location in the IR code.
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

// DestReg returns the destination register of this instruction.
func (c Combine) DestReg() Register {
	return c.Dst
}

// WithDestReg returns the same isntruction with a new destination register.
func (c Combine) WithDestReg(r Register) Instruction {
	c.Dst = r
	return c
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (c Combine) ProcessInstr(p InstrProcessor) {
	p.ProcessCombineInstr(c)
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

// DestReg returns the destination register of this instruction.
func (t Transform) DestReg() Register {
	return t.Dst
}

// WithDestReg returns the same isntruction with a new destination register.
func (t Transform) WithDestReg(r Register) Instruction {
	t.Dst = r
	return t
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (t Transform) ProcessInstr(p InstrProcessor) {
	p.ProcessTransformInstr(t)
}

func (t Transform) String() string {
	return fmt.Sprintf("%s := %s(%s)", t.Dst, t.Op, t.Src)
}

// LoadConst loads a constant into a register.
type LoadConst struct {
	Dst  Register // Destination register
	Kidx uint     // Index of the constant to load
}

// DestReg returns the destination register of this instruction.
func (l LoadConst) DestReg() Register {
	return l.Dst
}

// WithDestReg returns the same isntruction with a new destination register.
func (l LoadConst) WithDestReg(r Register) Instruction {
	l.Dst = r
	return l
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (l LoadConst) ProcessInstr(p InstrProcessor) {
	p.ProcessLoadConstInstr(l)
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

// ProcessInstr makes the InstrProcessor process this instruction.
func (p Push) ProcessInstr(ip InstrProcessor) {
	ip.ProcessPushInstr(p)
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

// ProcessInstr makes the InstrProcessor process this instruction.
func (j Jump) ProcessInstr(p InstrProcessor) {
	p.ProcessJumpInstr(j)
}

// JumpIf jumps to the given label if the boolean value in Cond is different from Not.
type JumpIf struct {
	Cond  Register
	Label Label
	Not   bool
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (j JumpIf) ProcessInstr(p InstrProcessor) {
	p.ProcessJumpIfInstr(j)
}

func (j JumpIf) String() string {
	return fmt.Sprintf("jump %s if %s is not %t", j.Label, j.Cond, j.Not)
}

// Call moves execution to the given continuation
type Call struct {
	Cont Register
	Tail bool
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (c Call) ProcessInstr(p InstrProcessor) {
	p.ProcessCallInstr(c)
}

func (c Call) String() string {
	return fmt.Sprintf("call %s, tail=%t", c.Cont, c.Tail)
}

// MkClosure creates a new closure with the given code and upvalues and puts it in Dst.
type MkClosure struct {
	Dst      Register
	Code     uint
	Upvalues []Register
}

// DestReg returns the destination register of this instruction.
func (m MkClosure) DestReg() Register {
	return m.Dst
}

// WithDestReg returns the same isntruction with a new destination register.
func (m MkClosure) WithDestReg(r Register) Instruction {
	m.Dst = r
	return m
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (m MkClosure) ProcessInstr(p InstrProcessor) {
	p.ProcessMkClosureInstr(m)
}

func (m MkClosure) String() string {
	return fmt.Sprintf("%s := mkclos(k%d; %s)", m.Dst, m.Code, joinRegisters(m.Upvalues, ", "))
}

func joinRegisters(regs []Register, sep string) string {
	us := []string{}
	for _, r := range regs {
		us = append(us, r.String())
	}
	return strings.Join(us, sep)
}

// MkCont creates a new continuation for the given closure and puts it in Dst.
type MkCont struct {
	Dst     Register
	Closure Register
	Tail    bool
}

// DestReg returns the destination register of this instruction.
func (m MkCont) DestReg() Register {
	return m.Dst
}

// WithDestReg returns the same isntruction with a new destination register.
func (m MkCont) WithDestReg(r Register) Instruction {
	m.Dst = r
	return m
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (m MkCont) ProcessInstr(p InstrProcessor) {
	p.ProcessMkContInstr(m)
}

func (m MkCont) String() string {
	return fmt.Sprintf("%s := mkcont(%s, tail=%t)", m.Dst, m.Closure, m.Tail)
}

// ClearReg resets the given register to nil (if it contained a cell, this cell
// is removed).
type ClearReg struct {
	Dst Register
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (i ClearReg) ProcessInstr(p InstrProcessor) {
	p.ProcessClearRegInstr(i)
}

func (i ClearReg) String() string {
	return fmt.Sprintf("clrreg(%s)", i.Dst)
}

// MkTable creates a new empty table and puts it i Dst.
type MkTable struct {
	Dst Register
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (m MkTable) ProcessInstr(p InstrProcessor) {
	p.ProcessMkTableInstr(m)
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

// DestReg returns the destination register of this instruction.
func (s Lookup) DestReg() Register {
	return s.Dst
}

// WithDestReg returns the same isntruction with a new destination register.
func (s Lookup) WithDestReg(r Register) Instruction {
	s.Dst = r
	return s
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (s Lookup) ProcessInstr(p InstrProcessor) {
	p.ProcessLookupInstr(s)
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

// ProcessInstr makes the InstrProcessor process this instruction.
func (s SetIndex) ProcessInstr(p InstrProcessor) {
	p.ProcessSetIndexInstr(s)
}

func (s SetIndex) String() string {
	return fmt.Sprintf("%s[%s] := %s", s.Table, s.Index, s.Src)
}

// Receive will put the result of pushes in the given registers.
type Receive struct {
	Dst []Register
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (r Receive) ProcessInstr(p InstrProcessor) {
	p.ProcessReceiveInstr(r)
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

// ProcessInstr makes the InstrProcessor process this instruction.
func (r ReceiveEtc) ProcessInstr(p InstrProcessor) {
	p.ProcessReceiveEtcInstr(r)
}

// EtcLookup finds the value at index Idx in the Etc register and puts it in
// Dst.
type EtcLookup struct {
	Etc Register
	Dst Register
	Idx int
}

// DestReg returns the destination register of this instruction.
func (l EtcLookup) DestReg() Register {
	return l.Dst
}

// WithDestReg returns the same isntruction with a new destination register.
func (l EtcLookup) WithDestReg(r Register) Instruction {
	l.Dst = r
	return l
}

func (l EtcLookup) String() string {
	return fmt.Sprintf("%s := %s[%d]", l.Dst, l.Etc, l.Idx)
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (l EtcLookup) ProcessInstr(p InstrProcessor) {
	p.ProcessEtcLookupInstr(l)
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

// ProcessInstr makes the InstrProcessor process this instruction.
func (f FillTable) ProcessInstr(p InstrProcessor) {
	p.ProcessFillTableInstr(f)
}

// TakeRegister is not a real instruction.  It is a hint to the next stage that
// the value in this register will need to be used, so not to overwrite it.  A
// ReleaseRegister for the same register should be emitted some time later.
type TakeRegister struct {
	Reg Register
}

func (t TakeRegister) String() string {
	return fmt.Sprintf("take %s", t.Reg)
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (t TakeRegister) ProcessInstr(p InstrProcessor) {
	p.ProcessTakeRegisterInstr(t)
}

// ReleaseRegister is not a real instruction.  It is a hint to the next stage
// that the value in this register no longer needs to be used, so can be
// overwritten.  It should be preceded by a TakeRegister for the same register.
type ReleaseRegister struct {
	Reg Register
}

func (r ReleaseRegister) String() string {
	return fmt.Sprintf("release %s", r.Reg)
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (r ReleaseRegister) ProcessInstr(p InstrProcessor) {
	p.ProcessReleaseRegisterInstr(r)
}

// DeclareLabel is not a real IR instruction.  It is a placeholder for the
// destination location of a jump label.  Being an instruction makes it easier
// to perform IR code transformations and keep the labels in correct locations.
type DeclareLabel struct {
	Label Label
}

func (l DeclareLabel) String() string {
	return fmt.Sprintf("L%d:", l.Label)
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (l DeclareLabel) ProcessInstr(p InstrProcessor) {
	p.ProcessDeclareLabelInstr(l)
}
