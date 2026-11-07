package ir

import (
	"slices"
	"testing"
)

// Closing a scope clears the cells of its captured locals in register order,
// so the same source always compiles to the same code.
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
		var cleared []Register
		for _, instr := range c.code[start:] {
			if in, ok := instr.(ClearReg); ok {
				cleared = append(cleared, in.Dst)
			}
		}
		if len(cleared) != len(names) || !slices.IsSorted(cleared) {
			t.Fatalf("cleared %v, want all %d registers in order", cleared, len(names))
		}
	}
}
