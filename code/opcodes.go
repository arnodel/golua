package code

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Opcode is the type of opcodes
type Opcode uint32

// There are 7 types of opcodes (Typ0 - Type7).  The type of opcode is defined
// by the most significant 4 bits of the opcode.

// Prefixes for the different types of opcodes.  Note: there is an unused prefix
// (0001).
const (
	Type1Pfx Opcode = 1 << 31 // 1......
	Type2Pfx Opcode = 7 << 28 // 0111...
	Type3Pfx Opcode = 6 << 28 // 0110...
	Type4Pfx Opcode = 5 << 28 // 0101...
	Type5Pfx Opcode = 4 << 28 // 0100...
	Type6Pfx Opcode = 3 << 28 // 0011...
	Type0Pfx Opcode = 0 << 28 // 0000...

	type4aFlag Opcode = 1 << 24
)

// TypePfx returns the type prefix of the opcode. (valid for all types apart
// from Type1, which has MSB 1).
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

// HasType0 returns true if the opcdoe is Type0.
func (c Opcode) HasType0() bool {
	return c&(0xf<<28) == 0
}

// ==================================================================
// Type1:  1XXXXabc AAAAAAAA BBBBBBBB CCCCCCCC
//
// Opcodes for binary operations.
// - XXXX encodes the operator op
// - aAAAAAAAA encodes the destination register rA
// - bBBBBBBBB encodes the left operand register rB
// - cCCCCCCCC encodes the right operand register rC

// BinOp is the type of binary operator available int Type1 opcodes.
type BinOp uint8

// Here is the list of available binary operators.
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

// This functions builds the opcode for
//    rA <- op(rB, rC)
func mkType1(op BinOp, rA, rB, rC Reg) Opcode {
	return Type1Pfx | rA.toA() | rB.toB() | rC.toC() | op.encodeX()
}

// ==================================================================
// Type2:  0111Fabc AAAAAAAA BBBBBBBB CCCCCCCC
//
// Opcodes for table lookup / setting
// - F encodes the flag f (On / Off)
// - aAAAAAAAA encodes the register rA (source or destination depending on f)
// - bBBBBBBBB encodes the register holding the table rB
// - cCCCCCCCC encodes the register holding the index rC

// This builds the opcode for
//     rA <- rB[rC]  if f is Off
//     rB[rC] <- rA  if f is On
func mkType2(f Flag, rA, rB, rC Reg) Opcode {
	return Type2Pfx | rA.toA() | rB.toB() | rC.toC() | f.encodeF()
}

// ==================================================================
// Type3:  0110FaYY AAAAAAAA NNNNNNNN NNNNNNNN
//
// Setting reg from constant

// UnOpK16 is the type of operator available in Type3 opcodes.
type UnOpK16 uint8

