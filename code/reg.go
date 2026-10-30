package code

import "fmt"

// RegType is the kind of a register: value or cell.
type RegType uint8

// The two kinds of registers.  A value register holds a Value; a cell register
// holds a reference to a Value that a closure and its enclosing function
// share.
const (
	ValueRegType RegType = 0
	CellRegType          = 1
)

// Reg names a register of a frame: its kind and its 8-bit index.
type Reg struct {
	tp  RegType
	idx uint8
}

// ValueReg returns a value register.
func ValueReg(idx uint8) Reg {
	return Reg{
		idx: idx,
		tp:  ValueRegType,
	}
}

// CellReg returns a cell register.
func CellReg(idx uint8) Reg {
	return Reg{
		idx: idx,
		tp:  CellRegType,
	}
}

// Idx returns the index of the register
func (r Reg) Idx() uint8 {
	return r.idx
}

// RegType returns the type of the register
func (r Reg) RegType() RegType {
	return r.tp
}

// IsCell returns true if r is a cell.
func (r Reg) IsCell() bool {
	return r.tp == CellRegType
}

// toA, toB and toC encode r at the A, B and C positions of an opcode.
func (r Reg) toA() Opcode {
	return Opcode(r.Idx())<<16 | Opcode(r.RegType())<<26
}

func (r Reg) toB() Opcode {
	return Opcode(r.Idx())<<8 | Opcode(r.RegType())<<25
}

func (r Reg) toC() Opcode {
	return Opcode(r.Idx()) | Opcode(r.RegType())<<24
}

func (r Reg) String() string {
	switch r.RegType() {
	case ValueRegType:
		return fmt.Sprintf("r%d", r.Idx())
	case CellRegType:
		return fmt.Sprintf("u%d", r.Idx())
	}
	return "??"
}
