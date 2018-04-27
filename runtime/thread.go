package runtime

import (
	"errors"
	"sync"
)

// Value is a runtime value.
type Value interface{}

// ThreadStarter is an interface for things that can make a start a
// thread.
type ThreadStarter interface {
	StartThread(*Thread, []Value) ([]Value, error)
}

// ThreadStatus is the type of a thread status
type ThreadStatus uint

// Available statuses for threads.
const (
	ThreadRunning   ThreadStatus = 0
	ThreadSuspended ThreadStatus = 1
	ThreadDead      ThreadStatus = 3
)

type valuesError struct {
	args []Value
	err  error
}

// Thread is a lua thread.
//
// The mutex guarantees that if status == ThreadRunning, then caller
// is not nil.
//
type Thread struct {
	*Runtime
	mux      sync.Mutex
	status   ThreadStatus
	resumeCh chan valuesError
	caller   *Thread
}

func (t *Thread) getResumeValues() ([]Value, error) {
	res := <-t.resumeCh
	return res.args, res.err
}

func (t *Thread) sendResumeValues(args []Value, err error) {
	t.resumeCh <- valuesError{args, err}
}

// NewThread creates a new thread out of a ThreadStarter.  Its initial
// status is suspended.  Call Resume to run it.
func NewThread(rt *Runtime) *Thread {
	return &Thread{
		resumeCh: make(chan valuesError),
		status:   ThreadSuspended,
		Runtime:  rt,
	}
}

func (t *Thread) RunContinuation(c Continuation) (err error) {
	for c != nil {
		c, err = c.RunInThread(t)
	}
	return
}

func (t *Thread) Call(c Callable, args []Value, next Continuation) error {
	cont := c.Continuation()
	cont.Push(next)
	for _, arg := range args {
		cont.Push(arg)
	}
	return t.RunContinuation(cont)
}

func (t *Thread) Start(c Callable) {
	go func() {
		args, err := t.getResumeValues()
		if err == nil {
			next := NewTerminationWith(0, true)
			err = t.Call(c, args, next)
			args = next.Etc()
		}
		t.end(args, err)
	}()
}

// Status returns the status of a thread (suspended, running or dead).
func (t *Thread) Status() ThreadStatus {
	return t.status
}

// Resume execution of a suspended thread.  Its status switches to
// running while its caller's status switches to suspended.
func (t *Thread) Resume(caller *Thread, args []Value) ([]Value, error) {
	// You cannot wake up from the dead, so no need for a lock.
	if t.status == ThreadDead {
		return nil, errors.New("Cannot resume a dead thread")
	}
	t.mux.Lock()
	caller.mux.Lock()
	if t.status != ThreadSuspended {
		panic("Thread to resume is not suspended")
	} else if caller.status != ThreadRunning {
		panic("Caller of thread to resume is not running")
	}
	t.caller = caller
	t.status = ThreadRunning
	caller.status = ThreadSuspended
	t.mux.Unlock()
	caller.mux.Unlock()
	t.sendResumeValues(args, nil)
	return caller.getResumeValues()
}

// Yield to the caller thread.  The yielding thread's status switches
// to suspended while the caller's status switches back to running.
func (t *Thread) Yield(args []Value) ([]Value, error) {
	t.mux.Lock()
	if t.status != ThreadRunning {
		panic("Thread to yield is not running")
	}
	caller := t.caller
	caller.mux.Lock()
	if caller.status != ThreadSuspended {
		panic("Caller of thread to yield is not suspended")
	}
	t.status = ThreadSuspended
	caller.status = ThreadRunning
	t.caller = nil
	t.mux.Unlock()
	caller.mux.Unlock()
	caller.sendResumeValues(args, nil)
	return t.getResumeValues()
}

func (t *Thread) end(args []Value, err error) {
	t.mux.Lock()
	var caller *Thread
	if t.status == ThreadRunning {
		t.status = ThreadDead
		caller = t.caller
		t.caller = nil
		t.mux.Unlock()
	} else {
		panic("Called Thread.end on a non-running thread")
	}
	caller.sendResumeValues(args, err)
}
