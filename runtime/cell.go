package runtime

// Cell is the data structure that represents a reference to a value.  Whenever
// a value is put into a register that contains a cell, it is put in the cell
// rather than the register itself.  It is used in order to implement upvalues
// in lua. Example:
//
//     local x
//     local function f() return x + 1 end
//     x = 3
//     f()
//
// The variable x is an upvalue in f so when x is set to 3 the upvalue of f must
// be set to 3.  This is achieved by setting the register that contains x to a
// Cell.
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
