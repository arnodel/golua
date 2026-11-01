package code

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Opcode is one 32-bit bytecode instruction.  The package documentation
// describes the encoding and the meaning of each type of opcode; the comments
// in this file give the bit layout of each type next to the code that encodes
// and decodes it.
type Opcode uint32

// There are 9 types of opcodes (Type0 to Type8), identified by the most
// significant 4 bits of the opcode, except that Type1 is identified by the
// most significant bit alone, leaving the next 4 bits for its operator.
const (
	Type1Pfx Opcode = 1 << 31 // 1......
	Type2Pfx Opcode = 7 << 28 // 0111...
	Type8Pfx Opcode = 1 << 28 // 0001...
	Type3Pfx Opcode = 6 << 28 // 0110...
	Type4Pfx Opcode = 5 << 28 // 0101...
	Type5Pfx Opcode = 4 << 28 // 0100...
	Type6Pfx Opcode = 3 << 28 // 0011...
	Type7Pfx Opcode = 2 << 28 // 0010...
	Type0Pfx Opcode = 0 << 28 // 0000...

	type4aFlag Opcode = 1 << 24
)

// IsPushInto reports whether c is "push r, x" for some register x.
func (c Opcode) IsPushInto(r Reg) bool {
	// Ignore x and its kind bit.
	return c&^0x0200ff00 == Type4Pfx|On.encodeF()|type4aFlag|r.toA()|OpId.encodeZ()
}

// IsCallOf reports whether c is "call r" or "tailcall r".
func (c Opcode) IsCallOf(r Reg) bool {
	// Ignore F, which says whether it is a tail call.
	return c&^On.encodeF() == Type5Pfx|OpCall.encodeJ()|r.toA()
}

// IsClearValue reports whether c is "clr r" for a value register r.
func (c Opcode) IsClearValue() bool {
	// Ignore r; the kind bit must say value register.
	return c&^0x00ff0000 == Type4Pfx|OpClear.encodeZ()
}

// TypePfx returns the type prefix of the opcode, that is its most significant
// 4 bits.  It identifies the type of every opcode but Type1, which is
// recognised with HasType1.
func (c Opcode) TypePfx() Opcode {
	return Opcode(c) & 0xf0000000
}

// HasType1 returns true if the opcode is Type1.
func (c Opcode) HasType1() bool {
	return c&(1<<31) != 0
}

// HasType4a returns true if the opcode is Type4a, assuming that it is Type4.
func (c Opcode) HasType4a() bool {
	return c&type4aFlag != 0
}

// HasType0 returns true if the opcode is Type0.
func (c Opcode) HasType0() bool {
	return c&(0xf<<28) == 0
}

// ==================================================================
// Type1:  1XXXXabc AAAAAAAA BBBBBBBB CCCCCCCC
//
// Binary operations: rA <- rB op rC.
//
//   - XXXX is the operator (a BinOp)
//   - aAAAAAAAA is the destination register rA
//   - bBBBBBBBB is the left operand register rB
//   - cCCCCCCCC is the right operand register rC

// BinOp is the operator of a Type1 opcode.
type BinOp uint8

// The binary operators.  The comparisons produce a boolean.
const (
	OpAdd BinOp = iota
	OpSub
	OpMul
	OpDiv
	OpFloorDiv
	OpMod
	OpPow
	OpBitAnd
	OpBitOr
	OpBitXor
	OpShiftL
	OpShiftR
	OpEq
	OpLt
	OpLeq
	OpConcat
)

// encodeX encodes a BinOp into an opcode.
func (op BinOp) encodeX() Opcode {
	return Opcode(op) << 27
}

// GetX decodes the BinOp from this opcode.
func (c Opcode) GetX() BinOp {
	return BinOp((c >> 27) & 0xf)
}

// mkType1 builds the opcode for rA <- op(rB, rC).
func mkType1(op BinOp, rA, rB, rC Reg) Opcode {
	return Type1Pfx | rA.toA() | rB.toB() | rC.toC() | op.encodeX()
}

