//go:build noregpool
// +build noregpool

// This version disables the register pool.
// TODO: remove it, as it is proved to be a fair amount slower.

package runtime

type valuePool struct{}

func mkValuePool(size, maxAge uint) valuePool {
	return valuePool{}
}
func (p valuePool) get(sz int) []Value {
	return make([]Value, sz)
}

func (p valuePool) release(r []Value) {
}

type cellPool struct{}

func mkCellPool(size, maxAge uint) cellPool {
	return cellPool{}
}

func (p cellPool) get(sz int) []Cell {
	return make([]Cell, sz)
}

func (p cellPool) release(c []Cell) {
}
