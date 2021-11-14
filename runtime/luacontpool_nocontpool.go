//go:build nocontpool
// +build nocontpool

// This tag disables the continuation pool

package runtime

// Dummy pool.
type luaContPool struct{}

// Same as new(LuaCont).
func (p luaContPool) get() *LuaCont {
	return new(LuaCont)
}

// Does nothing.
func (p luaContPool) release(cont *LuaCont) {
}