// ==================================================================
// Type2:  0111Fabc AAAAAAAA BBBBBBBB CCCCCCCC
//
// Table access: rA <- rB[rC] when F is Off, rB[rC] <- rA when F is On.
//
//   - F selects between loading and storing
//   - aAAAAAAAA is the register rA (destination or source, depending on F)
//   - bBBBBBBBB is the register rB holding the table
//   - cCCCCCCCC is the register rC holding the key

// mkType2 builds the opcode for rA <- rB[rC] (f Off) or rB[rC] <- rA (f On).
func mkType2(f Flag, rA, rB, rC Reg) Opcode {
	return Type2Pfx | rA.toA() | rB.toB() | rC.toC() | f.encodeF()
}

// ==================================================================
// Type8:  0001Fab0 AAAAAAAA BBBBBBBB KKKKKKKK
//
// Table access by constant key: rA <- rB[K] when F is Off, rB[K] <- rA when F
// is On.  This is what Type2 does when the key is a constant, without the
// opcode to load the key into a register first; the compiler uses it for
// field access and only when the constant index fits in 8 bits.
//
//   - F selects between loading and storing
//   - aAAAAAAAA is the register rA (destination or source, depending on F)
//   - bBBBBBBBB is the register rB holding the table
//   - KKKKKKKK is the index of the key in the constant table (a KIndex8)

// KIndex8 is the index of a constant that fits in 8 bits (0 to 255).
type KIndex8 uint8

// encodeK8 encodes a KIndex8 into an Opcode.
func (i KIndex8) encodeK8() Opcode {
	return Opcode(i)
}

// ToKIndex converts the index to a KIndex.
func (i KIndex8) ToKIndex() KIndex {
	return KIndex(i)
}

// KIndex8FromInt returns a KIndex8 encoding the given int, which must fit in
// an uint8.
func KIndex8FromInt(i int) KIndex8 {
	if i < 0 || i > math.MaxUint8 {
		panic("constant index out of range")
	}
	return KIndex8(i)
}

// GetK8 decodes the KIndex8 from the opcode.
func (c Opcode) GetK8() KIndex8 {
	return KIndex8(c)
}

// SetK8 returns a copy of the opcode with a new KIndex8.
func (c Opcode) SetK8(i KIndex8) Opcode {
	return c&0xffffff00 | i.encodeK8()
}

// mkType8 builds the opcode for rA <- rB[K] (f Off) or rB[K] <- rA (f On).
func mkType8(f Flag, rA, rB Reg, k KIndex8) Opcode {
	return Type8Pfx | rA.toA() | rB.toB() | k.encodeK8() | f.encodeF()
}

// ==================================================================
// Type3:  0110FaYY AAAAAAAA NNNNNNNN NNNNNNNN
//
// Loading a 16-bit literal or a constant: rA <- value(N) when F is Off, or
// push the value into the continuation rA when F is On.
//
//   - F selects between storing in rA and pushing into rA
//   - aAAAAAAAA is the register rA
//   - YY says how N is read (an UnOpK16)
//   - N is a 16-bit literal or an index into the constant table

// UnOpK16 says how a Type3 opcode reads its 16-bit N field.
type UnOpK16 uint8

// The ways of reading the N field of a Type3 opcode.
const (
	OpInt16    UnOpK16 = iota // the integer int16(N)
	OpK                       // constant number N
	OpClosureK                // a new closure of the function constant number N
	OpStr2                    // the 2-byte string whose bytes are N (little-endian)
)

// encodeY encodes an UnOpK16 into an opcode.
func (op UnOpK16) encodeY() Opcode {
	return Opcode(op) << 24
}

// LoadsK returns true if it loads a constant from the constant vector (not a
// literal encoded in the opcode).
func (op UnOpK16) LoadsK() bool {
	return op == OpK || op == OpClosureK
}

type encoderToN interface {
	encodeN() Opcode
}

// Lit16 is a 16 bit literal used in several opcode types, used to represent
// constant values, offsets or indexes into the constants table.
type Lit16 uint16

// encodeN encodes l into an opcode at the N position.
func (l Lit16) encodeN() Opcode {
	return Opcode(l)
}

// ToInt16 converts l to an int16
func (l Lit16) ToInt16() int16 {
	return int16(l)
}

