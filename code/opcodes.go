package code

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Type1:  1XXXXabc AAAAAAAA BBBBBBBB CCCCCCCC
//
// Binary ops

// Type2:  0111Fabc AAAAAAAA BBBBBBBB CCCCCCCC
//
// Table lookup / setting

// Type3:  0110FaYY AAAAAAAA NNNNNNNN NNNNNNNN
//
// Setting reg from constant

// Type4a: 0101Fab1 AAAAAAAA BBBBBBBB CCCCCCCC
//
// Unary ops + upvalues

// Type4b: 0101Fa00 AAAAAAAA BBBBBBBB CCCCCCCC
//
// Setting reg from constant (2)

// Type5:  0100FaYY AAAAAAAA NNNNNNNN NNNNNNNN
//
// Jump / call

// Type6:  0011Fab0 AAAAAAAA BBBBBBBB MMMMMMMM
//
// Load from etc

// Type0:  0000Fabc AAAAAAAA BBBBBBBB CCCCCCCC
//
// Receiving args

// Opcode is the type of opcodes
type Opcode uint32

const (
	Type1Pfx uint32 = 8 << 28
	Type2Pfx uint32 = 7 << 28
	Type3Pfx uint32 = 6 << 28
	Type4Pfx uint32 = 5 << 28
	Type5Pfx uint32 = 4 << 28
	Type6Pfx uint32 = 3 << 28
	Type0Pfx uint32 = 0
)

func MkType1(op BinOp, rA, rB, rC Reg) Opcode {
	return Opcode(1<<31 | rA.ToA() | rB.ToB() | rC.ToC() | op.ToX())
}

func MkType2(f Flag, rA, rB, rC Reg) Opcode {
	return Opcode(0x7<<28 | rA.ToA() | rB.ToB() | rC.ToC() | f.ToF())
}

func MkType3(f Flag, op UnOpK16, rA Reg, k Lit16) Opcode {
	return Opcode(0x6<<28 | f.ToF() | op.ToY() | rA.ToA() | k.ToN())
}

func MkType4a(f Flag, op UnOp, rA, rB Reg) Opcode {
	return Opcode(0x5<<28 | 1<<24 | f.ToF() | op.ToC() | rA.ToA() | rB.ToB())
}

func MkType4b(f Flag, op UnOpK, rA Reg, k Lit8) Opcode {
	return Opcode(0x5<<28 | f.ToF() | rA.ToA() | k.ToB() | op.ToC())
}

func MkType5(f Flag, op JumpOp, rA Reg, k Lit16) Opcode {
	return Opcode(Type5Pfx | f.ToF() | op.ToY() | rA.ToA() | k.ToN())
}

func MkType6(f Flag, rA, rB Reg, k Lit8) Opcode {
	return Opcode(Type6Pfx | f.ToF() | rA.ToA() | rB.ToB() | k.ToC())
}

func MkType0(f Flag, rA Reg) Opcode {
	return Opcode(f.ToF() | rA.ToA())
}

func (c Opcode) GetA() Reg {
	return Reg((c >> 18 & 0x100) | (c >> 16 & 0xff))
}

func (c Opcode) GetB() Reg {
	return Reg((c >> 17 & 0x100) | (c >> 8 & 0xff))
}

func (c Opcode) GetLit8() Lit8 {
	return Lit8(c >> 8)
}

func (c Opcode) GetC() Reg {
	return Reg((c >> 16 & 0x100) | (c & 0xff))
}

func (c Opcode) GetZ() UnOp {
	return UnOp(c & 0xff)
}

func (c Opcode) GetN() Lit16 {
	return Lit16(c)
}

func (c Opcode) SetN(n uint16) Opcode {
	return Opcode(uint32(c)&0xffff0000 | uint32(n))
}

func (c Opcode) GetM() uint8 {
	return uint8(c)
}

func (c Opcode) GetX() BinOp {
	return BinOp((c >> 27) & 0xf)
}

func (c Opcode) GetY() uint8 {
	return uint8((c >> 24) & 3)
}

func (c Opcode) GetF() bool {
	return c&(1<<27) != 0
}

func (c Opcode) GetType() uint8 {
	return uint8(c >> 28)
}

func (c Opcode) TypePfx() uint32 {
	return uint32(c) & 0xf0000000
}

func (c Opcode) GetR() uint8 {
	return uint8(c >> 28)
}

func (c Opcode) HasType1() bool {
	return c&(1<<31) != 0
}

func (c Opcode) HasType2or4() bool {
	return c&(5<<28) == 5<<28
}

func (c Opcode) HasSubtypeFlagSet() bool {
	return c&(1<<29) != 0
}

func (c Opcode) HasType4a() bool {
	return c&(1<<24) != 0
}

func (c Opcode) HasType6() bool {
	return c&(3<<30) == 0
}

