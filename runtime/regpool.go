//go:build !noregpool && noquotas
// +build !noregpool,noquotas

package runtime

const (
	regPoolSize  = 10 // Size of a pool of cell of value register sets.
	regSetMaxAge = 10 // Age at which it's OK to discard a register set in the pool.
)

// This is an experimental pool for re-using allocated sets of registers across
// different Lua continuations.  Profiling showed allocating register sets for
// each continuation is costly, and it can easily be avoided in the case of e.g.
// a function repeatedly called in a loop, or a tail recursive function (or
// mutually tail recursive functions).
//
// The pool keeps up to regPoolSize register sets of each type (plain values and
// cells).  A register set can be released into the pool and it will replace an
// old (> maxAge) register set if there is one. Age increase each time a new
// register set is requested.  The idea is that in the common case where the
// same function keeps being called, the same register set can be re-used.
//
// Setting regPoolSize to 0 makes cellPool.get / valuePool.get allocate
// a new register set each time, and cellPool.release / valuePool.release
// be no-ops.

type cellPool struct {
	cells  [][]Cell // Pool of cell sets
	exps   []uint   // Expiry generation of cell sets
	gen    uint     // Current cell register set generation
	maxAge uint
}

func mkCellPool(size, maxAge uint) cellPool {
	return cellPool{
		cells:  make([][]Cell, size),
		exps:   make([]uint, size),
		maxAge: maxAge,
	}
}

func (p *cellPool) get(sz int) []Cell {
	p.gen++
	for i := 0; i < regPoolSize; i++ {
		c := p.cells[i]
		if len(c) == sz {
			p.cells[i] = nil
			p.exps[i] = 0
			return c
		}
	}
	return make([]Cell, sz)
}

func (p *cellPool) release(c []Cell) {
	for i := 0; i < regPoolSize; i++ {
		if p.exps[i] < p.gen {
			for i := range c {
				c[i] = Cell{}
			}
			p.cells[i] = c
			p.exps[i] = p.gen + p.maxAge
			return
		}
	}
}

type valuePool struct {
	values [][]Value // Pool of value sets
	exps   []uint    // Expiry generation of value sets
	gen    uint      // Current cell register set generation
	maxAge uint
}

func mkValuePool(size, maxAge uint) valuePool {
	return valuePool{
		values: make([][]Value, size),
		exps:   make([]uint, size),
		maxAge: maxAge,
	}
}

func (p *valuePool) get(sz int) []Value {
	p.gen++
	for i := 0; i < regPoolSize; i++ {
		v := p.values[i]
		if len(v) == sz {
			p.values[i] = nil
			p.exps[i] = 0
			return v
		}
	}
	return make([]Value, sz)
}

func (p *valuePool) release(v []Value) {
	for i := 0; i < regPoolSize; i++ {
		if p.exps[i] < p.gen {
			for i := range v {
				v[i] = Value{}
			}
			p.values[i] = v
			p.exps[i] = p.gen + p.maxAge
			return
		}
	}
}
