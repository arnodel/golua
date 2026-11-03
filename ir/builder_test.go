package ir

import (
	"slices"
	"testing"
)

// Closing a scope clears the cells of its captured locals and releases its
// registers in register order, so the same source always compiles to the same
// code.
func TestPopContextOrder(t *testing.T) {
	names := []Name{"a", "b", "c", "d", "e", "f", "g", "h"}
	for round := 0; round < 20; round++ {
		c := NewCodeBuilder("test", NewConstantPool())
		c.PushContext()
		for _, name := range names {
			c.DeclareLocal(name, c.GetFreeRegister())
			c.context.getRegister(name, regHasUpvalue) // captured by a closure
		}
		start := len(c.code)
		c.PopContext()
		var cleared, released []Register
		for _, instr := range c.code[start:] {
			switch in := instr.(type) {
			case ClearReg:
				cleared = append(cleared, in.Dst)
			case ReleaseRegister:
				released = append(released, in.Reg)
			}
		}
		if len(cleared) != len(names) || !slices.IsSorted(cleared) {
			t.Fatalf("cleared %v, want all %d registers in order", cleared, len(names))
		}
		if len(released) != len(names) || !slices.IsSorted(released) {
			t.Fatalf("released %v, want all %d registers in order", released, len(names))
		}
	}
}
