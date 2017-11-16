package runtime

type Function struct {
	code          Code
	constants     []Value
	upvalueCount  uint
	registerCount uint
}

type Receiver struct {
	currentIndex int
	res          []Value
	rest         *[]Value
}

func (r Receiver) Push(v Value) {
	if r.currentIndex < len(r.res) {
		r.res[r.currentIndex] = v
		r.currentIndex++
	} else if r.rest != nil {
		*r.rest = append(*r.rest, v)
	}
}

func (c *Closure) Call(args []Value, res []Value, rest *[]Value) error {
	rec := Receiver{0, res, rest}
	cont := c.NewContinuation()
	cont.Push(rec)
	for arg := range args {
		cont.Push(arg)
	}
	cont.Resume()
	return nil
}