// ToStr2 converts a Lit16 to a slice of two bytes.
func (l Lit16) ToStr2() []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, uint16(l))
	return b
}

// ToKIndex converts a Lit16 into a KIndex.
func (l Lit16) ToKIndex() KIndex {
	return KIndex(l)
}

// Lit16FromStr2 converts a slice into a Lit16.  The slice must have length 2.
func Lit16FromStr2(b []byte) Lit16 {
	return Lit16(binary.LittleEndian.Uint16(b))
}

// Lit16FromInt16 converts an int16 into a Lit16.
func Lit16FromInt16(n int16) Lit16 {
	return Lit16(n)
}

// KIndex is an index into the constant table.
type KIndex uint16

// encodeN encodes i into an opcode at the N position.
func (i KIndex) encodeN() Opcode {
	return Opcode(i)
}

// KIndexFromInt returns a KIndex encoding the given index i, panicking if out
// of range.
func KIndexFromInt(i int) KIndex {
	if i < 0 || i > math.MaxUint16 {
		panic("constant index out of range")
	}
	return KIndex(i)
}

// SetKIndex returns a copy of the opcode with a new KIndex.
func (c Opcode) SetKIndex(i KIndex) Opcode {
	return c&0xffff0000 | i.encodeN()
}

// GetKIndex decodes the KIndex from the opcode.
func (c Opcode) GetKIndex() KIndex {
	return KIndex(c)
}

// GetN decodes the Lit16 from the opcode.
func (c Opcode) GetN() Lit16 {
	return Lit16(c)
}

// GetY decodes the UnOpK16 from the opcode.
func (c Opcode) GetY() UnOpK16 {
	return UnOpK16(c.getYorJ())
}

// mkType3 builds a Type3 opcode from its constituents.
func mkType3(f Flag, op UnOpK16, rA Reg, k encoderToN) Opcode {
	return Type3Pfx | f.encodeF() | op.encodeY() | rA.toA() | k.encodeN()
}

// ==================================================================
// Type4a: 0101Fab1 AAAAAAAA BBBBBBBB ZZZZZZZZ
//
// Unary operations and continuations: rA <- op(rB) when F is Off, or push
// op(rB) into the continuation rA when F is On.  A few operations (OpEtcId,
// OpUpvalue, OpCheckNotDefined) act on rA and rB in their own way, see the
// package documentation.
//
//   - F selects between storing in rA and pushing into rA
//   - aAAAAAAAA is the register rA
//   - bBBBBBBBB is the operand register rB
//   - bit 24 is 1 (0 means Type4b)
//   - ZZZZZZZZ is the operation (an UnOp)

// UnOp is the operation of a Type4a opcode.
type UnOp uint8

// The Type4a operations.
const (
	OpNeg             UnOp = iota // -rB
	OpBitNot                      // ~rB
	OpLen                         // #rB
	OpCont                        // a continuation for the closure rB, returning to the current frame
	OpTailCont                    // a continuation for the closure rB, returning where the current frame returns
	OpId                          // rB itself; with F On this is "push rA, rB"
	OpTruth                       // rB as a boolean
	OpNot                         // not rB
	OpUpvalue                     // add the cell uB to the upvalues of the closure rA
	OpEtcId                       // push all the values of the etc rB into the continuation rA
	OpMkVarargTable               // a table sharing the values of the etc rB, which also replaces rB (Lua 5.5)
	OpCheckNotDefined             // error if rA[rB] is not nil (Lua 5.5 global redefinition check)
)

// encodeZ encodes an UnOp into an opcode.
func (op UnOp) encodeZ() Opcode {
	return Opcode(op)
}

// GetUnOp decodes the UnOp in the opcode.
func (c Opcode) GetUnOp() UnOp {
	return UnOp(c & 0xff)
}

// mkType4a builds a Type4a opcode from its constituents.
func mkType4a(f Flag, op UnOp, rA, rB Reg) Opcode {
	return Type4Pfx | type4aFlag | f.encodeF() | op.encodeZ() | rA.toA() | rB.toB()
}

