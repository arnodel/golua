package code

// Combine encodes r1 <- op(r2, r3)
func Combine(op BinOp, r1, r2, r3 Reg) Opcode {
	return mkType1(op, r1, r2, r3)
}

// Transform encodes r1 <- op(r2)
func Transform(op UnOp, r1, r2 Reg) Opcode {
	return mkType4a(Off, op, r1, r2)
}

// LoadConst encodes r <- Ki
func LoadConst(r Reg, i KIndex) Opcode {
	return mkType3(Off, OpK, r, i)
}

// LoadClosure encodes r <- clos(Ki)
func LoadClosure(r1 Reg, i KIndex) Opcode {
	return mkType3(Off, OpClosureK, r1, i)
}

// LoadInt16 encodes r <- n
func LoadInt16(r Reg, n int16) Opcode {
	return mkType3(Off, OpInt16, r, Lit16(n))
}

// LoadSmallInt attempts to load a small integer (atm it has to be representable
// as an int16).
func LoadSmallInt(r Reg, n int) (Opcode, bool) {
	sn := int16(n)
	if int(sn) != n {
		return 0, false
	}
	return LoadInt16(r, sn), true
}

// LoadStr0 encodes r <- ""
func LoadStr0(r Reg) Opcode {
	return mkType4b(Off, OpStr0, r, 0)
}

// LoadStr1 encodes r <- "x"
func LoadStr1(r Reg, b []byte) Opcode {
	return mkType4b(Off, OpStr1, r, Lit8FromStr1(b))
}

// LoadStr2 encodes r <- "xy"
func LoadStr2(r Reg, b []byte) Opcode {
	return mkType3(Off, OpStr2, r, Lit16FromStr2(b))
}

// LoadShortString attempts to encode loading a short string.  Currently succees
// when len(b) <= 2.
func LoadShortString(r Reg, b []byte) (Opcode, bool) {
	// This code is commented out because it turns out that it causes many
	// allocations, slowing down the runtime considerably in some cases.
	//
	// switch len(b) {case 0:
	//  return LoadStr0(r), true
	// case 1:
	//  return LoadStr1(r, b), true
	// case 2:
	//  return LoadStr2(r, b), true
	// }
	return 0, false
}

// LoadBool encodes r <- true or r <- false.
func LoadBool(r Reg, b bool) Opcode {
	return mkType4b(Off, OpBool, r, Lit8FromBool(b))
}

// LoadEmptyTable encodes r <- {}
func LoadEmptyTable(r Reg) Opcode {
	return mkType4b(Off, OpTable, r, 0)
}

// LoadNil encodes r <- nil
func LoadNil(r Reg) Opcode {
	return mkType4b(Off, OpNil, r, 0)
}

// LoadLookup encodes r1 <- r2[r3]
func LoadLookup(r1, r2, r3 Reg) Opcode {
	return mkType2(Off, r1, r2, r3)
}

// SetIndex encodes r2[r3] <- r1
func SetIndex(r1, r2, r3 Reg) Opcode {
	return mkType2(On, r1, r2, r3)
}

// Push encodes push r1, r2
//
// r1 must contain a continuation.
func Push(r1, r2 Reg) Opcode {
	return mkType4a(On, OpId, r1, r2)
}

// PushEtc encodes pushetc r1, ...r2
//
// r1 must contain a continuation, r2 an etc.
func PushEtc(r1, r2 Reg) Opcode {
	return mkType4a(On, OpEtcId, r1, r2)
}

// Jump encodes an unconditional jump
//
// jump j
func Jump(j Offset) Opcode {
	return mkType5(Off, OpJump, Reg{}, j)
}

// JumpIf encodes a conditional jump.
//
// jump j if r
func JumpIf(j Offset, r Reg) Opcode {
	return mkType5(On, OpJumpIf, r, j)
}

// JumpIfNot encodes a conditional jump.
//
// jump j if not r
func JumpIfNot(j Offset, r Reg) Opcode {
	return mkType5(Off, OpJumpIf, r, j)
}

// Call encodes call r
//
// r must contain a continuation that is ready to be called.
func Call(r Reg) Opcode {
	return mkType5(Off, OpCall, r, Offset(0))
}

// TailCall encodes tailcall r
//
// r must contain a continuation that is ready to be called.
func TailCall(r Reg) Opcode {
	return mkType5(On, OpCall, r, Offset(0))
}

// ClTrunc encodes cltrunc h
//
// It truncates the close stack to the given height.  Each value on the close
// stack which is removed should either be nil or false, or be a value with a
// "__close" metamethod, in which case this metamethod is called.  This opcode
// is introduced to support Lua 5.4's "to-be-closed" variables.
func ClTrunc(h uint16) Opcode {
	return mkType5(Off, OpClStack, Reg{}, ClStackOffset(h))
}

// ClPush encodes clpush r
//
// r should contain either nil, false, or a value with a "__close" metamethod.
// This opcode is introduced to support Lua 5.4's "to-be-closed" variables.
func ClPush(r Reg) Opcode {
	return mkType5(On, OpClStack, r, ClStackOffset(0))
}

// Upval encodes upval r1, r2
//
// r1 must contain a closure.  This appends the value of r2 to the list of
// upvalues of r1.
func Upval(r1, r2 Reg) Opcode {
	return mkType4a(Off, OpUpvalue, r1, r2)
}

// Cont encodes r1 <- cont(r2)
//
// r2 must contain a closure, r1 then contains a new continuation for that
// closure, whose next continuation is the cc.
func Cont(r1, r2 Reg) Opcode {
	return mkType4a(Off, OpCont, r1, r2)
}

// TailCont encodes r1 <- tailcont(r2)
//
// r2 must contain a closure, r1 then contains a new continuatino for that
// closure, whose next continuation is the cc's next continuation.
func TailCont(r1, r2 Reg) Opcode {
	return mkType4a(Off, OpTailCont, r1, r2)
}

// Clear encodes clear r
//
// This clears the register.  If the register contains a cell, the cell is
// removed, so this is different from r <- nil
func Clear(r Reg) Opcode {
	return mkType4b(Off, OpClear, r, 0)
}

// Receive encodes recv r
//
// recv r is the pendant of push.
func Receive(r Reg) Opcode {
	return mkType0(Off, r)
}

// ReceiveEtc encodes recv ...r
//
// accumulates pushes into r (as an Etc)
func ReceiveEtc(r Reg) Opcode {
	return mkType0(On, r)
}

// LoadEtcLookup encodes r1 <- etclookup(r2, i)
//
// loads the (i + 1)-th element of r2 (as an etc vector) into r1.
func LoadEtcLookup(r1, r2 Reg, i int) Opcode {
	return mkType6(Off, r1, r2, Index8FromInt(i))
}

// FillTable encodes fill r1, i, r2
//
// This fills the table r1 with all values from r2 (as an etc vector) starting
// from index i.
func FillTable(r1, r2 Reg, i int) Opcode {
	return mkType6(On, r1, r2, Index8FromInt(i))
}

// PrepForLoop makes sure rStart, rStep, rStop are all numbers and converts
// rStart and rStep to the same numeric type. If the for loop should already
// stop then rStart is set to nil
func PrepForLoop(rStart, rStop, rStep Reg) Opcode {
	return mkType7(Off, rStart, rStop, rStep)
}

// AdvForLoop increments rStart by rStep, making sure that it doesn't wrap
// around if it is an integer.  If it wraps around then the loop should stop. If
// the loop should stop, rStart is set to nil
func AdvForLoop(rStart, rStop, rStep Reg) Opcode {
	return mkType7(On, rStart, rStop, rStep)
}
