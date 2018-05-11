package runtime

type Closure struct {
	*Code
	upvalues     []Cell
	upvalueIndex int
}

func NewClosure(c *Code) *Closure {
	return &Closure{
		Code:     c,
		upvalues: make([]Cell, c.UpvalueCount),
	}
}

func (c *Closure) AddUpvalue(cell Cell) {
	c.upvalues[c.upvalueIndex] = cell
	c.upvalueIndex++
}

func (c *Closure) Continuation(next Cont) Cont {
	return NewLuaCont(c, next)
}