// ==================================================================
// Type4b: 0101Fa00 AAAAAAAA LLLLLLLL ZZZZZZZZ
//
// Loading a small constant: rA <- value when F is Off, or push the value into
// the continuation rA when F is On.
//
//   - F selects between storing in rA and pushing into rA
//   - aAAAAAAAA is the register rA
//   - bits 25 and 24 are 0 (bit 24 set means Type4a)
//   - LLLLLLLL is an 8-bit payload for the constants that need one
//   - ZZZZZZZZ says which constant (an UnOpK)

// UnOpK says which constant a Type4b opcode loads.
type UnOpK uint8

// The Type4b constants.
const (
	OpNil   UnOpK = iota // nil
	OpStr0               // ""
	OpTable              // a new empty table
	OpStr1               // the 1-byte string L
	OpBool               // the boolean L != 0
	OpCC                 // the current continuation
	OpClear              // not a load: resets rA, giving a cell register a fresh cell
	OpInt                // unused: meant to carry an int64 in the next 2 words
	OpFloat              // unused: meant to carry a float64 in the next 2 words
	OpStrN               // unused: meant to carry a string in the following words
)

// encodeZ encodes an UnOpK into an opcode.
func (op UnOpK) encodeZ() Opcode {
	return Opcode(op)
}

// Lit8 is the 8-bit payload of a Type4b opcode.
type Lit8 uint8

// encodeL encodes l into an opcode at the L position.
func (l Lit8) encodeL() Opcode {
	return Opcode(l) << 8
}

// ToStr1 converts a Lit8 to a slice of 1 byte.
func (l Lit8) ToStr1() []byte {
	return []byte{byte(l)}
}

// ToBool converts a Lit8 to a boolean.
func (l Lit8) ToBool() bool {
	return l != 0
}

// Lit8FromStr1 encodes a Lit8 from a byte string of length 1.  Panics if b has
// length 0.
func Lit8FromStr1(b []byte) Lit8 {
	return Lit8(b[0])
}

// Lit8FromBool encodes a Lit8 from a bool.
func Lit8FromBool(b bool) Lit8 {
	if b {
		return 1
	}
	return 0
}

// GetL decodes the L field of the opcode.
func (c Opcode) GetL() Lit8 {
	return Lit8(c >> 8)
}

// GetUnOpK decodes the UnOpK in the opcode.
func (c Opcode) GetUnOpK() UnOpK {
	return UnOpK(c & 0xff)
}

// mkType4b builds a Type4b opcode from its constituents.
func mkType4b(f Flag, op UnOpK, rA Reg, k Lit8) Opcode {
	return Type4Pfx | f.encodeF() | rA.toA() | k.encodeL() | op.encodeZ()
}

// ==================================================================
// Type5:  0100FaJJ AAAAAAAA DDDDDDDD DDDDDDDD
//
// Jumps, calls and the close stack.
//
//   - F: for OpJumpIf, jump on a truthy (On) or falsy (Off) rA; for OpCall,
//     On makes it a tail call; for OpClStack, On pushes rA and Off truncates
//   - aAAAAAAAA is the register rA (unused by OpJump and the truncation)
//   - JJ is the operation (a JumpOp)
//   - D is a signed 16-bit offset from the jump itself (OpJump, OpJumpIf) or
//     a height of the close stack (OpClStack truncation); 0 for OpCall

// JumpOp is the operation of a Type5 opcode.
type JumpOp uint8

// The Type5 operations.
const (
	OpCall    JumpOp = iota // transfer control to the continuation rA; "call r0" is a return
	OpJump                  // pc <- pc + D
	OpJumpIf                // pc <- pc + D if rA is truthy (F On) or falsy (F Off)
	OpClStack               // push rA onto the close stack (F On) or truncate it to height D (F Off)
)

// encodeJ encodes a jump type into and opcode.
func (op JumpOp) encodeJ() Opcode {
	return Opcode(op) << 24
}

// An Offset is the distance of a jump, counted from the jump opcode itself.
type Offset int16

// encodeD encodes d into an opcode at the D position.
func (d Offset) encodeD() Opcode {
	return Opcode(uint16(d))
}

