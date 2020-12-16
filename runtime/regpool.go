package runtime

var globalRegPool = regPool{}

const regPoolSize = 10 // Size of a pool of cell of value register sets.
const maxAge = 10      // Age at which it's OK to discard a register set in the pool.

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
// Setting regPoolSize to 0 makes regPool.getCells / regPool.getRegs allocate
// a new register set each time, and regPool.releaseRegs / regPool.releaseCells
// be no-ops.
type regPool struct {
	cells    [regPoolSize][]Cell  // Pool of cell register sets
	regs     [regPoolSize][]Value // Pool of value register sets
	cellExps [regPoolSize]uint64  // Expiry generation of cell register sets
	regExps  [regPoolSize]uint64  // Expiry generation of value register sets
	cellGen  uint64               // Current cell register set generation
	regGen   uint64               // current value register set generation
}

// Get a register set of size sz, taken from the pool if possible (and
// increase the current generation).
func (p *regPool) getRegs(sz int) []Value {
	p.regGen++
	for i := 0; i < regPoolSize; i++ {
		r := p.regs[i]
		if len(r) == sz {
			p.regs[i] = nil
			p.regExps[i] = 0
			return r
		}
	}
	return make([]Value, sz)
}

// Get a cell register set of size sz, taken from the pool if possible (and
// increase the current generation).
func (p *regPool) getCells(sz int) []Cell {
	p.cellGen++
	for i := 0; i < regPoolSize; i++ {
		c := p.cells[i]
		if len(c) == sz {
			p.cells[i] = nil
			p.cellExps[i] = 0
			return c
		}
	}
	return make([]Cell, sz)
}

// Return the regiter set to the pool if there is a slot available (i.e. empty
// slot or expired register set).
func (p *regPool) releaseRegs(r []Value) {
	for i := 0; i < regPoolSize; i++ {
		if p.regExps[i] < p.regGen {
			for i := range r {
				r[i] = NilValue
			}
			p.regs[i] = r
			p.regExps[i] = p.regGen + maxAge
			return
		}
	}
}

// Return the cell regiter set to the pool if there is a slot available (i.e.
// empty slot or expired register set).
func (p *regPool) releaseCells(c []Cell) {
	for i := 0; i < regPoolSize; i++ {
		if p.cellExps[i] < p.cellGen {
			for i := range c {
				c[i] = Cell{}
			}
			p.cells[i] = c
			p.cellExps[i] = p.cellGen + maxAge
			return
		}
	}
}
