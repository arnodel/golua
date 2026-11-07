package ir

import (
	"fmt"
	"strings"

	"github.com/arnodel/golua/ops"
)

// Instruction is the interface that all ir instruction types must implement.
type Instruction interface {
	fmt.Stringer
	Operands
	ProcessInstr(InstrProcessor)
}

// Operands describes which registers an instruction reads and which it
// writes, and lets a pass substitute one read register for another.  Every
// instruction implements it; a register that an instruction both reads and
// writes (a for loop's control registers, say) appears in both lists.
//
// A table being stored into or filled counts as read: the instruction reads
// the register to find the table.  Upvalues captured by a closure count as
// read too, although they are cells shared by reference; passes do not
// substitute cells.
type Operands interface {
	Reads() []Register
	Writes() []Register
	// ReplaceRead returns the instruction with every read of from reading to
	// instead.  Writes are not affected.
	ReplaceRead(from, to Register) Instruction
}

func sub(r, from, to Register) Register {
	if r == from {
		return to
	}
	return r
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
	ProcessMkVarargTableInstr(MkVarargTable)
	ProcessLookupInstr(Lookup)
	ProcessLookupKInstr(LookupK)
	ProcessSetIndexInstr(SetIndex)
	ProcessSetIndexKInstr(SetIndexK)
	ProcessReceiveInstr(Receive)
	ProcessReceiveEtcInstr(ReceiveEtc)
	ProcessEtcLookupInstr(EtcLookup)
	ProcessFillTableInstr(FillTable)
	ProcessTruncateCloseStackInstr(TruncateCloseStack)
	ProcessPushCloseStackInstr(PushCloseStack)
	ProcessPrepForLoopInstr(PrepForLoop)
	ProcessAdvForLoopInstr(AdvForLoop)

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

func (c Combine) Reads() []Register  { return []Register{c.Lsrc, c.Rsrc} }
func (c Combine) Writes() []Register { return []Register{c.Dst} }
func (c Combine) ReplaceRead(from, to Register) Instruction {
	c.Lsrc, c.Rsrc = sub(c.Lsrc, from, to), sub(c.Rsrc, from, to)
	return c
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

func (t Transform) Reads() []Register  { return []Register{t.Src} }
func (t Transform) Writes() []Register { return []Register{t.Dst} }
func (t Transform) ReplaceRead(from, to Register) Instruction {
	t.Src = sub(t.Src, from, to)
	return t
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

func (l LoadConst) Reads() []Register                         { return nil }
func (l LoadConst) Writes() []Register                        { return []Register{l.Dst} }
func (l LoadConst) ReplaceRead(from, to Register) Instruction { return l }

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

func (p Push) Reads() []Register  { return []Register{p.Cont, p.Item} }
func (p Push) Writes() []Register { return nil }
func (p Push) ReplaceRead(from, to Register) Instruction {
	p.Cont, p.Item = sub(p.Cont, from, to), sub(p.Item, from, to)
	return p
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

func (j Jump) Reads() []Register                         { return nil }
func (j Jump) Writes() []Register                        { return nil }
func (j Jump) ReplaceRead(from, to Register) Instruction { return j }

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

func (j JumpIf) Reads() []Register  { return []Register{j.Cond} }
func (j JumpIf) Writes() []Register { return nil }
func (j JumpIf) ReplaceRead(from, to Register) Instruction {
	j.Cond = sub(j.Cond, from, to)
	return j
}

func (j JumpIf) String() string {
	return fmt.Sprintf("jump %s if %s is not %t", j.Label, j.Cond, j.Not)
}

// Call pushes the values of Args into the given continuation, in order, and
// moves execution to it.  Values pushed earlier (an etc, say) come first.
type Call struct {
	Cont Register
	Args []Register
	Tail bool
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (c Call) ProcessInstr(p InstrProcessor) {
	p.ProcessCallInstr(c)
}

func (c Call) Reads() []Register  { return append([]Register{c.Cont}, c.Args...) }
func (c Call) Writes() []Register { return nil }
func (c Call) ReplaceRead(from, to Register) Instruction {
	c.Cont = sub(c.Cont, from, to)
	args := make([]Register, len(c.Args))
	for i, r := range c.Args {
		args[i] = sub(r, from, to)
	}
	c.Args = args
	return c
}

func (c Call) String() string {
	return fmt.Sprintf("call %s%v, tail=%t", c.Cont, c.Args, c.Tail)
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

func (m MkClosure) Reads() []Register  { return m.Upvalues }
func (m MkClosure) Writes() []Register { return []Register{m.Dst} }
func (m MkClosure) ReplaceRead(from, to Register) Instruction {
	ups := make([]Register, len(m.Upvalues))
	for j, r := range m.Upvalues {
		ups[j] = sub(r, from, to)
	}
	m.Upvalues = ups
	return m
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

func (m MkCont) Reads() []Register  { return []Register{m.Closure} }
func (m MkCont) Writes() []Register { return []Register{m.Dst} }
func (m MkCont) ReplaceRead(from, to Register) Instruction {
	m.Closure = sub(m.Closure, from, to)
	return m
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

func (i ClearReg) Reads() []Register                         { return nil }
func (i ClearReg) Writes() []Register                        { return []Register{i.Dst} }
func (i ClearReg) ReplaceRead(from, to Register) Instruction { return i }

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

func (m MkTable) Reads() []Register                         { return nil }
func (m MkTable) Writes() []Register                        { return []Register{m.Dst} }
func (m MkTable) ReplaceRead(from, to Register) Instruction { return m }

func (m MkTable) String() string {
	return fmt.Sprintf("%s := mktable()", m.Dst)
}

// MkVarargTable creates a table whose array part points to the vararg data in Etc.
// This allows modifications to the table to affect what ... expands to (Lua 5.5 semantics).
type MkVarargTable struct {
	Dst Register
	Etc Register
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (m MkVarargTable) ProcessInstr(p InstrProcessor) {
	p.ProcessMkVarargTableInstr(m)
}

func (m MkVarargTable) Reads() []Register  { return []Register{m.Etc} }
func (m MkVarargTable) Writes() []Register { return []Register{m.Dst, m.Etc} }
func (m MkVarargTable) ReplaceRead(from, to Register) Instruction {
	m.Etc = sub(m.Etc, from, to)
	return m
}

func (m MkVarargTable) String() string {
	return fmt.Sprintf("%s := mkvargtable(%s)", m.Dst, m.Etc)
}

// DestReg returns the destination register of this instruction.
func (m MkVarargTable) DestReg() Register {
	return m.Dst
}

// WithDestReg returns the same instruction with a new destination register.
func (m MkVarargTable) WithDestReg(dst Register) Instruction {
	m.Dst = dst
	return m
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

func (s Lookup) Reads() []Register  { return []Register{s.Table, s.Index} }
func (s Lookup) Writes() []Register { return []Register{s.Dst} }
func (s Lookup) ReplaceRead(from, to Register) Instruction {
	s.Table, s.Index = sub(s.Table, from, to), sub(s.Index, from, to)
	return s
}

func (s Lookup) String() string {
	return fmt.Sprintf("%s := %s[%s]", s.Dst, s.Table, s.Index)
}

// LookupK finds the value associated with the constant key Kidx in Table and
// puts it in Dst.  It is Lookup for a key known at compile time, saving the
// instruction that would load the key into a register.
type LookupK struct {
	Dst   Register
	Table Register
	Kidx  uint // Index of the key in the constant table
}

// DestReg returns the destination register of this instruction.
func (s LookupK) DestReg() Register {
	return s.Dst
}

// WithDestReg returns the same isntruction with a new destination register.
func (s LookupK) WithDestReg(r Register) Instruction {
	s.Dst = r
	return s
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (s LookupK) ProcessInstr(p InstrProcessor) {
	p.ProcessLookupKInstr(s)
}

func (s LookupK) Reads() []Register  { return []Register{s.Table} }
func (s LookupK) Writes() []Register { return []Register{s.Dst} }
func (s LookupK) ReplaceRead(from, to Register) Instruction {
	s.Table = sub(s.Table, from, to)
	return s
}

func (s LookupK) String() string {
	return fmt.Sprintf("%s := %s[K%d]", s.Dst, s.Table, s.Kidx)
}

// SetIndex associates Index with Src in the table Table.
// If CheckNotDefined is true, a runtime check is performed first: if
// Table[Index] is already non-nil, a "global 'name' already defined" error
// is raised.  This implements Lua 5.5's global redefinition check.
type SetIndex struct {
	Table           Register
	Index           Register
	Src             Register
	CheckNotDefined bool
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (s SetIndex) ProcessInstr(p InstrProcessor) {
	p.ProcessSetIndexInstr(s)
}

func (s SetIndex) Reads() []Register  { return []Register{s.Table, s.Index, s.Src} }
func (s SetIndex) Writes() []Register { return nil }
func (s SetIndex) ReplaceRead(from, to Register) Instruction {
	s.Table, s.Index, s.Src = sub(s.Table, from, to), sub(s.Index, from, to), sub(s.Src, from, to)
	return s
}

func (s SetIndex) String() string {
	return fmt.Sprintf("%s[%s] := %s", s.Table, s.Index, s.Src)
}

// SetIndexK associates the constant key Kidx with Src in the table Table.  It
// is SetIndex for a key known at compile time.
type SetIndexK struct {
	Table Register
	Kidx  uint // Index of the key in the constant table
	Src   Register
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (s SetIndexK) ProcessInstr(p InstrProcessor) {
	p.ProcessSetIndexKInstr(s)
}

func (s SetIndexK) Reads() []Register  { return []Register{s.Table, s.Src} }
func (s SetIndexK) Writes() []Register { return nil }
func (s SetIndexK) ReplaceRead(from, to Register) Instruction {
	s.Table, s.Src = sub(s.Table, from, to), sub(s.Src, from, to)
	return s
}

func (s SetIndexK) String() string {
	return fmt.Sprintf("%s[K%d] := %s", s.Table, s.Kidx, s.Src)
}

// Receive will put the result of pushes in the given registers.
type Receive struct {
	Dst []Register
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (r Receive) ProcessInstr(p InstrProcessor) {
	p.ProcessReceiveInstr(r)
}

func (r Receive) Reads() []Register                         { return nil }
func (r Receive) Writes() []Register                        { return r.Dst }
func (r Receive) ReplaceRead(from, to Register) Instruction { return r }

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

func (r ReceiveEtc) Reads() []Register                         { return nil }
func (r ReceiveEtc) Writes() []Register                        { return append(append([]Register{}, r.Dst...), r.Etc) }
func (r ReceiveEtc) ReplaceRead(from, to Register) Instruction { return r }

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

func (l EtcLookup) Reads() []Register  { return []Register{l.Etc} }
func (l EtcLookup) Writes() []Register { return []Register{l.Dst} }
func (l EtcLookup) ReplaceRead(from, to Register) Instruction {
	l.Etc = sub(l.Etc, from, to)
	return l
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

func (f FillTable) Reads() []Register  { return []Register{f.Etc, f.Dst} }
func (f FillTable) Writes() []Register { return nil }
func (f FillTable) ReplaceRead(from, to Register) Instruction {
	f.Etc, f.Dst = sub(f.Etc, from, to), sub(f.Dst, from, to)
	return f
}

// TruncateCloseStack truncates the close stack to Height (which should be >=
// 0).  This instruction was introduced to support to-be-closed variables which
// are part of Lua 5.4
type TruncateCloseStack struct {
	Height int
}

func (t TruncateCloseStack) String() string {
	return fmt.Sprintf("trunc close stack to %d", t.Height)
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (t TruncateCloseStack) ProcessInstr(p InstrProcessor) {
	p.ProcessTruncateCloseStackInstr(t)
}

func (t TruncateCloseStack) Reads() []Register                         { return nil }
func (t TruncateCloseStack) Writes() []Register                        { return nil }
func (t TruncateCloseStack) ReplaceRead(from, to Register) Instruction { return t }

// PushCloseStack truncates pushes the value Src to the close stack.  This
// instruction was introduced to support to-be-closed variables which are part
// of Lua 5.4
type PushCloseStack struct {
	Src Register
}

func (i PushCloseStack) String() string {
	return fmt.Sprintf("push %s to close stack", i.Src)
}

// ProcessInstr makes the InstrProcessor process this instruction.
func (i PushCloseStack) ProcessInstr(p InstrProcessor) {
	p.ProcessPushCloseStackInstr(i)
}

func (i PushCloseStack) Reads() []Register  { return []Register{i.Src} }
func (i PushCloseStack) Writes() []Register { return nil }
func (i PushCloseStack) ReplaceRead(from, to Register) Instruction {
	i.Src = sub(i.Src, from, to)
	return i
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

func (t TakeRegister) Reads() []Register                         { return nil }
func (t TakeRegister) Writes() []Register                        { return nil }
func (t TakeRegister) ReplaceRead(from, to Register) Instruction { return t }

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

func (r ReleaseRegister) Reads() []Register                         { return nil }
func (r ReleaseRegister) Writes() []Register                        { return nil }
func (r ReleaseRegister) ReplaceRead(from, to Register) Instruction { return r }

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

func (l DeclareLabel) Reads() []Register                         { return nil }
func (l DeclareLabel) Writes() []Register                        { return nil }
func (l DeclareLabel) ReplaceRead(from, to Register) Instruction { return l }

// PrepForLoop prepares a for loop
type PrepForLoop struct {
	Start, Stop, Step Register
}

func (i PrepForLoop) String() string {
	return fmt.Sprintf("prepfor %s, %s, %s", i.Start, i.Stop, i.Step)
}

func (i PrepForLoop) ProcessInstr(p InstrProcessor) {
	p.ProcessPrepForLoopInstr(i)
}

func (i PrepForLoop) Reads() []Register  { return []Register{i.Start, i.Stop, i.Step} }
func (i PrepForLoop) Writes() []Register { return []Register{i.Start, i.Stop, i.Step} }
func (i PrepForLoop) ReplaceRead(from, to Register) Instruction {
	i.Start, i.Stop, i.Step = sub(i.Start, from, to), sub(i.Stop, from, to), sub(i.Step, from, to)
	return i
}

// AdvForLoop advances a for loop

type AdvForLoop struct {
	Start, Stop, Step Register
}

func (i AdvForLoop) String() string {
	return fmt.Sprintf("advfor %s, %s, %s", i.Start, i.Stop, i.Step)
}

func (i AdvForLoop) ProcessInstr(p InstrProcessor) {
	p.ProcessAdvForLoopInstr(i)
}

func (i AdvForLoop) Reads() []Register  { return []Register{i.Start, i.Stop, i.Step} }
func (i AdvForLoop) Writes() []Register { return []Register{i.Start, i.Stop, i.Step} }
func (i AdvForLoop) ReplaceRead(from, to Register) Instruction {
	i.Start, i.Stop, i.Step = sub(i.Start, from, to), sub(i.Stop, from, to), sub(i.Step, from, to)
	return i
}
