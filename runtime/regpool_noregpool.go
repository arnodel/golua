// +build noregpool
// This version disables the register pool

package runtime

var globalRegPool = dummyRegPool{}

// Dummy register pool
type dummyRegPool struct{}

func (p dummyRegPool) getRegs(sz int) []Value {
	return make([]Value, sz)
}

func (p dummyRegPool) getCells(sz int) []Cell {
	return make([]Cell, sz)
}

func (p dummyRegPool) releaseRegs(r []Value) {
}

func (p dummyRegPool) releaseCells(c []Cell) {
}
