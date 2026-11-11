package runtime

import "errors"

// Cell is a box: a reference to a Value, which a function and the closures
// that capture one of its locals share.  Example:
//
//	local x
//	local function f() return x + 1 end
//	x = 3
//	f()
//
// The variable x is an upvalue in f, so when x is set to 3 the upvalue of f
// must be set to 3.  This is achieved by having x live in a Cell, which the
// function holds in a register and f holds as an upvalue; the load, store
// and newbox opcodes are the only ones that touch its contents.
type Cell struct {
	ref *Value
}

// newCell returns a new Cell instance containing the given value.
func newCell(v Value) Cell {
	return Cell{&v}
}

// get returns the value that the cell c contains.
func (c Cell) get() Value {
	return *c.ref
}

// set sets the the value contained by c to v.
func (c Cell) set(v Value) {
	*c.ref = v
}

// BoxValue returns a Value holding the box c.  Such a value lives only in
// the registers the compiler gives boxes; it never reaches Lua code.
func BoxValue(c Cell) Value {
	return Value{iface: c}
}

var errNotABox = errors.New("register does not hold a box")
