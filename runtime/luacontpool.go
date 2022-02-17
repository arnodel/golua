//go:build !nocontpool
// +build !nocontpool

package runtime

// Size of the LuaCont pool.  Setting it to 0 makes luaContPool.get() behave
// like new(LuaCont) and luaContPool.release(c) be a no-op. The value of 100 was
// reached by trial-and-error but is probably not optimal.
const luaContPoolSize = 100

// Pool for reusing allocated Lua continuations.  Some profiling
// showed there was significant overhead to allocating lua continuations all the
// time on the heap.  This is a very simple implementation, but it reduces
// significantly pressure on memory management for a fair range of workloads.
type luaContPool struct {
	conts [luaContPoolSize]*LuaCont
	next  int
}

// Get a LuaCont from the pool (or make a new one if the pool is empty).  Note
// that it is ok not to release this LuaCont to the pool later - if it is not
// released but becomes unreachable, then it will be GCed at some point by the
// Go runtime
func (p *luaContPool) get() *LuaCont {
	if p.next == 0 {
		return new(LuaCont)
	}
	p.next--
	c := p.conts[p.next]
	p.conts[p.next] = nil
	return c
}

// Return a used LuaCont into the pool (this will first erase the fields of the
// continuation, to allow GC of the data they contain).  If the pool is full,
// the continuation is simply discarded.
func (p *luaContPool) release(cont *LuaCont) {
	*cont = LuaCont{}
	if p.next == luaContPoolSize {
		return
	}
	p.conts[p.next] = cont
	p.next++
}
