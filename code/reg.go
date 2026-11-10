package code

import "fmt"

// Reg names a register of a frame by its 8-bit index.  A register holds a
// Value.  A local that a closure captures lives in a box, a value of its own
// that the closure and the function share: a register holds the box, and
// only OpLoad, OpStore and OpNewBox touch its contents.
type Reg uint8

// ValueReg returns the register with index idx.
func ValueReg(idx uint8) Reg {
	return Reg(idx)
}

// Idx returns the index of the register.
func (r Reg) Idx() uint8 {
	return uint8(r)
}

// toA, toB and toC encode r at the A, B and C positions of an opcode.
func (r Reg) toA() Opcode {
	return Opcode(r) << 16
}

func (r Reg) toB() Opcode {
	return Opcode(r) << 8
}

func (r Reg) toC() Opcode {
	return Opcode(r)
}

func (r Reg) String() string {
	return fmt.Sprintf("r%d", r.Idx())
}
