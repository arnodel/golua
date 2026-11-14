//go:build noquotas

package runtime

import (
	"unsafe"

	"github.com/arnodel/golua/code"
)

// The run loop reads and writes registers, and reads the words of a
// function's code, its constants and its lookup hints, on every opcode.
// Without quotas, the accessors below index without a bounds check, and rely
// on what Code.verify has established before the function could run:
//
//   - every register an opcode names (or lists as an argument of a call) is
//     within the register count that NewLuaCont sizes the frame from;
//   - execution never leaves the code: every jump lands inside it, and
//     wherever execution goes on to after an opcode is there;
//   - every constant an opcode names is in the constant table;
//   - there is a lookup hint for every word of code (the two ways of loading
//     a function make Code.slotCache as long as its code).
//
// The build with quotas, which is the one that runs code it does not trust,
// keeps the checks (luacont_access.go).

// setReg and getReg access a register.  A box's contents are reached only
// through the load and store opcodes (ir.LowerCells).
func setReg(regs []Value, reg code.Reg, val Value) {
	*(*Value)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(regs)), uintptr(reg)*unsafe.Sizeof(Value{}))) = val
}

func getReg(regs []Value, reg code.Reg) Value {
	return *(*Value)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(regs)), uintptr(reg)*unsafe.Sizeof(Value{})))
}

// opcodeAt, constAt and hintAt read the word at pc of a function's code, a
// constant, and the lookup hint of the opcode at pc.
func opcodeAt(opcodes []code.Opcode, pc int32) code.Opcode {
	return *(*code.Opcode)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(opcodes)), uintptr(pc)*unsafe.Sizeof(code.Opcode(0))))
}

func constAt(consts []Value, k uintptr) Value {
	return *(*Value)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(consts)), k*unsafe.Sizeof(Value{})))
}

func hintAt(hints []lookupHint, pc int32) *lookupHint {
	return (*lookupHint)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(hints)), uintptr(pc)*unsafe.Sizeof(lookupHint{})))
}
