// +build nocontpool
// This tag disables the continuation pool

package runtime

var globalLuaContPool = dummyLuaContPool{}

// Dummy pool.
type dummyLuaContPool struct{}

// Same as new(LuaCont).
func (p dummyLuaContPool) get() *LuaCont {
	return new(LuaCont)
}

// Does nothing.
func (p dummyLuaContPool) release(cont *LuaCont) {
}
