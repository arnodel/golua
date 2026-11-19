package runtime

// POC B: a frame is one pooled object.  A released LuaCont keeps its
// registers, cleared, and goes on the list of its size class; a new frame of
// that class takes it from there.  One list to pop per call and one to push
// per return, where the continuation and its registers each had a pool.
//
// The lists are linked through the continuations (LuaCont.free), so the pool
// itself holds one pointer and one count per class.
type luaContPool struct {
	free  [poolClasses]*LuaCont
	count [poolClasses]int
	max   int // Number of frames kept in the smallest class
}

// get returns a continuation with n registers, all nil, for a function whose
// registers have the given class and size (see poolClass).  Its other fields
// are zero.
func (p *luaContPool) get(class, size, n int) *LuaCont {
	if class >= 0 {
		if c := p.free[class]; c != nil {
			p.free[class] = c.free
			p.count[class]--
			c.free = nil
			c.registers = c.registers[:n]
			return c
		}
	}
	return &LuaCont{registers: make([]Value, n, size)}
}

// release clears c and its registers and keeps it for reuse, unless the
// class has enough already or the registers are too many to pool.
func (p *luaContPool) release(class, size int, c *LuaCont) {
	regs := c.registers
	if len(regs) <= 8 {
		for i := 0; i < len(regs); i++ {
			regs[i] = Value{}
		}
	} else {
		clear(regs)
	}
	c.Closure = nil
	c.pc = 0
	c.acc = nil
	c.running = false
	c.closeStackBase = 0
	if class < 0 || cap(regs) != size || p.count[class] >= max(p.max>>class, 1) {
		c.registers = nil
		return
	}
	c.registers = regs[:0]
	c.free = p.free[class]
	p.free[class] = c
	p.count[class]++
}
