//go:build nocontpool || !noquotas
// +build nocontpool !noquotas

// This tag disables the continuation pool

package runtime

// Dummy pool.
type goContPool struct{}

// Same as new(GoCont).
func (p goContPool) get() *GoCont {
	return new(GoCont)
}

// Does nothing.
func (p goContPool) release(cont *GoCont) {
}
