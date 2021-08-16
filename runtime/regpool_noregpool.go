// +build noregpool
// This version disables the register pool.
// TODO: remove it, as it is proved to be a fair amount slower.

package runtime

var (
	globalRegPool  = dummyRegPool{}
	globalCellPool = dummyCellPool{}
	globalArgsPool = dummyRegPool{}
)

type dummyValuePool struct{}

func (p dummyValuePool) get(sz int) []Value {
	return make([]Value, sz)
}

func (p dummyValuePool) release(r []Value) {
}

type dummyCellPool struct{}

func (p dummyCellPool) get(sz int) []Cell {
	return make([]Cell, sz)
}

func (p dummyCellPool) release(c []Cell) {
}
