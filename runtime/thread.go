package runtime

import (
	"errors"
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

// The depth of GoFunction calls in one thread is limited by this number in
// order to avoid irrecoverable Go stack overflows.
const maxGoFunctionCallDepth = 1000

// The number of nested continuation loops (RunContinuation) active in one
// thread is limited by this number, for the same reason. A Lua function that
// the VM calls directly as a metamethod (__index, __newindex, __eq, __lt,
// __le, arithmetic, __len, __concat, __close) runs in a nested loop on the Go
// stack without going through a GoFunction, so maxGoFunctionCallDepth alone
// does not bound it: a metamethod that triggers itself would otherwise grow
// the Go stack until the process aborts.
const maxRunContinuationDepth = 1000

// Data passed between Threads via their resume channel (Thread.resumeCh).
//
// Supported types for exception are ContextTerminationError (which means
// execution has run out of resources) and threadClose (which means the thread
// should be closed without resuming execution, new in Lua 5.4 via the
// coroutine.close() function).  Other types will cause a panic.
type valuesError struct {
	args      []Value     // arguments ot yield or resume
	err       error       // execution error
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
	closeErr    error // Error that caused the thread to stop
	currentCont Cont  // Currently running continuation
	resumeCh    chan valuesError
	caller      *Thread // Who resumed this thread

	// Depth of GoFunction calls in the thread.  This should not exceed
	// maxGoFunctionCallDepth.  The aim is to avoid Go stack overflows that
	// cannot be recovered from (note that this does not limit recursion for Lua
	// functions).
	goFunctionCallDepth int

	// Depth of nested RunContinuation loops in the thread.  This should not
	// exceed maxRunContinuationDepth.
	runContinuationDepth int

	// The loop that is running, as the continuations captured in it know it
	// (see runLoop).  It is nil until a continuation is captured in the
	// loop.
	loop *runLoop

	// Number of calls of Lua code under way that a yield cannot cross: those
	// made by a Go function declared NoYieldAcross, and those of a debug
	// hook.  While there is one the thread cannot yield.
	noYieldDepth int

	// Depth of the protected calls under way in the thread (see
	// protectedcall.go).  This should not exceed maxProtectedCallDepth.
	protectedCallDepth int

	// Depth of __call metamethod chain (Lua 5.5).  This should not exceed
	// maxCallChainLength to prevent infinite loops through chained __call metamethods.
	metacallChainDepth int

	DebugHooks

	closeStack // Stack of pending to-be-closed values

	// A light thread has no goroutine of its own: see lightthread.go, the
	// only place where these change.
	light       bool         // The thread is light
	body        Callable     // What the thread runs, until it has started
	term        *Termination // Receives what the body returns
	resumed     bool         // The thread was resumed and its yield has yet to return
	resumedWith []Value      // The values of that resume, which the yield returns
}

// NewThread creates a new thread out of a Runtime.  Its initial
// status is suspended.  Call Resume to run it.
func NewThread(r *Runtime) *Thread {
	r.RequireSize(unsafe.Sizeof(Thread{}))
	return &Thread{
		status:  ThreadSuspended,
		Runtime: r,
	}
}

// ensureResumeCh makes the channel that the thread waits on for values from
// another thread, if it has none.  A thread needs it from when it has a
// goroutine, or waits for one that has: it must be there before the other
// goroutine can send to it.  A light thread that resumes only light threads
// never has one.
func (t *Thread) ensureResumeCh() {
	if t.resumeCh == nil {
		t.RequireSize(100) // A guess at the size of a channel
		t.resumeCh = make(chan valuesError)
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
func (t *Thread) RunContinuation(c Cont) error {
	return t.runContinuation(c, true)
}

// enterLoop is called when a loop starts running continuations in the
// thread, and leaveLoop when it has ended, with what enterLoop returned.
// They count the loops under way and keep the record of the one that is
// running (see runLoop): a call from Go is a new loop, whereas a thread that
// goes on in a goroutine is still in the loop it was in.
//
// Each is one line of runContinuation, in the place of a line that was
// there: the build's profile counts lines (see benchmarks/README.md).
func (t *Thread) enterLoop(isCall bool) (outer *runLoop) {
	t.runContinuationDepth++
	outer = t.loop
	if isCall {
		t.loop = nil
	}
	return outer
}

func (t *Thread) leaveLoop(outer *runLoop) {
	t.runContinuationDepth--
	t.loop = outer
}

// runContinuation is RunContinuation, with the choice of whether c is the
// start of a call, which the call hook is told of, or the rest of something
// that was under way (a thread that goes on in a goroutine).
func (t *Thread) runContinuation(c Cont, isCall bool) (err error) {
	// Go code is calling Lua.  If it is a function that no yield crosses,
	// the Lua code cannot yield.
	caller, _ := t.currentCont.(*GoCont)
	noYield := caller != nil && caller.noYieldAcross
	if t.light && !noYield && t.noYieldDepth == 0 {
		// The Lua code may yield: something a light thread cannot have
		// Go code waiting for.
		return errNeedsGoroutine
	}
	if t.runContinuationDepth >= maxRunContinuationDepth {
		return errors.New("stack overflow")
	}
	if noYield {
		t.noYieldDepth++
	}
	// On the way out, put back the continuation the thread was running.
	// Otherwise it stays on the termination this loop ended on, and the next
	// call from Go makes its own termination with that one as parent: a
	// chain that grows with every call and keeps all their results alive.
	//
	// Not when a coroutine's own loop ends: the coroutine is dead, and its
	// traceback shows the continuation it stopped in.
	current := t.currentCont
	outer := t.enterLoop(isCall)
	defer func() {
		if noYield {
			t.noYieldDepth--
		}
		t.leaveLoop(outer)
		coroutineEnds := t.runContinuationDepth == 0 && t.caller != nil
		if !coroutineEnds {
			t.currentCont = current
		}
	}()

	var next Cont
	if isCall {
		_ = t.triggerCall(t, c)
	}
	for c != nil {
		// Checked at every continuation, so twice per call: the check is
		// one load, inline, and the work only when there is some.
		if t.weakRefPool.HasPending() && t != t.gcThread {
			t.runPendingFinalizers()
		}
		t.currentCont = c
		next, err = c.RunInThread(t)
		if err != nil {
			next, err = t.contAfterError(c, err)
			if err != nil {
				return err
			}
		}
		c = next
	}
	return
}

// contAfterError returns the continuation to run after c has failed with
// err, or the error that the loop ends with if there is none.
//
// An error first goes to the message handler, where it happens: the
// continuation is the call of the handler.  Once it has been through that
// (it is then "handled") it goes to the protected call that c is part of, if
// there is one in this loop, and the continuation is where that call
// returns.  If there is none the loop ends, and the Go code that started it
// gets the error.
//
// A handler that fails is given its own error, and so on, a limited number
// of times.
func (t *Thread) contAfterError(c Cont, err error) (Cont, error) {
	rtErr := ToError(err)
	if !rtErr.Handled() {
		rtErr = rtErr.AddContext(c, -1)
		handler, handlerDepth := t.messageHandlerFor(c)
		switch {
		case handlerDepth >= maxErrorsInMessageHandler:
			rtErr = newHandledError(errErrorInMessageHandler)
		case handler == nil:
			// Nothing to call: the error is handled as it is, which is
			// what running this continuation does.
			next := newMessageHandlerCont(c)
			next.Push(t.Runtime, ErrorValue(rtErr))
			return next, nil
		default:
			next := handler.Continuation(t, newMessageHandlerCont(c))
			next.Push(t.Runtime, ErrorValue(rtErr))
			return next, nil
		}
	}
	if p := protectedCallFor(c); p != nil {
		return p.caught(t, rtErr), nil
	}
	return nil, rtErr
}

// This is to be able to close a suspended coroutine without completing it, but
// still allow cleaning up the to-be-closed variables.  If this is put on the
// resume channel of a running thread, yield will cause a panic in the goroutine
// and that will be caught in the defer() clause below.
type threadClose struct{}

//
// Coroutine management
//

// startGoroutine starts the goroutine that the thread runs in from now on,
// which goes on with the continuation c.  If isCall is true c is the call of
// the thread's body, and if not the thread had got further than that.
func (t *Thread) startGoroutine(c Cont, isCall bool) {
	t.ensureResumeCh()
	t.RequireBytes(2 << 10) // A goroutine starts off with 2k stack
	go func() {
		var (
			args []Value
			err  error
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
		err = t.runContinuation(c, isCall)
		args = t.term.Etc()
	}()
}

// Status returns the status of a thread (suspended, running or dead).
func (t *Thread) Status() ThreadStatus {
	return t.status
}

// Resume execution of a suspended thread.  Its status switches to
// running while its caller's status switches to suspended.
func (t *Thread) Resume(caller *Thread, args []Value) ([]Value, error) {
	if t.light {
		return t.resumeLight(caller, args)
	}
	t.mux.Lock()
	if t.status != ThreadSuspended {
		t.mux.Unlock()
		switch t.status {
		case ThreadDead:
			return nil, errors.New("cannot resume dead thread")
		default:
			return nil, errors.New("cannot resume running thread")
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
	caller.ensureResumeCh()
	t.sendResumeValues(args, nil, nil)
	return caller.getResumeValues()
}

// Close a suspended thread.  If successful, its status switches to dead.  The
// boolean returned is true if it was possible to close the thread (i.e. it was
// suspended or already dead).  The error is non-nil if there was an error in
// the cleanup process, or if the thread had already stopped with an error
// previously.
func (t *Thread) Close(caller *Thread) (bool, error) {
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
	if t.light {
		_ = t.endLight(nil)
		t.mux.Unlock()
		caller.mux.Unlock()
		return true, nil
	}
	caller.ensureResumeCh()
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

var errYieldAcrossGo = errors.New("attempt to yield across a Go-call boundary")

// Yield to the caller thread.  The yielding thread's status switches to
// suspended.  The caller's status must be OK.
func (t *Thread) Yield(args []Value) ([]Value, error) {
	if t.status == ThreadDead || t.noYieldDepth > 0 {
		// Go code that cannot be suspended is waiting for the code that
		// yields: a function declared NoYieldAcross, a debug hook, or the
		// thread's own ending, which calls the __close handlers of the
		// values a dead thread had left to close.
		return nil, errYieldAcrossGo
	}
	if t.light {
		// Only a Go function declared NoYieldAcross gets here.
		return nil, errors.New("a Go function declared NoYieldAcross must yield with YieldFrom")
	}
	t.mux.Lock()
	if t.status != ThreadOK {
		panic("Thread to yield is not running")
	}
	caller := t.caller
	if caller == nil {
		t.mux.Unlock()
		return nil, errors.New("cannot yield from main thread")
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
func (t *Thread) end(args []Value, err error, exception interface{}) {
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
	// Before the caller gets the values: from then on it is the caller that
	// runs, and this goroutine must touch nothing they share.
	t.ReleaseBytes(2 << 10) // The goroutine will terminate after this
	caller.sendResumeValues(args, err, exception)
}

func (t *Thread) call(c Callable, args []Value, next Cont) error {
	cont := c.Continuation(t, next)
	t.Push(cont, args...)
	return t.RunContinuation(cont)
}

func (t *Thread) getResumeValues() ([]Value, error) {
	res := <-t.resumeCh
	if res.exception != nil {
		panic(res.exception)
	}
	return res.args, res.err
}

func (t *Thread) sendResumeValues(args []Value, err error, exception interface{}) {
	t.resumeCh <- valuesError{args: args, err: err, exception: exception}
}

//
// Calling
//

// CallContext pushes a new runtime context on the thread's runtime and attempts
// to run f() in the thread.  If the context runs out of resources while f() is
// running, all operations should abort and the CallContext should return
// immediately and not finalizing pending to-be-closed values.
//
// Otherwise (even if f() returns an error), pending to-be-closed values should
// be finalized.
//
// See quotas.md for details about this API.
func (t *Thread) CallContext(def RuntimeContextDef, f func() error) (ctx RuntimeContext, err error) {
	t.PushContext(def)
	c, h := t.CurrentCont(), t.closeStack.size()
	t.startCont = c
	protectedCallDepth := t.protectedCallDepth
	defer func() {
		// The protected calls that f made are over, also those that a
		// termination of the context went through.
		t.protectedCallDepth = protectedCallDepth
		ctx = t.PopContext()
		if r := recover(); r != nil {
			t.closeStack.truncate(h) // No resources to run that, so just discard it.
			termErr, ok := r.(ContextTerminationError)
			if !ok {
				panic(r)
			}
			err = termErr
		}
	}()
	err = t.cleanupCloseStack(c, h, f())
	if t.GCPolicy() == IsolateGCPolicy {
		t.runFinalizers(t.weakRefPool.ExtractAllMarkedFinalize())
	}
	if err != nil {
		t.setStatus(StatusError)
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

// pushToBeClosed puts v on the close stack: it is the value of a variable
// declared <close>.  It is an error if v has no __close metamethod, unless it
// is nil or false, which are closed by doing nothing.
func (t *Thread) pushToBeClosed(v Value) error {
	if Truth(v) {
		if t.metaGet(v, metaClose).IsNil() {
			return errors.New("to be closed value missing a __close metamethod")
		}
		if t.light && t.noYieldDepth == 0 {
			// Go code calls its handler, which may yield: something a
			// light thread cannot have Go code waiting for.
			return errNeedsGoroutine
		}
	}
	t.closeStack.push(v)
	return nil
}

// Truncate the close stack to size h, calling the __close metamethods in the
// context of the given continuation c and feeding them with the given error.
func (t *Thread) cleanupCloseStack(c Cont, h int, err error) error {
	closeStack := &t.closeStack
	for closeStack.size() > h {
		v, _ := closeStack.pop()
		if Truth(v) {
			// Lua 5.5: __close receives 1 arg (the object) when closing normally,
			// and 2 args (the object and the error) when closing due to an error.
			var args []Value
			if err == nil {
				args = []Value{v}
			} else {
				args = []Value{v, ErrorValue(err)}
			}
			closeErr, ok := metacall(t, v, metaClose, args, NewTerminationWith(c, 0, false))
			if !ok {
				return errors.New("to be closed value missing a __close metamethod")
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

func (c *messageHandlerCont) RunInThread(t *Thread) (Cont, error) {
	return nil, newHandledError(c.err)
}
