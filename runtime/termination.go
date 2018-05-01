package runtime

// Termination is a 'dead-end' continuation: it cannot be run
type Termination struct {
	args      []Value
	pushIndex int
	etc       *[]Value
}

func NewTermination(args []Value, etc *[]Value) *Termination {
	return &Termination{args: args, etc: etc}
}

func NewTerminationWith(nArgs int, hasEtc bool) *Termination {
	var args []Value
	var etc *[]Value
	if nArgs > 0 {
		args = make([]Value, nArgs)
	}
	if hasEtc {
		etc = new([]Value)
	}
	return NewTermination(args, etc)
}

// Push implements Cont.Push.  It just accumulates values into
// a slice.
func (c *Termination) Push(v Value) {
	if c.pushIndex < len(c.args) {
		c.args[c.pushIndex] = v
		c.pushIndex++
	} else if c.etc != nil {
		*c.etc = append(*c.etc, v)
	}
}

// RunInThread implements Cont.RunInThread. A termination exits
// immediately so it always returns nil.
func (c *Termination) RunInThread(t *Thread) (Cont, *Error) {
	return nil, nil
}

// Next implmements Cont.Next.
func (c *Termination) Next() Cont {
	return nil
}

func (c *Termination) Get(n int) Value {
	return c.args[n]
}

func (c *Termination) Etc() []Value {
	if c.etc == nil {
		return nil
	}
	return *c.etc
}
