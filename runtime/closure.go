package runtime

type Closure struct {
	*Code
	Upvalues     []Cell
	upvalueIndex int
}

func NewClosure(c *Code) *Closure {
	return &Closure{
		Code:     c,
		Upvalues: make([]Cell, c.UpvalueCount),
	}
}

func (c *Closure) AddUpvalue(cell Cell) {
	c.Upvalues[c.upvalueIndex] = cell
	c.upvalueIndex++
}

func (c *Closure) Continuation(next Cont) Cont {
	return NewLuaCont(c, next)
}

func (c *Closure) GetUpvalue(n int) Value {
	return c.Upvalues[n].Get()
}

func (c *Closure) SetUpValue(n int, val Value) {
	c.Upvalues[n].Set(val)
}
