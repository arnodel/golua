package runtime

type Bool bool
type Int int64
type Float float64
type NilType struct{}
type String string

type Callable interface {
	Continuation() Continuation
}

type ToStringable interface {
	ToString() string
}

type Metatabler interface {
	Metatable() *Table
}

type Closure struct {
	*Code
	upvalues     []Value
	upvalueIndex int
}

func NewClosure(c *Code) *Closure {
	return &Closure{
		Code:     c,
		upvalues: make([]Value, c.UpvalueCount),
	}
}

func (c *Closure) AddUpvalue(v Value) {
	c.upvalues[c.upvalueIndex] = v
	c.upvalueIndex++
}

func (c *Closure) Continuation() Continuation {
	return NewLuaContinuation(c)
}

type GoFunction func(t *Thread, args []Value, next Continuation) error

func (f GoFunction) Continuation() Continuation {
	return &GoContinuation{f: f}
}

func Call(c Callable, args []Value, next Continuation) Continuation {
	cont := c.Continuation()
	cont.Push(next)
	for _, arg := range args {
		cont.Push(arg)
	}
	return cont
}