// ClStackOffset is a height of the close stack, counted from its bottom.
type ClStackOffset uint16

// encodeD encodes d into an opcode at the D position.
func (d ClStackOffset) encodeD() Opcode {
	return Opcode(d)
}

// encoderToD is implemented by the types that can fill the D field.
type encoderToD interface {
	encodeD() Opcode
}

// GetJ decodes the JumpOp from this opcode.
func (c Opcode) GetJ() JumpOp {
	return JumpOp(c.getYorJ())
}

// GetOffset decodes the Offset from the opcode.
func (c Opcode) GetOffset() Offset {
	return Offset(uint16(c))
}

// GetClStackOffset decodes the ClStackOffset from the opcode.
func (c Opcode) GetClStackOffset() ClStackOffset {
	return ClStackOffset(uint16(c))
}

// SetOffset returns a copy of the opcode with the given offset.
func (c Opcode) SetOffset(n Offset) Opcode {
	return c&0xffff0000 | n.encodeD()
}

// mkType5 builds a Type5 opcode from its constituents.
func mkType5(f Flag, op JumpOp, rA Reg, k encoderToD) Opcode {
	return Type5Pfx | f.encodeF() | op.encodeJ() | rA.toA() | k.encodeD()
}

// ==================================================================
// Type6:  0011Fab0 AAAAAAAA BBBBBBBB MMMMMMMM
//
// Reading an etc: rA <- the M-th value of the etc rB when F is Off (nil if
// there is none); rA[M+i] <- the i-th value of the etc rB for every i when F
// is On.
//
//   - F selects between loading one value and filling a table
//   - aAAAAAAAA is the register rA
//   - bBBBBBBBB is the register rB holding the etc
//   - MMMMMMMM is an index (an Index8), counted from 0

// Index8 is an 8-bit index (0 to 255).
type Index8 uint8

// encodeM encodes an Index8 into an Opcode.
func (i Index8) encodeM() Opcode {
	return Opcode(i)
}

// Index8FromInt returns an Index8 encoding the given int, which must fit in an
// uint8.
func Index8FromInt(n int) Index8 {
	if n < 0 || n > math.MaxUint8 {
		panic("n out of range")
	}
	return Index8(n)
}

// GetM decodes the Index8 from the opcode.
func (c Opcode) GetM() Index8 {
	return Index8(c)
}

// mkType6 builds a Type6 opcode from its constituents.
func mkType6(f Flag, rA, rB Reg, i Index8) Opcode {
	return Type6Pfx | f.encodeF() | rA.toA() | rB.toB() | i.encodeM()
}

// ==================================================================
// Type7:  0010Fabc AAAAAAAA BBBBBBBB CCCCCCCC
//
// Numeric for loops, over the variable rA with limit rB and step rC.  F Off
// prepares the loop (checks the values are numbers, gives rA and rC the same
// numeric type, sets rA to nil if the loop must not run); F On advances it
// (rA <- rA + rC, then sets rA to nil if the limit is passed or the addition
// wrapped around).  A conditional jump on rA follows.
//
//   - F selects between preparing and advancing
//   - aAAAAAAAA is the loop variable rA
//   - bBBBBBBBB is the limit rB
//   - cCCCCCCCC is the step rC

// mkType7 builds a Type7 opcode from its constituents.
func mkType7(f Flag, rA, rB, rC Reg) Opcode {
	return Type7Pfx | f.encodeF() | rA.toA() | rB.toB() | rC.toC()
}

// ==================================================================
// Type0:  0000Fa00 AAAAAAAA 00000000 00000000
//
// Receiving arguments: the next value pushed into this continuation goes to
// rA when F is Off; all the remaining pushed values go to rA as an etc when F
// is On.  When the continuation runs, a Type0 opcode that has not received a
// value sets rA to nil (or to an empty etc).
//
//   - F selects between receiving one value and receiving the rest
//   - aAAAAAAAA is the register rA

// mkType0 builds a Type0 opcode from its constituents.
func mkType0(f Flag, rA Reg) Opcode {
	return f.encodeF() | rA.toA()
}

// ==================================================================
// Fields shared by several types of opcodes.

