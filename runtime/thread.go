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

// A Thread is a lua thread.
//
// The mutex guarantees that if status == ThreadRunning, then caller
// is not nil.
//
type Thread struct {
	*Runtime
	mux         sync.Mutex
	status      ThreadStatus
	currentCont Cont
	resumeCh    chan valuesError
	caller      *Thread
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

// CurrentCont returns the continuation currently running (or suspended) in the
// thread.
func (t *Thread) CurrentCont() Cont {
	return t.currentCont
}

// IsMain returns true if the thread is the runtime's main thread.
func (t *Thread) IsMain() bool {
	return t.caller == nil
}

// RunContinuation runs the continuation c in the thread. It keeps running until
// the next continuation is nil or an error occurs, in which case it returns the
// error.
func (t *Thread) RunContinuation(c Cont) (err *Error) {
	for c != nil && err == nil {
		t.currentCont = c
		c, err = c.RunInThread(t)
	}
	return
}

// Start starts the thread in a goroutine, giving it the callable c to run.  the
// t.Resume() method needs to be called to provide arguments to the callable.
func (t *Thread) Start(c Callable) {
	go func() {
		args, err := t.getResumeValues()
		if err == nil {
			next := NewTerminationWith(0, true)
			err = t.call(c, args, next)
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
	t.mux.Lock()
	if t.status != ThreadSuspended {
		t.mux.Unlock()
		switch t.status {
		case ThreadDead:
			return nil, NewErrorS("cannot resume dead thread")
		default:
			return nil, NewErrorS("cannot resume running thread")
		}
	}
	caller.mux.Lock()
	if caller.status != ThreadOK {
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
	if caller == nil {
		t.mux.Unlock()
		return nil, NewErrorS("cannot yield from main thread")
	}
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

func (t *Thread) call(c Callable, args []Value, next Cont) *Error {
	cont := t.ContWithArgs(c, args, next)
	return t.RunContinuation(cont)
}

func (t *Thread) getResumeValues() ([]Value, *Error) {
	res := <-t.resumeCh
	return res.args, res.err
}

func (t *Thread) sendResumeValues(args []Value, err *Error) {
	t.resumeCh <- valuesError{args, err}
}
