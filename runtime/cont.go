package runtime

import "errors"

// Continuation is an interface for things that can be run.
type Continuation interface {
	Push(Value)
	RunInThread(*Thread) (Continuation, error)
}

// Termination is a 'dead-end' continuation: it cannot be run
type Termination struct {
	args []Value
}

// Push implements Continuation.Push.  It just accumulates values into
// a slice.
func (c *Termination) Push(v Value) {
	c.args = append(c.args, v)
}

// RunInThread implements Continuation.RunInThread.  It is not
// possible to run a Termination, so this always returns an error.
func (c *Termination) RunInThread(t *Thread) (Continuation, error) {
	return nil, errors.New("Terminations cannot be run")
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
	f    func(*Thread, []Value) ([]Value, error)
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
	rets, err := c.f(t, c.args)
	if err != nil {
		return nil, err
	}
	for _, ret := range rets {
		c.next.Push(ret)
	}
	return c.next, nil
}