// Flag is the On / Off switch at bit 27 of every opcode but Type1.  Its
// meaning depends on the type.
type Flag uint8

// The two values of a Flag.
const (
	On  Flag = 1
	Off Flag = 0
)

// encodeF encodes the flag into an opcode at the F position.
func (f Flag) encodeF() Opcode {
	return Opcode(f) << 27
}

// GetF decodes the flag and returns true if the flag is On.
func (c Opcode) GetF() bool {
	return c&(1<<27) != 0
}

// Register operands.  Each is an 8-bit index in one of the three low bytes
// of the opcode, with a kind bit (0 value register, 1 cell register) at bit
// 26 (A), 25 (B) or 24 (C).

// GetA returns the register rA encoded in the opcode.
func (c Opcode) GetA() Reg {
	return Reg{
		idx: uint8(c >> 16 & 0xff),
		tp:  RegType(c >> 26 & 1),
	}
}

// GetB returns the register rB encoded in the opcode.
func (c Opcode) GetB() Reg {
	return Reg{
		idx: uint8(c >> 8 & 0xff),
		tp:  RegType(c >> 25 & 1),
	}
}

// GetC returns the register rC encoded in the opcode.
func (c Opcode) GetC() Reg {
	return Reg{
		idx: uint8(c & 0xff),
		tp:  RegType(c >> 24 & 1),
	}
}

// getYorJ decodes the 2-bit field at bits 25-24, which is Y in Type3 and J in
// Type5.
func (c Opcode) getYorJ() uint8 {
	return uint8((c >> 24) & 3)
}

// OpcodeDisassembler is an interface that helps disassemble an opcode.
type OpcodeDisassembler interface {
	ShortKString(KIndex) string // Gets a string representation of a constant
	GetLabel(int) string        // Gets a constistent label name for a code offset
}

