package code

import "fmt"

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

func MkType6(f Flag, n uint8, rA, rB, rC Reg) Opcode {
	return Opcode(f.ToF() | uint32(n)<<28 | rA.ToA() | rB.ToB() | rC.ToC())
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

func (c Opcode) GetC() Reg {
	return Reg((c >> 16 & 0x100) | (c & 0xff))
}

func (c Opcode) GetZ() UnOp {
	return UnOp(c & 0xff)
}

func (c Opcode) GetN() uint16 {
	return uint16(c)
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

type Lit16 uint16

func (l Lit16) ToN() uint32 {
	return uint32(l)
}

type UnOp uint8

const (
	OpNeg UnOp = iota
	OpBitNot
	OpLen
	OpClosure
	OpCont
	OpId
	OpTruth // Turn operand to boolean
	OpCell  // ?
	OpNot   // Added afterwards - why did I not have it in the first place?
	OpUpvalue
	OpEtcId
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

type JumpOp uint8

const (
	OpCall JumpOp = iota
	OpJump
	OpJumpIf
	OpJumpIfForLoopDone // Extra opcode (3 registers needed)
)

func (op JumpOp) ToY() uint32 {
	return uint32(op) << 24
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
			tpl := "???"
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
		case OpK:
			tpl = fmt.Sprintf("K%d (%s)", n, d.ShortKString(n))
		case OpClosureK:
			tpl = fmt.Sprintf("clos(K%d) (%s)", n, d.ShortKString(n))
		}
		if f {
			return fmt.Sprintf("push %s, "+tpl, rA)
		}
		return fmt.Sprintf("%s <- "+tpl, rA)
	case Type5Pfx:
		rA := c.GetA()
		n := c.GetN()
		f := c.GetF()
		y := c.GetY()
		switch JumpOp(y) {
		case OpJump:
			dest := i + int(int16(n))
			return fmt.Sprintf("jump %+d (%s)", int16(n), d.GetLabel(dest))
		case OpJumpIf:
			dest := i + int(int16(n))
			not := ""
			if !f {
				not = " not"
			}
			return fmt.Sprintf("if%s %s jump %+d (%s)", not, rA, int16(n), d.GetLabel(dest))
		case OpCall:
			return fmt.Sprintf("call %s", rA)
		default:
			return "???"
		}
	default:
		return "???"
	}
}
