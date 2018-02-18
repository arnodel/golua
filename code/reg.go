package code

// RegType represents the type of a register
type RegType uint8

// Register types
const (
	Register RegType = 0
	Upvalue          = 1
)

// Reg is a register
type Reg uint16

func MkRegister(idx uint8) Reg {
	return Reg(uint16(Register)<<8 | uint16(idx))
}

func MkUpvalue(idx uint8) Reg {
	return Reg(uint16(Upvalue)<<8 | uint16(idx))
}

func (r Reg) Idx() uint8 {
	return uint8(r)
}

func (r Reg) Tp() RegType {
	return RegType(uint8(r >> 8))
}

func (r Reg) ToA() uint32 {
	return uint32(r.Idx())<<16 | uint32(r.Tp())<<26
}

func (r Reg) ToB() uint32 {
	return uint32(r.Idx())<<8 | uint32(r.Tp())<<25
}

func (r Reg) ToC() uint32 {
	return uint32(r.Idx()) | uint32(r.Tp())<<24
}