// Disassemble gets a human readable representation of an opcode.
func (c Opcode) Disassemble(d OpcodeDisassembler, i int) string {
	if c.HasType1() {
		// Type1
		rA := c.GetA()
		rB := c.GetB()
		rC := c.GetC()
		tpl := "???"
		switch c.GetX() {
		case OpAdd:
			tpl = "%s + %s"
		case OpSub:
			tpl = "%s - %s"
		case OpMul:
			tpl = "%s * %s"
		case OpDiv:
			tpl = "%s / %s"
		case OpFloorDiv:
			tpl = "%s floor/ %s"
		case OpMod:
			tpl = "%s mod %s"
		case OpPow:
			tpl = "%s ^ %s"
		case OpBitAnd:
			tpl = "%s & %s"
		case OpBitOr:
			tpl = "%s | %s"
		case OpBitXor:
			tpl = "%s ~ %s"
		case OpShiftL:
			tpl = "%s << %s"
		case OpShiftR:
			tpl = "%s >> %s"
		case OpEq:
			tpl = "%s == %s"
		case OpLt:
			tpl = "%s < %s"
		case OpLeq:
			tpl = "%s <= %s"
		case OpConcat:
			tpl = "%s .. %s"
		}
		return fmt.Sprintf("%s <- "+tpl, rA, rB, rC)
	}
	switch c.TypePfx() {
	case Type2Pfx:
		rA := c.GetA()
		f := c.GetF()
		rB := c.GetB()
		rC := c.GetC()
		if !f {
			return fmt.Sprintf("%s <- %s[%s]", rA, rB, rC)
		}
		return fmt.Sprintf("%s[%s] <- %s", rB, rC, rA)
	case Type8Pfx:
		rA := c.GetA()
		f := c.GetF()
		rB := c.GetB()
		k := c.GetK8()
		key := fmt.Sprintf("K%d (%s)", k, d.ShortKString(k.ToKIndex()))
		if !f {
			return fmt.Sprintf("%s <- %s[%s]", rA, rB, key)
		}
		return fmt.Sprintf("%s[%s] <- %s", rB, key, rA)
	case Type4Pfx:
		rA := c.GetA()
		f := c.GetF()
		if c.HasType4a() {
			rB := c.GetB()
			// Type4a
			tpl := "??? %s"
			switch c.GetUnOp() {
			case OpNeg:
				tpl = "-%s"
			case OpBitNot:
				tpl = "~%s"
			case OpLen:
				tpl = "#%s"
			case OpCont:
				tpl = "cont(%s)"
			case OpTailCont:
				tpl = "tailcont(%s)"
			case OpId:
				tpl = "%s"
			case OpEtcId:
				tpl = "...%s"
			case OpTruth:
				tpl = "bool(%s)"
			case OpNot:
				tpl = "not %s"
			case OpUpvalue:
				// Special case
				return fmt.Sprintf("upval %s, %s", rA, rB)
			case OpMkVarargTable:
				tpl = "mkvargtable(%s)"
			case OpCheckNotDefined:
				return fmt.Sprintf("checknotdef %s[%s]", rA, rB)
			}
			if f {
				// It's a push
				return fmt.Sprintf("push %s, "+tpl, rA, rB)
			}
			return fmt.Sprintf("%s <- "+tpl, rA, rB)
		}
		k := "??"
		switch c.GetUnOpK() {
		case OpCC:
			k = "CC"
		case OpTable:
			k = "{}"
		case OpStr0:
			k = `""`
		case OpStr1:
			k = fmt.Sprintf("%q", c.GetL().ToStr1())
		case OpBool:
			k = fmt.Sprintf("%t", c.GetL().ToBool())
		case OpNil:
			k = "nil"
		case OpClear:
			// Special case
			return fmt.Sprintf("clr %s", rA)
		}
		if f {
			return fmt.Sprintf("push %s, "+k, rA)
		}
		return fmt.Sprintf("%s <- "+k, rA)
	case Type0Pfx:
		rA := c.GetA()
		if c.GetF() {
			return "recv ..." + rA.String()
		}
		return "recv " + rA.String()
	case Type3Pfx:
		rA := c.GetA()
		n := c.GetN()
		f := c.GetF()
		// Type3
		tpl := "???"
		switch c.GetY() {
		case OpInt16:
			tpl = fmt.Sprint(n)
		case OpStr2:
			tpl = fmt.Sprintf("%q", n.ToStr2())
		case OpK:
			tpl = fmt.Sprintf("K%d (%s)", n, d.ShortKString(n.ToKIndex()))
		case OpClosureK:
			tpl = fmt.Sprintf("clos(K%d) (%s)", n, d.ShortKString(n.ToKIndex()))
		}
		if f {
			return fmt.Sprintf("push %s, "+tpl, rA)
		}
		return fmt.Sprintf("%s <- "+tpl, rA)
	case Type5Pfx:
		rA := c.GetA()
		f := c.GetF()
		switch c.GetJ() {
		case OpJump:
			j := int(c.GetOffset())
			dest := i + j
			return fmt.Sprintf("jump %+d (%s)", j, d.GetLabel(dest))
		case OpJumpIf:
			j := int(c.GetOffset())
			dest := i + j
			not := ""
			if !f {
				not = " not"
			}
			return fmt.Sprintf("if%s %s jump %+d (%s)", not, rA, j, d.GetLabel(dest))
		case OpCall:
			instr := "call"
			if c.GetF() {
				instr = "tailcall"
			}
			return fmt.Sprintf("%s %s", instr, rA)
		case OpClStack:
			if c.GetF() {
				return fmt.Sprintf("clpush %s", rA)
			} else {
				return fmt.Sprintf("cltrunc %d", c.GetClStackOffset())
			}
		default:
			return "???"
		}
	case Type6Pfx:
		rA := c.GetA()
		rB := c.GetB()
		f := c.GetF()
		m := c.GetM()
		if f {
			return fmt.Sprintf("fill %s, %d, %s", rA, m, rB)
		}
		return fmt.Sprintf("%s <- etclookup(%s, %d)", rA, rB, m)
	case Type7Pfx:
		rStart, rStop, rStep := c.GetA(), c.GetB(), c.GetC()
		action := "prep"
		if c.GetF() {
			action = "adv"
		}
		return fmt.Sprintf("%sfor %s, %s, %s", action, rStart, rStop, rStep)
	default:
		return "???"
	}
}
