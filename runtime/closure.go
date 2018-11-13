package runtime

// Closure is the data structure that represents a Lua function.  It is simply a
// reference to a Code instance and a set of upvalues.
type Closure struct {
	*Code
	Upvalues     []Cell
	upvalueIndex int
}

// NewClosure returns a pointer to a new Closure instance for the given code.
func NewClosure(c *Code) *Closure {
	return &Closure{
		Code:     c,
		Upvalues: make([]Cell, c.UpvalueCount),
	}
}

// AddUpvalue append a new upvalue to the closure.
func (c *Closure) AddUpvalue(cell Cell) {
	c.Upvalues[c.upvalueIndex] = cell
	c.upvalueIndex++
}

// Continuation implements Callable.Continuation
func (c *Closure) Continuation(next Cont) Cont {
	return NewLuaCont(c, next)
}

// GetUpvalue returns the upvalue for c at index n.
func (c *Closure) GetUpvalue(n int) Value {
	return c.Upvalues[n].Get()
}

// SetUpvalue sets teh upvalue for c at index n to v.
func (c *Closure) SetUpvalue(n int, val Value) {
	c.Upvalues[n].Set(val)
}
