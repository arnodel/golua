package ir

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

func (i Combine) Reads() []Register  { return []Register{i.Lsrc, i.Rsrc} }
func (i Combine) Writes() []Register { return []Register{i.Dst} }
func (i Combine) ReplaceRead(from, to Register) Instruction {
	i.Lsrc, i.Rsrc = sub(i.Lsrc, from, to), sub(i.Rsrc, from, to)
	return i
}

func (i Transform) Reads() []Register  { return []Register{i.Src} }
func (i Transform) Writes() []Register { return []Register{i.Dst} }
func (i Transform) ReplaceRead(from, to Register) Instruction {
	i.Src = sub(i.Src, from, to)
	return i
}

func (i LoadConst) Reads() []Register                         { return nil }
func (i LoadConst) Writes() []Register                        { return []Register{i.Dst} }
func (i LoadConst) ReplaceRead(from, to Register) Instruction { return i }

func (i Push) Reads() []Register  { return []Register{i.Cont, i.Item} }
func (i Push) Writes() []Register { return nil }
func (i Push) ReplaceRead(from, to Register) Instruction {
	i.Cont, i.Item = sub(i.Cont, from, to), sub(i.Item, from, to)
	return i
}

func (i Jump) Reads() []Register                         { return nil }
func (i Jump) Writes() []Register                        { return nil }
func (i Jump) ReplaceRead(from, to Register) Instruction { return i }

func (i JumpIf) Reads() []Register  { return []Register{i.Cond} }
func (i JumpIf) Writes() []Register { return nil }
func (i JumpIf) ReplaceRead(from, to Register) Instruction {
	i.Cond = sub(i.Cond, from, to)
	return i
}

func (i Call) Reads() []Register  { return []Register{i.Cont} }
func (i Call) Writes() []Register { return nil }
func (i Call) ReplaceRead(from, to Register) Instruction {
	i.Cont = sub(i.Cont, from, to)
	return i
}

func (i MkClosure) Reads() []Register  { return i.Upvalues }
func (i MkClosure) Writes() []Register { return []Register{i.Dst} }
func (i MkClosure) ReplaceRead(from, to Register) Instruction {
	ups := make([]Register, len(i.Upvalues))
	for j, r := range i.Upvalues {
		ups[j] = sub(r, from, to)
	}
	i.Upvalues = ups
	return i
}

func (i MkCont) Reads() []Register  { return []Register{i.Closure} }
func (i MkCont) Writes() []Register { return []Register{i.Dst} }
func (i MkCont) ReplaceRead(from, to Register) Instruction {
	i.Closure = sub(i.Closure, from, to)
	return i
}

func (i ClearReg) Reads() []Register                         { return nil }
func (i ClearReg) Writes() []Register                        { return []Register{i.Dst} }
func (i ClearReg) ReplaceRead(from, to Register) Instruction { return i }

func (i MkTable) Reads() []Register                         { return nil }
func (i MkTable) Writes() []Register                        { return []Register{i.Dst} }
func (i MkTable) ReplaceRead(from, to Register) Instruction { return i }

// MkVarargTable also replaces the etc register with the table.
func (i MkVarargTable) Reads() []Register  { return []Register{i.Etc} }
func (i MkVarargTable) Writes() []Register { return []Register{i.Dst, i.Etc} }
func (i MkVarargTable) ReplaceRead(from, to Register) Instruction {
	i.Etc = sub(i.Etc, from, to)
	return i
}

func (i Lookup) Reads() []Register  { return []Register{i.Table, i.Index} }
func (i Lookup) Writes() []Register { return []Register{i.Dst} }
func (i Lookup) ReplaceRead(from, to Register) Instruction {
	i.Table, i.Index = sub(i.Table, from, to), sub(i.Index, from, to)
	return i
}

func (i LookupK) Reads() []Register  { return []Register{i.Table} }
func (i LookupK) Writes() []Register { return []Register{i.Dst} }
func (i LookupK) ReplaceRead(from, to Register) Instruction {
	i.Table = sub(i.Table, from, to)
	return i
}

func (i SetIndex) Reads() []Register  { return []Register{i.Table, i.Index, i.Src} }
func (i SetIndex) Writes() []Register { return nil }
func (i SetIndex) ReplaceRead(from, to Register) Instruction {
	i.Table, i.Index, i.Src = sub(i.Table, from, to), sub(i.Index, from, to), sub(i.Src, from, to)
	return i
}

func (i SetIndexK) Reads() []Register  { return []Register{i.Table, i.Src} }
func (i SetIndexK) Writes() []Register { return nil }
func (i SetIndexK) ReplaceRead(from, to Register) Instruction {
	i.Table, i.Src = sub(i.Table, from, to), sub(i.Src, from, to)
	return i
}

