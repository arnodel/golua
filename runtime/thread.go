package runtime

import (
	"sync"
)

// ThreadStarter is an interface for things that can make a start a
// thread.
type ThreadStarter interface {
	StartThread(*Thread, []Value) ([]Value, error)
}

// ThreadStatus is the type of a thread status
type ThreadStatus uint

// Available statuses for threads.
const (
	ThreadOK        ThreadStatus = 0
	ThreadSuspended ThreadStatus = 1
	ThreadDead      ThreadStatus = 3
)

type valuesError struct {
	args []Value
	err  *Error
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

func (t *Thread) IsMain() bool {
	return t.caller == nil
}

func (t *Thread) getResumeValues() ([]Value, *Error) {
	res := <-t.resumeCh
	return res.args, res.err
}

func (t *Thread) sendResumeValues(args []Value, err *Error) {
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

func (t *Thread) RunContinuation(c Cont) (err *Error) {
	for c != nil && err == nil {
		c, err = c.RunInThread(t)
	}
	return
}

func (t *Thread) Call(c Callable, args []Value, next Cont) *Error {
	cont := ContWithArgs(c, args, next)
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
func (t *Thread) Resume(caller *Thread, args []Value) ([]Value, *Error) {
	// You cannot wake up from the dead, so no need for a lock.
	if t.status == ThreadDead {
		return nil, NewErrorS("Cannot resume a dead thread")
	}
	t.mux.Lock()
	caller.mux.Lock()
	if t.status != ThreadSuspended {
		panic("Thread to resume is not suspended")
	} else if caller.status != ThreadOK {
		panic("Caller of thread to resume is not running")
	}
	t.caller = caller
	t.status = ThreadOK
	t.mux.Unlock()
	caller.mux.Unlock()
	t.sendResumeValues(args, nil)
	return caller.getResumeValues()
}

// Yield to the caller thread.  The yielding thread's status switches
// to suspended while the caller's status switches back to running.
func (t *Thread) Yield(args []Value) ([]Value, *Error) {
	t.mux.Lock()
	if t.status != ThreadOK {
		panic("Thread to yield is not running")
	}
	caller := t.caller
	caller.mux.Lock()
	if caller.status != ThreadOK {
		panic("Caller of thread to yield is not OK")
	}
	t.status = ThreadSuspended
	t.caller = nil
	t.mux.Unlock()
	caller.mux.Unlock()
	caller.sendResumeValues(args, nil)
	return t.getResumeValues()
}

func (t *Thread) end(args []Value, err *Error) {
	caller := t.caller
	t.mux.Lock()
	caller.mux.Lock()
	defer t.mux.Unlock()
	defer caller.mux.Unlock()
	switch {
	case t.status != ThreadOK:
		panic("Called Thread.end on a non-running thread")
	case caller.status != ThreadOK:
		panic("Caller thread of ending thread is not OK")
	default:
		t.status = ThreadDead
		t.caller = nil
	}
	caller.sendResumeValues(args, err)
}