func (c Opcode) HasType0() bool {
	return c&(0xf<<28) == 0
}

type BinOp uint8

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

func (op BinOp) ToX() uint32 {
	return uint32(op) << 27
}

type Flag uint8

const (
	On  Flag = 1
	Off Flag = 0
)

func (f Flag) ToF() uint32 {
	return uint32(f) << 27
}

type UnOpK16 uint8

const (
	OpInt16 UnOpK16 = iota
	OpK
	OpClosureK
	OpStr2
)

func (op UnOpK16) ToY() uint32 {
	return uint32(op) << 24
}

// LoadsK returns true if it loads a constant from the constant vector (not a
// literal encoded in the opcode).
func (op UnOpK16) LoadsK() bool {
	return op == OpK || op == OpClosureK
}

type Lit16 uint16

func (l Lit16) ToN() uint32 {
	return uint32(l)
}

func (l Lit16) ToInt16() int16 {
	return int16(l)
}

func (l Lit16) ToKIndex() KIndex {
	return KIndex(l)
}

func (l Lit16) ToOffset() Offset {
	return Offset(l)
}

// ToStr2 converts a Lit16 to two bytes.
func (l Lit16) ToStr2() []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, uint16(l))
	return b
}

func Lit16FromStr2(b []byte) Lit16 {
	return Lit16(binary.LittleEndian.Uint16(b))
}

type UnOp uint8

const (
	OpNeg      UnOp = iota // numerical negation
	OpBitNot               // bitwise negation
	OpLen                  // length
	OpClosure              // make a closure for the code
	OpCont                 // make a continuation for the closure
	OpTailCont             // make a "tail continuation" for the closure (its next is cc's next)
	OpId                   // identity
	OpTruth                // Turn operand to boolean
	OpCell                 // ?
	OpNot                  // Added afterwards - why did I not have it in the first place?
	OpUpvalue              // get an upvalue
	OpEtcId                // etc identity
	OpToNumber             // convert to number
)

func (op UnOp) ToC() uint32 {
	return uint32(op)
}

type UnOpK uint8

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

func (op UnOpK) ToC() uint32 {
	return uint32(op)
}

type Lit8 uint8

func (l Lit8) ToB() uint32 {
	return uint32(l) << 8
}

func (l Lit8) ToC() uint32 {
	return uint32(l)
}

// ToStr1 converts a Lit8 to a slice of 1 byte.
func (l Lit8) ToStr1() []byte {
	return []byte{byte(l)}
}

func (l Lit8) ToInt() int {
	return int(l)
}
func Lit8FromStr1(b []byte) Lit8 {
	return Lit8(b[0])
}

func Lit8FromInt(n int) Lit8 {
	if n < 0 || n > math.MaxInt8 {
		panic("n out of range")
	}
	return Lit8(n)
}

type JumpOp uint8

const (
	OpCall JumpOp = iota
	OpJump
	OpJumpIf
)

func (op JumpOp) ToY() uint32 {
	return uint32(op) << 24
}

type KIndex uint16

func (i KIndex) ToLit16() Lit16 {
	return Lit16(i)
}

func KIndexFromInt(i int) KIndex {
	if i < 0 || i > math.MaxUint16 {
		panic("constant index out of range")
	}
	return KIndex(i)
}

type Offset int16

func (d Offset) ToLit16() Lit16 {
	return Lit16(d)
}

func (c Opcode) Disassemble(d *UnitDisassembler, i int) string {
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
			switch c.GetZ() {
			case OpNeg:
				tpl = "-%s"
			case OpBitNot:
				tpl = "~%s"
			case OpLen:
				tpl = "#%s"
			case OpClosure:
				tpl = "clos(%s)"
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
			case OpCell:
				tpl = "cell(%s)"
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
		switch UnOpK(c.GetZ()) {
		case OpCC:
			k = "CC"
		case OpTable:
			k = "{}"
		case OpStr0:
			k = `""`
		case OpStr1:
			k = fmt.Sprintf("%q", c.GetLit8().ToStr1())
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
		y := c.GetY()
		// Type3
		tpl := "???"
		switch UnOpK16(y) {
		case OpInt16:
			tpl = fmt.Sprint(n)
		case OpStr2:
			tpl = fmt.Sprintf("%q", Lit16(n).ToStr2())
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
		y := c.GetY()
		switch JumpOp(y) {
		case OpJump:
			j := int(c.GetN().ToOffset())
			dest := i + j
			return fmt.Sprintf("jump %+d (%s)", j, d.GetLabel(dest))
		case OpJumpIf:
			j := int(c.GetN().ToOffset())
			dest := i + j
			not := ""
			if !f {
				not = " not"
			}
			return fmt.Sprintf("if%s %s jump %+d (%s)", not, rA, j, d.GetLabel(dest))
		case OpCall:
			return fmt.Sprintf("call %s", rA)
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
