package code

import "fmt"

// RegType represents the type of a register
type RegType uint8

// ValueRegType types
const (
	ValueRegType RegType = 0
	CellRegType          = 1
)

// Reg is a register
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

func (r Reg) toA() uint32 {
	return uint32(r.Idx())<<16 | uint32(r.RegType())<<26
}

func (r Reg) toB() uint32 {
	return uint32(r.Idx())<<8 | uint32(r.RegType())<<25
}

func (r Reg) toC() uint32 {
	return uint32(r.Idx()) | uint32(r.RegType())<<24
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
