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

// Data passed between Threads via their resume channel (Thread.resumeCh).
//
// Supported types for exception are ContextTerminationError (which means
// execution has run out of resources) and threadClose (which means the thread
// should be closed without resuming execution, new in Lua 5.4 via the
// coroutine.close() function).  Other types will cause a panic.
type valuesError struct {
	args      []Value     // arguments ot yield or resume
	err       *Error      // execution error
	exception interface{} // used when the thread should be closed right away
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
	closeErr    *Error // Error that caused the thread to stop
	currentCont Cont   // Currently running continuation
	resumeCh    chan valuesError
	caller      *Thread // Who resumed this thread
	DebugHooks

	closeStack // Stack of pending to-be-closed values
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
				// Now we can do cleanup, cleanup should always return handled
				// errors (fingers crossed).
				// for c != nil {
				// 	log.Printf("RC Cleanup %s, %s", c.DebugInfo(), err)
				// 	err = c.Cleanup(t, err)
				// 	c = c.Next()
				// }
				return err
			}
			err.AddContext(c, -1)
			errContCount++
			if t.messageHandler != nil {
				if errContCount > maxErrorsInMessageHandler {
					return newHandledError(errErrorInMessageHandler)
				}
				next = t.messageHandler.Continuation(t, newMessageHandlerCont(c))
				next.Push(t.Runtime, err.Value())
			} else {
				next = newMessageHandlerCont(c)
				next.Push(t.Runtime, err.Value())
			}
		}
		c = next
	}
	return
}

// This is to be able to close a suspended coroutine without completing it, but
// still allow cleaning up the to-be-closed variables.  If this is put on the
// resume channel of a running thread, yield will cause a panic in the goroutine
// and that will be caught in the defer() clause below.
type threadClose struct{}

//
// Coroutine management
//

// Start starts the thread in a goroutine, giving it the callable c to run.  the
// t.Resume() method needs to be called to provide arguments to the callable.
func (t *Thread) Start(c Callable) {
	t.RequireBytes(2 << 10) // A goroutine starts off with 2k stack
	go func() {
		var (
			args []Value
			err  *Error
		)
		// If there was a panic due to an exceeded quota, we need to end the
		// thread and propagate that panic to the calling thread
		defer func() {
			r := recover()
			if r != nil {
				switch r.(type) {
				case ContextTerminationError:
				case threadClose:
					// This means we want to close the coroutine, so no panic!
					r = nil
				default:
					panic(r)
				}
			}
			t.end(args, err, r)
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

// Close a suspended thread.  If successful, its status switches to dead.  The
// boolean returned is true if it was possible to close the thread (i.e. it was
// suspended or already dead).  The error is non-nil if there was an error in
// the cleanup process, or if the thread had already stopped with an error
// previously.
func (t *Thread) Close(caller *Thread) (bool, *Error) {
	t.mux.Lock()
	if t.status != ThreadSuspended {
		t.mux.Unlock()
		switch t.status {
		case ThreadDead:
			return true, t.closeErr
		default:
			return false, nil
		}
	}
	caller.mux.Lock()
	if caller.status != ThreadOK {
		panic("Caller of thread to close is not running")
	}
	// The thread needs to go back to running to empty its close stack, before
	// becoming dead.
	t.caller = caller
	t.status = ThreadOK
	t.mux.Unlock()
	caller.mux.Unlock()
	t.sendResumeValues(nil, nil, threadClose{})
	_, err := caller.getResumeValues()
	return true, err
}

// Yield to the caller thread.  The yielding thread's status switches to
// suspended.  The caller's status must be OK.
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

// This turns off the thread, cleaning up its close stack.  The thread must be
// running.
func (t *Thread) end(args []Value, err *Error, exception interface{}) {
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
	}
	close(t.resumeCh)
	t.status = ThreadDead
	t.caller = nil
	err = t.cleanupCloseStack(nil, 0, err) // TODO: not nil
	t.closeErr = err
	caller.sendResumeValues(args, err, exception)
	t.ReleaseBytes(2 << 10) // The goroutine will terminate after this
}

func (t *Thread) call(c Callable, args []Value, next Cont) *Error {
	cont := t.ContWithArgs(c, args, next)
	return t.RunContinuation(cont)
}

func (t *Thread) getResumeValues() ([]Value, *Error) {
	res := <-t.resumeCh
	if res.exception != nil {
		panic(res.exception)
	}
	return res.args, res.err
}

func (t *Thread) sendResumeValues(args []Value, err *Error, exception interface{}) {
	t.resumeCh <- valuesError{args: args, err: err, exception: exception}
}

//
// Calling
//

func (t *Thread) CallContext(def RuntimeContextDef, f func() *Error) (ctx RuntimeContext, err *Error) {
	t.PushContext(def)
	c, h := t.CurrentCont(), t.closeStack.size()
	defer func() {
		ctx = t.PopContext()
		t.closeStack.truncate(h) // No resources to run that, so just discard it.
		if r := recover(); r != nil {
			_, ok := r.(ContextTerminationError)
			if !ok {
				panic(r)
			}
		}
	}()
	err = t.cleanupCloseStack(c, h, f())
	if err != nil {
		t.Runtime.setStatus(StatusError)
	}
	return
}

//
// close stack operations
//

type closeStack struct {
	stack []Value
}

func (s closeStack) size() int {
	return len(s.stack)
}

func (s *closeStack) push(v Value) {
	s.stack = append(s.stack, v)
}

func (s *closeStack) pop() (Value, bool) {
	sz := len(s.stack)
	if sz == 0 {
		return NilValue, false
	}
	sz--
	v := s.stack[sz]
	s.stack = s.stack[:sz]
	return v, true
}

func (s *closeStack) truncate(h int) {
	sz := len(s.stack)
	if sz > h {
		s.stack = s.stack[:h]
	}
}

// Truncate the close stack to size h, calling the __close metamethods in the
// context of the given continuation c and feeding them with the given error.
func (t *Thread) cleanupCloseStack(c Cont, h int, err *Error) *Error {
	closeStack := &t.closeStack
	for closeStack.size() > h {
		v, _ := closeStack.pop()
		if Truth(v) {
			closeErr, ok := Metacall(t, v, "__close", []Value{v, err.Value()}, NewTerminationWith(c, 0, false))
			if !ok {
				return NewErrorS("to be closed value missing a __close metamethod")
			}
			if closeErr != nil {
				err = closeErr
			}
		}
	}
	return err
}

//
// messageHandlerCont is a continuation that handles an error message (i.e.
// turns it to handled).
//
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

func (c *messageHandlerCont) Cleanup(t *Thread, err *Error) *Error {
	return err
}