// Here is the list of available UnOpK16 operators.
const (
	OpInt16 UnOpK16 = iota
	OpK
	OpClosureK
	OpStr2
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

// ToN encodes l into an opcode at N position.
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

// KIndex is an index into the constants table
type KIndex uint16

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

// Build a Type3 opcode from its constituents.
func mkType3(f Flag, op UnOpK16, rA Reg, k encoderToN) Opcode {
	return Type3Pfx | f.encodeF() | op.encodeY() | rA.toA() | k.encodeN()
}

// ==================================================================
// Type4a: 0101Fab1 AAAAAAAA BBBBBBBB ZZZZZZZZ
//
// Unary ops + upvalues

// UnOp is the type of operators available in Type4a opcodes.
type UnOp uint8

// Available unary operators
const (
	OpNeg      UnOp = iota // numerical negation
	OpBitNot               // bitwise negation
	OpLen                  // length
	OpCont                 // make a continuation for the closure
	OpTailCont             // make a "tail continuation" for the closure (its next is cc's next)
	OpId                   // identity
	OpTruth                // Turn operand to boolean
	OpNot                  // Added afterwards - why did I not have it in the first place?
	OpUpvalue              // get an upvalue
	OpEtcId                // etc identity
	OpToNumber             // convert to number
)

// encodeZ enocodes an UnOp into an opcode.
func (op UnOp) encodeZ() Opcode {
	return Opcode(op)
}

// GetUnOp decodes the UnOp in the opcode.
func (c Opcode) GetUnOp() UnOp {
	return UnOp(c & 0xff)
}

func mkType4a(f Flag, op UnOp, rA, rB Reg) Opcode {
	return Type4Pfx | type4aFlag | f.encodeF() | op.encodeZ() | rA.toA() | rB.toB()
}

// ==================================================================
// Type4b: 0101Fa00 AAAAAAAA LLLLLLLL ZZZZZZZZ
//
// Setting reg from constant (2)

// UnOpK is the type of operators available in Type4b opcodes.
type UnOpK uint8

// Available Constant operators
const (
	OpNil UnOpK = iota
	OpStr0
	OpTable
	OpStr1
	OpBool
	OpCC
	OpClear
	OpInt   // Extra 64 bits (2 opcodes)
	OpFloat // Extra 64 bits (2 opcodes)
	OpStrN  // Extra [n / 4] opcodes
)

// encodeZ encodes an UnOpK into an opcode.
func (op UnOpK) encodeZ() Opcode {
	return Opcode(op)
}

// Lit8 is an 8 bit literal that can encode different types of constants inline.
type Lit8 uint8

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

// Build a Type4b opcode from its constituents
func mkType4b(f Flag, op UnOpK, rA Reg, k Lit8) Opcode {
	return Type4Pfx | f.encodeF() | rA.toA() | k.encodeL() | op.encodeZ()
}

// ==================================================================
// Type5:  0100FaJJ AAAAAAAA DDDDDDDD DDDDDDDD
//
// Jump / call

// JumpOp is the type of jump supported by Type5 opcode.
type JumpOp uint8

// Here are the types of jumps
const (
	OpCall JumpOp = iota
	OpJump
	OpJumpIf
	OpClStack // Operate on the Close Stack
)

// encodeJ encodes a jump type into and opcode.
func (op JumpOp) encodeJ() Opcode {
	return Opcode(op) << 24
}

// An Offset is a relative position in the code for jumping to.
type Offset int16

func (d Offset) encodeD() Opcode {
	return Opcode(uint16(d))
}

type ClStackOffset uint16

func (d ClStackOffset) encodeD() Opcode {
	return Opcode(d)
}

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

func (c Opcode) GetClStackOffset() ClStackOffset {
	return ClStackOffset(uint16(c))
}

// SetOffset returns a copy of the opcode with the given offset.
func (c Opcode) SetOffset(n Offset) Opcode {
	return c&0xffff0000 | n.encodeD()
}

func mkType5(f Flag, op JumpOp, rA Reg, k encoderToD) Opcode {
	return Type5Pfx | f.encodeF() | op.encodeJ() | rA.toA() | k.encodeD()
}

// ==================================================================
// Type6:  0011Fab0 AAAAAAAA BBBBBBBB MMMMMMMM
//
// Load from etc

// Index8 is an 8 bit index (0 - 255).
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

func mkType6(f Flag, rA, rB Reg, i Index8) Opcode {
	return Type6Pfx | f.encodeF() | rA.toA() | rB.toB() | i.encodeM()
}

// ==================================================================
// Type0:  0000Fabc AAAAAAAA BBBBBBBB CCCCCCCC
//
// Receiving args

func mkType0(f Flag, rA Reg) Opcode {
	return f.encodeF() | rA.toA()
}

// ==================================================================
// Types that are used in several opcodes.

// Flag is an On / Off switch that can be used in several types of opcodes.
type Flag uint8

// Flags have two values: On or Off
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

//
// Methods on opcodes to registers.
//

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
			case OpToNumber:
				tpl = "tonumber(%s)"
			case OpUpvalue:
				// Special case
				return fmt.Sprintf("upval %s, %s", rA, rB)
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
	default:
		return "???"
	}
}
