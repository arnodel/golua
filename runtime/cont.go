package runtime

// Continuation is an interface for things that can be run.
type Continuation interface {
	Push(Value)
	RunInThread(*Thread) (Continuation, error)
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

// RunInThread implements Continuation.RunInThread.  It is not
// possible to run a Termination, so this always returns an error.
func (c *Termination) RunInThread(t *Thread) (Continuation, error) {
	return nil, nil
}

// ContinuationThreadStarter wraps a value that implements
// Continuation and turns it into an implementor of ThreadStarter.
type ContinuationThreadStarter struct {
	cont Continuation
}

// StartThread implements ThreadStarter.StartThread.
func (ts *ContinuationThreadStarter) StartThread(t *Thread, args []Value) ([]Value, error) {
	term := new(Termination)
	cont := ts.cont
	cont.Push(term)
	for _, arg := range args {
		cont.Push(arg)
	}
	var err error
	for cont != term {
		cont, err = cont.RunInThread(t)
		if err != nil {
			return nil, err
		}
	}
	return term.args, nil
}

// GoContinuation implements Continuation for functions written in
// Go. It's not optimal but simple and general. Probably later add
// other ways of turning Go functions with other signatures into
// continuations.
type GoContinuation struct {
	f    func(*Thread, []Value, Continuation) error
	next Continuation
	args []Value
}

// Push implements Continuation.Push.
func (c *GoContinuation) Push(v Value) {
	if c.next == nil {
		var ok bool
		c.next, ok = v.(Continuation)
		if !ok {
			panic("First push must be a continuation")
		}
	} else {
		c.args = append(c.args, v)
	}
}

// RunInThread implements Continuation.RunInThread
func (c *GoContinuation) RunInThread(t *Thread) (Continuation, error) {
	err := c.f(t, c.args, c.next)
	if err != nil {
		return nil, err
	}
	return c.next, nil
}
