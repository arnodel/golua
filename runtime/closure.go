package runtime

import "unsafe"

// Closure is the data structure that represents a Lua function.  It is simply a
// reference to a Code instance and a set of upvalues.
type Closure struct {
	*Code
	Upvalues     []Cell
	upvalueIndex int
}

// NewClosure returns a pointer to a new Closure instance for the given code.
func NewClosure(r *Runtime, c *Code) *Closure {
	if c.UpvalueCount > 0 {
		r.RequireArrSize(unsafe.Sizeof(Cell{}), int(c.UpvalueCount))
	}
	return &Closure{
		Code:     c,
		Upvalues: make([]Cell, c.UpvalueCount),
	}
}

func (c *Closure) Equals(c1 *Closure) bool {
	if c.Code != c1.Code || len(c.Upvalues) != len(c1.Upvalues) {
		return false
	}
	for i, upv := range c.Upvalues {
		if c1.Upvalues[i] != upv {
			return false
		}
	}
	return true
}

// AddUpvalue append a new upvalue to the closure.
func (c *Closure) AddUpvalue(cell Cell) {
	c.Upvalues[c.upvalueIndex] = cell
	c.upvalueIndex++
}

// Continuation implements Callable.Continuation
func (c *Closure) Continuation(r *Runtime, next Cont) Cont {
	return NewLuaCont(r, c, next)
}

// GetUpvalue returns the upvalue for c at index n.
func (c *Closure) GetUpvalue(n int) Value {
	return c.Upvalues[n].Get()
}

// SetUpvalue sets the upvalue for c at index n to v.
func (c *Closure) SetUpvalue(n int, val Value) {
	c.Upvalues[n].Set(val)
}
