package runtime

import (
	"errors"
	"fmt"
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

	// Depth of __call metamethod chain (Lua 5.5).  This should not exceed
	// maxCallChainLength to prevent infinite loops through chained __call metamethods.
	metacallChainDepth int

	DebugHooks

	closeStack // Stack of pending to-be-closed values

	// A coroutine starts light: it has no goroutine, and Resume runs it in
	// a loop of its own on the resumer's goroutine, which returns when it
	// yields.  That is possible as long as nothing of the coroutine's is on
	// the Go stack when it yields, so a light thread never has a Go
	// function under way that can call Lua, a value to close, or a debug
	// hook: before it gets one it is given a goroutine (startGoroutine) and
	// is light no more.  See notes/coroutines.md.
	light       bool
	body        Callable     // light, never resumed: what it will run
	term        *Termination // receives what the body returns
	suspendedIn *GoCont      // light, suspended: the call of yield waiting for the next resume
	rerun       Cont         // light, asking for a goroutine: the continuation to run on it
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
func (t *Thread) RunContinuation(c Cont) (err error) {
	if t.light && t.runContinuationDepth > 0 {
		// Go code of a light thread is calling Lua: the thread needs a
		// goroutine first.  Nothing of the call has run yet.
		return errLuaCallRefused
	}
	if t.runContinuationDepth >= maxRunContinuationDepth {
		return errors.New("stack overflow")
	}
	// On the way out, put back the continuation the thread was running.
	// Otherwise it stays on the termination this loop ended on, and the next
	// call from Go makes its own termination with that one as parent: a
	// chain that grows with every call and keeps all their results alive.
	//
	// Not when a coroutine's own loop ends: the coroutine is dead, and its
	// traceback shows the continuation it stopped in.
	current := t.currentCont
	t.runContinuationDepth++
	defer func() {
		t.runContinuationDepth--
		coroutineEnds := t.runContinuationDepth == 0 && t.caller != nil
		if !coroutineEnds {
			t.currentCont = current
		}
	}()

	var next Cont
	var errContCount = 0
	_ = t.triggerCall(t, c)
	for c != nil {
		// Checked at every continuation, so twice per call: the check is
		// one load, inline, and the work only when there is some.
		if t.weakRefPool.HasPending() && t != t.gcThread {
			t.runPendingFinalizers()
		}
		t.currentCont = c
		next, err = c.RunInThread(t)
		if err != nil {
			switch err {
			case errThreadYield:
				// A light thread has yielded: back to Resume.
				return err
			case errNeedsGoroutine, errLuaCallRefused:
				// A light thread needs a goroutine before c can run: back
				// to Resume, which gives it one.
				if err = t.askForGoroutine(c, err); err == errNeedsGoroutine {
					return err
				}
			}
			rtErr := ToError(err)
			if rtErr.Handled() {
				return rtErr
			}
			err = rtErr.AddContext(c, -1)
			errContCount++
			if t.messageHandler != nil {
				if errContCount > maxErrorsInMessageHandler {
					return newHandledError(errErrorInMessageHandler)
				}
				next = t.messageHandler.Continuation(t, newMessageHandlerCont(c))
			} else {
				next = newMessageHandlerCont(c)
			}
			next.Push(t.Runtime, ErrorValue(err))
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
	t.startGoroutine(true, func(args []Value) Cont {
		t.term = NewTerminationWith(t.CurrentCont(), 0, true)
		cont := c.Continuation(t, t.term)
		t.Push(cont, args...)
		return cont
	})
}

// startGoroutine starts the goroutine the thread runs in from now on.  The
// goroutine runs the continuation that first returns, after waiting for a
// resume if wait is true: first is then given the values of that resume.
func (t *Thread) startGoroutine(wait bool, first func(args []Value) Cont) {
	t.light = false
	if t.resumeCh == nil {
		t.resumeCh = make(chan valuesError)
	}
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
		if wait {
			args, err = t.getResumeValues()
		}
		if err == nil {
			err = t.RunContinuation(first(args))
			args = t.term.Etc()
		}
	}()
}

//
// Light threads
//

var (
	// errThreadYield is what coroutine.yield returns in a light thread.  It
	// makes the thread's loop return, to Resume.
	errThreadYield = errors.New("yield")

	// errNeedsGoroutine comes from something a light thread cannot do before
	// it has a goroutine, and that has not begun: a Go function that may
	// call Lua, a value to close.
	errNeedsGoroutine = errors.New("needs a goroutine")

	// errLuaCallRefused is returned by RunContinuation to Go code of a light
	// thread that calls Lua: a metamethod called by an opcode, which can be
	// run again, or a Go function that was declared never to call Lua.
	errLuaCallRefused = errors.New("call to Lua refused")
)

// maxResumeDepth is how deep resumes of light threads can nest: each is a
// loop on the Go stack of the one that resumed it.
const maxResumeDepth = 200

// NewCoroutine returns a new thread that will run body when it is first
// resumed.  Its status is suspended.  It is light: it has no goroutine until
// it needs one.
func NewCoroutine(r *Runtime, body Callable) *Thread {
	r.RequireSize(unsafe.Sizeof(Thread{}))
	return &Thread{
		status:  ThreadSuspended,
		Runtime: r,
		light:   true,
		body:    body,
	}
}

// HasGoroutine is true if the thread runs in a goroutine of its own, false
// if it is light and runs in the goroutine of whoever resumes it.
func (t *Thread) HasGoroutine() bool {
	return !t.light
}

// askForGoroutine is called by the loop of a light thread when the
// continuation c returns errNeedsGoroutine or errLuaCallRefused.  It returns
// errNeedsGoroutine once it has noted that c is to be run again on a
// goroutine, or the error to raise if it cannot be.
func (t *Thread) askForGoroutine(c Cont, err error) error {
	if _, isLua := c.(*LuaCont); err == errLuaCallRefused && !isLua {
		// The call was refused to a Go function already under way.
		return fmt.Errorf("a Go function declared never to call Lua called it: %s", c.DebugInfo().Name)
	}
	t.rerun = c
	return errNeedsGoroutine
}

// resumeLight runs the light thread t, which is suspended, with the given
// values, until it yields or ends.  If it comes to need a goroutine on the
// way, it gets one and the wait is for that.
func (t *Thread) resumeLight(caller *Thread, args []Value) (res []Value, err error) {
	r := t.Runtime
	if r.resumeDepth >= maxResumeDepth {
		return nil, errors.New("stack overflow")
	}
	if t.DebugHooks.DebugHookFlags != 0 {
		// A hook is called from Go: this is a thread for a goroutine.
		if yc := t.suspendedIn; yc != nil {
			t.suspendedIn = nil
			t.startGoroutine(true, func(args []Value) Cont { return yc.resumeWith(t, args) })
		} else {
			t.Start(t.body)
			t.body = nil
		}
		t.caller = caller
		t.status = ThreadOK
		return t.handOver(caller, args)
	}
	var c Cont
	if yc := t.suspendedIn; yc != nil {
		t.suspendedIn = nil
		c = yc.resumeWith(t, args)
	} else {
		t.term = NewTerminationWith(nil, 0, true)
		c = t.body.Continuation(t, t.term)
		t.body = nil
		t.Push(c, args...)
	}
	t.caller = caller
	t.status = ThreadOK
	r.resumeDepth++
	returned := false
	defer func() {
		r.resumeDepth--
		if !returned {
			// A panic is on its way through: the context has run out of
			// resources.  The thread is dead and closes nothing.
			t.status = ThreadDead
			t.caller = nil
			t.light = false
			t.closeStack.truncate(0)
		}
	}()
	err = t.RunContinuation(c)
	returned = true
	switch err {
	case errThreadYield:
		t.currentCont = t.suspendedIn
		return t.suspendedIn.Etc(), nil
	case errNeedsGoroutine:
		c, t.rerun = t.rerun, nil
		if caller.resumeCh == nil {
			caller.resumeCh = make(chan valuesError)
		}
		// From here the wait is the one of a thread with a goroutine.
		t.startGoroutine(false, func([]Value) Cont { return c })
		return caller.getResumeValues()
	}
	// The thread has run to its end or died of an error.  It has nothing to
	// close but nil and false, which have no handler.
	t.status = ThreadDead
	t.caller = nil
	t.light = false
	t.closeStack.truncate(0)
	t.closeErr = err
	return t.term.Etc(), err
}

// YieldFrom yields from the Go function whose continuation is c, called by
// the thread t, with the arguments of c.  The Go function must return what
// YieldFrom returns: the values of the next resume go to c.Next().
func (t *Thread) YieldFrom(c *GoCont) (Cont, error) {
	if !t.light {
		res, err := t.Yield(c.Etc())
		if err != nil {
			return nil, err
		}
		return c.PushingNext(t.Runtime, res...), nil
	}
	// A light thread is running, so it was resumed; and none of its Go
	// functions that could call Lua is under way, so nothing is in the way.
	t.suspendedIn = c
	t.status = ThreadSuspended
	t.caller = nil
	return nil, errThreadYield
}

// Status returns the status of a thread (suspended, running or dead).
func (t *Thread) Status() ThreadStatus {
	return t.status
}

// Resume execution of a suspended thread.  Its status switches to
// running while its caller's status switches to suspended.
func (t *Thread) Resume(caller *Thread, args []Value) ([]Value, error) {
	if t.light {
		// No other goroutine has to do with a light thread.
		if t.status != ThreadSuspended {
			return nil, errors.New("cannot resume running thread")
		}
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
	return t.handOver(caller, args)
}

// handOver gives the values of a resume to the thread's goroutine and waits
// for what the thread yields or returns.
func (t *Thread) handOver(caller *Thread, args []Value) ([]Value, error) {
	if caller.resumeCh == nil {
		// The caller is a light thread, which has had nothing to wait for
		// until now.
		caller.resumeCh = make(chan valuesError)
	}
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
		// A light thread has nothing to close but nil and false.
		t.status = ThreadDead
		t.light = false
		t.body, t.suspendedIn = nil, nil
		t.closeStack.truncate(0)
		t.mux.Unlock()
		caller.mux.Unlock()
		return true, nil
	}
	if caller.resumeCh == nil {
		caller.resumeCh = make(chan valuesError)
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

var errYieldAcrossGo = errors.New("attempt to yield across a Go-call boundary")

// Yield to the caller thread.  The yielding thread's status switches to
// suspended.  The caller's status must be OK.
func (t *Thread) Yield(args []Value) ([]Value, error) {
	if t.status == ThreadDead {
		// A dead thread still runs code: the __close handlers of the
		// values it had left to close.  They are called from Go, by the
		// thread's own ending, which cannot be suspended.
		return nil, errYieldAcrossGo
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
	caller.sendResumeValues(args, err, exception)
	t.ReleaseBytes(2 << 10) // The goroutine will terminate after this
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
	defer func() {
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
		if t.light {
			// Its handler will be called from Go, perhaps when the
			// function has half returned: not something to run again.
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

// messageHandlerCont is a continuation that handles an error message (i.e.
// turns it to handled).
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