func (i Receive) Reads() []Register                         { return nil }
func (i Receive) Writes() []Register                        { return i.Dst }
func (i Receive) ReplaceRead(from, to Register) Instruction { return i }

func (i ReceiveEtc) Reads() []Register                         { return nil }
func (i ReceiveEtc) Writes() []Register                        { return append(append([]Register{}, i.Dst...), i.Etc) }
func (i ReceiveEtc) ReplaceRead(from, to Register) Instruction { return i }

func (i EtcLookup) Reads() []Register  { return []Register{i.Etc} }
func (i EtcLookup) Writes() []Register { return []Register{i.Dst} }
func (i EtcLookup) ReplaceRead(from, to Register) Instruction {
	i.Etc = sub(i.Etc, from, to)
	return i
}

// FillTable fills the table in Dst, which it reads to find the table.
func (i FillTable) Reads() []Register  { return []Register{i.Etc, i.Dst} }
func (i FillTable) Writes() []Register { return nil }
func (i FillTable) ReplaceRead(from, to Register) Instruction {
	i.Etc, i.Dst = sub(i.Etc, from, to), sub(i.Dst, from, to)
	return i
}

func (i TruncateCloseStack) Reads() []Register                         { return nil }
func (i TruncateCloseStack) Writes() []Register                        { return nil }
func (i TruncateCloseStack) ReplaceRead(from, to Register) Instruction { return i }

func (i PushCloseStack) Reads() []Register  { return []Register{i.Src} }
func (i PushCloseStack) Writes() []Register { return nil }
func (i PushCloseStack) ReplaceRead(from, to Register) Instruction {
	i.Src = sub(i.Src, from, to)
	return i
}

// TakeRegister and ReleaseRegister mark a register's lifetime; they neither
// read nor write it.
func (i TakeRegister) Reads() []Register                         { return nil }
func (i TakeRegister) Writes() []Register                        { return nil }
func (i TakeRegister) ReplaceRead(from, to Register) Instruction { return i }

func (i ReleaseRegister) Reads() []Register                         { return nil }
func (i ReleaseRegister) Writes() []Register                        { return nil }
func (i ReleaseRegister) ReplaceRead(from, to Register) Instruction { return i }

func (i DeclareLabel) Reads() []Register                         { return nil }
func (i DeclareLabel) Writes() []Register                        { return nil }
func (i DeclareLabel) ReplaceRead(from, to Register) Instruction { return i }

// The for loop instructions update their control registers in place.
func (i PrepForLoop) Reads() []Register  { return []Register{i.Start, i.Stop, i.Step} }
func (i PrepForLoop) Writes() []Register { return []Register{i.Start, i.Stop, i.Step} }
func (i PrepForLoop) ReplaceRead(from, to Register) Instruction {
	i.Start, i.Stop, i.Step = sub(i.Start, from, to), sub(i.Stop, from, to), sub(i.Step, from, to)
	return i
}

func (i AdvForLoop) Reads() []Register  { return []Register{i.Start, i.Stop, i.Step} }
func (i AdvForLoop) Writes() []Register { return []Register{i.Start, i.Stop, i.Step} }
func (i AdvForLoop) ReplaceRead(from, to Register) Instruction {
	i.Start, i.Stop, i.Step = sub(i.Start, from, to), sub(i.Stop, from, to), sub(i.Step, from, to)
	return i
}

// Every instruction implements Operands.
var (
	_ Operands = Combine{}
	_ Operands = Transform{}
	_ Operands = LoadConst{}
	_ Operands = Push{}
	_ Operands = Jump{}
	_ Operands = JumpIf{}
	_ Operands = Call{}
	_ Operands = MkClosure{}
	_ Operands = MkCont{}
	_ Operands = ClearReg{}
	_ Operands = MkTable{}
	_ Operands = MkVarargTable{}
	_ Operands = Lookup{}
	_ Operands = LookupK{}
	_ Operands = SetIndex{}
	_ Operands = SetIndexK{}
	_ Operands = Receive{}
	_ Operands = ReceiveEtc{}
	_ Operands = EtcLookup{}
	_ Operands = FillTable{}
	_ Operands = TruncateCloseStack{}
	_ Operands = PushCloseStack{}
	_ Operands = TakeRegister{}
	_ Operands = ReleaseRegister{}
	_ Operands = DeclareLabel{}
	_ Operands = PrepForLoop{}
	_ Operands = AdvForLoop{}
)
