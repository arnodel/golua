//go:build !noquotas

package runtime

import "github.com/arnodel/golua/code"

// The run loop reads and writes registers, and reads the words of a
// function's code, its constants and its lookup hints, through the accessors
// below.  Code.verify has established, before a function can run, that none
// of these can be out of bounds, and the build without quotas relies on it
// to index without a check (access_noquotas.go).  This build is the one that
// runs code it does not trust, and its speed comes second: it keeps Go's
// bounds checks as a second line of defence, so that a mistake in the
// verifier is a panic rather than a read or a write outside a frame.

// setReg and getReg access a register.  A box's contents are reached only
// through the load and store opcodes (ir.LowerCells).
func setReg(regs []Value, reg code.Reg, val Value) {
	regs[reg] = val
}

func getReg(regs []Value, reg code.Reg) Value {
	return regs[reg]
}

// opcodeAt, constAt and hintAt read the word at pc of a function's code, a
// constant, and the lookup hint of the opcode at pc.
func opcodeAt(opcodes []code.Opcode, pc int32) code.Opcode {
	return opcodes[pc]
}

func constAt(consts []Value, k uintptr) Value {
	return consts[k]
}

func hintAt(hints []lookupHint, pc int32) *lookupHint {
	return &hints[pc]
}
