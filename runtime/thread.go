package runtime

import (
	"sync"
	"unsafe"
)

// ThreadStatus is the type of a thread status
type ThreadStatus uint

// Available statuses for threads.
const (
	ThreadOK        ThreadStatus = 0 // Running thread
	ThreadSuspended ThreadStatus = 1 // Thread has yielded and is waiting to be resumed
	ThreadDead      ThreadStatus = 3 // Thread has finished and cannot be resumed
)

type valuesError struct {
	args     []Value
	err      *Error
	quotaErr *ContextTerminationError
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
	DebugHooks
}

// NewThread creates a new thread out of a Runtime.  Its initial
// status is suspended.  Call Resume to run it.
func NewThread(r *Runtime) *Thread {
	r.RequireSize(unsafe.Sizeof(Thread{}) + 100) // 100 is my guess at the size of a channel
	return &Thread{
		resumeCh: make(chan valuesError),
		status:   ThreadSuspended,
		Runtime:  r,
	}
}

// CurrentCont returns the continuation currently running (or suspended) in the
// thread.
func (t *Thread) CurrentCont() Cont {
	return t.currentCont
}

// IsMain returns true if the thread is the runtime's main thread.
func (t *Thread) IsMain() bool {
	return t == t.mainThread
}

const maxErrorsInMessageHandler = 10

var errErrorInMessageHandler = StringValue("error in error handling")

// RunContinuation runs the continuation c in the thread. It keeps running until
// the next continuation is nil or an error occurs, in which case it returns the
// error.
func (t *Thread) RunContinuation(c Cont) (err *Error) {
	var next Cont
	var errContCount = 0
	_ = t.triggerCall(t, c)
	for c != nil {
		t.currentCont = c
		next, err = c.RunInThread(t)
		if err != nil {
			if err.Handled() {
				return err
			}
			err.AddContext(c, -1)
			errContCount++
			if t.messageHandler != nil {
				if errContCount > maxErrorsInMessageHandler {
					return newHandledError(errErrorInMessageHandler)
				}
				next = t.messageHandler.Continuation(t.Runtime, newMessageHandlerCont(c))
				next.Push(t.Runtime, err.Value())
			} else {
				// If there is no message handler, just return the error
				return err
			}
		}
		c = next
	}
	return
}

// Start starts the thread in a goroutine, giving it the callable c to run.  the
// t.Resume() method needs to be called to provide arguments to the callable.
func (t *Thread) Start(c Callable) {
	t.RequireBytes(2 << 10) // A goroutine starts off with 2k stack
	go func() {
		var (
			args     []Value
			err      *Error
			quotaErr *ContextTerminationError
		)
		// If there was a panic due to an exceeded quota, we need to end the
		// thread and propagate that panic to the calling thread
		defer func() {
			if r := recover(); r != nil {
				quotaErr = new(ContextTerminationError)
				var ok bool
				*quotaErr, ok = r.(ContextTerminationError)
				if !ok {
					panic(r)
				}
			}
			t.end(args, err, quotaErr)
		}()
		args, err = t.getResumeValues()
		if err == nil {
			next := NewTerminationWith(t.CurrentCont(), 0, true)
			err = t.call(c, args, next)
			args = next.Etc()
		}
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
	t.sendResumeValues(args, nil, nil)
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
	caller.sendResumeValues(args, nil, nil)
	return t.getResumeValues()
}

func (t *Thread) end(args []Value, err *Error, quotaErr *ContextTerminationError) {
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
		close(t.resumeCh)
		t.status = ThreadDead
		t.caller = nil
	}
	caller.sendResumeValues(args, err, quotaErr)
	t.ReleaseBytes(2 << 10) // The goroutine will terminate after this
}

func (t *Thread) call(c Callable, args []Value, next Cont) *Error {
	cont := t.ContWithArgs(c, args, next)
	return t.RunContinuation(cont)
}

func (t *Thread) getResumeValues() ([]Value, *Error) {
	res := <-t.resumeCh
	if res.quotaErr != nil {
		panic(*res.quotaErr)
	}
	return res.args, res.err
}

func (t *Thread) sendResumeValues(args []Value, err *Error, quotaErr *ContextTerminationError) {
	t.resumeCh <- valuesError{args, err, quotaErr}
}

type messageHandlerCont struct {
	c    Cont
	err  Value
	done bool
}

func newMessageHandlerCont(c Cont) *messageHandlerCont {
	return &messageHandlerCont{c: c}
}

var _ Cont = (*messageHandlerCont)(nil)

func (c *messageHandlerCont) DebugInfo() *DebugInfo {
	return c.c.DebugInfo()
}

func (c *messageHandlerCont) Next() Cont {
	return c.c.Next()
}

func (c *messageHandlerCont) Parent() Cont {
	return c.Next()
}

func (c *messageHandlerCont) Push(r *Runtime, v Value) {
	if !c.done {
		c.done = true
		c.err = v
	}
}

func (c *messageHandlerCont) PushEtc(r *Runtime, etc []Value) {
	if c.done || len(etc) == 0 {
		return
	}
	c.Push(r, etc[0])
}

func (c *messageHandlerCont) RunInThread(t *Thread) (Cont, *Error) {
	return nil, newHandledError(c.err)
}
