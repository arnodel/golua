package runtime

// Cont is an interface for things that can be run.
type Cont interface {
	Push(Value)
	RunInThread(*Thread) (Cont, error)
}

func Push(c Cont, vals ...Value) {
	for _, v := range vals {
		c.Push(v)
	}
}

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

// Push implements Continuation.Push.  It just accumulates values into
// a slice.
func (c *Termination) Push(v Value) {
	if c.pushIndex < len(c.args) {
		c.args[c.pushIndex] = v
		c.pushIndex++
	} else if c.etc != nil {
		*c.etc = append(*c.etc, v)
	}
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

// RunInThread implements Continuation.RunInThread.  It is not
// possible to run a Termination, so this always returns an error.
func (c *Termination) RunInThread(t *Thread) (Cont, error) {
	return nil, nil
}

// GoCont implements Cont for functions written in Go.
type GoCont struct {
	f    func(*Thread, []Value, Cont) (Cont, error)
	next Cont
	args []Value
}

// Push implements Cont.Push.
func (c *GoCont) Push(v Value) {
	if c.next == nil {
		var ok bool
		c.next, ok = v.(Cont)
		if !ok {
			panic("First push must be a continuation")
		}
	} else {
		c.args = append(c.args, v)
	}
}

// RunInThread implements Cont.RunInThread
func (c *GoCont) RunInThread(t *Thread) (Cont, error) {
	return c.f(t, c.args, c.next)
}
