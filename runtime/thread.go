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

	// A thread that has been given something to run by Start is light: it
	// has no goroutine, and Resume runs it in a loop of its own on the
	// goroutine of the thread that resumed it, which returns when it yields.
	// That is possible as long as nothing of the thread is on the Go stack
	// when it yields, so a light thread never runs a Go function that can
	// call Lua, nor holds a value to close, nor has a debug hook: before it
	// does it is given a goroutine (startGoroutine) and is light no more.
	// See notes/coroutines.md.
	//
	// A light thread is in one of these states:
	//
	//	suspended, body != nil          not started: Resume starts the body
	//	suspended, suspendedIn != nil   yielded: Resume returns from the yield
	//	running                         run by a Resume, on the goroutine of the
	//	                                thread that called it
	//
	// and it has none of them once it is dead, or has a goroutine.  Only
	// takeResumption, suspendAt and die change these fields.
	light       bool
	body        Callable
	suspendedIn *GoCont
	term        *Termination // Receives what the body returns
}

// NewThread creates a new thread out of a Runtime.  Its initial
// status is suspended.  Call Resume to run it.
func NewThread(r *Runtime) *Thread {
	r.RequireSize(unsafe.Sizeof(Thread{}) + 100) // 100 is my guess at the size of a channel
	return &Thread{
		status:  ThreadSuspended,
		Runtime: r,
	}
}

// ensureResumeCh makes the channel that the thread waits on for values from
// another thread, if it has none.  A thread needs it from when it has a
// goroutine, or waits for one that has: a light thread does neither.
func (t *Thread) ensureResumeCh() {
	if t.resumeCh == nil {
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
				if err = askForGoroutine(c, err); isNeedsGoroutine(err) {
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

// Start gives the thread the callable c to run when it is first resumed.  The
// t.Resume() method needs to be called to provide arguments to the callable.
// The thread is light (see Thread) until it needs a goroutine.
func (t *Thread) Start(c Callable) {
	t.light = true
	t.body = c
}

// HasGoroutine is true if the thread runs in a goroutine of its own, false
// if it is light and runs in the goroutine of whoever resumes it.
func (t *Thread) HasGoroutine() bool {
	return !t.light
}

// startGoroutine gives the thread the goroutine it runs in from now on.  The
// goroutine runs the continuation that first returns, after waiting for a
// resume if wait is true: first is then given the values of that resume.
func (t *Thread) startGoroutine(wait bool, first func(args []Value) Cont) {
	t.light = false
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
	// run again, or a Go function that was declared NoYieldAcross.
	errLuaCallRefused = errors.New("call to Lua refused")
)

// needsGoroutine is the error with which the loop of a light thread returns
// to Resume when the continuation c, which has not begun, cannot run until
// the thread has a goroutine.  It is what the goroutine runs first.
type needsGoroutine struct {
	c Cont
}

func (*needsGoroutine) Error() string {
	return "needs a goroutine"
}

func isNeedsGoroutine(err error) bool {
	_, ok := err.(*needsGoroutine)
	return ok
}

// askForGoroutine is called by the loop of a light thread when the
// continuation c returns errNeedsGoroutine or errLuaCallRefused.  It returns
// a needsGoroutine error, or the error to raise if the thread cannot be given
// a goroutine to run c.
func askForGoroutine(c Cont, err error) error {
	if _, isLua := c.(*LuaCont); err == errLuaCallRefused && !isLua {
		// The call was refused to a Go function already under way.
		return fmt.Errorf("Go function %s was declared NoYieldAcross, but called Lua", c.DebugInfo().Name)
	}
	return &needsGoroutine{c}
}

// maxResumeDepth is how deep resumes of light threads can nest: each is a
// loop on the Go stack of the one that resumed it.
const maxResumeDepth = 200

// resumeLight runs the light thread t, which is suspended, with the given
// values, until it yields or ends.  If it comes to need a goroutine on the
// way, it gets one and the wait is for that.
func (t *Thread) resumeLight(caller *Thread, args []Value) ([]Value, error) {
	if t.Runtime.resumeDepth >= maxResumeDepth {
		return nil, errors.New("stack overflow")
	}
	t.caller = caller
	t.status = ThreadOK
	if t.DebugHooks.DebugHookFlags != 0 {
		// A hook is called from Go: this is a thread for a goroutine.
		t.startGoroutine(true, t.takeResumption)
		return t.handOver(caller, args)
	}
	err := t.runLight(t.takeResumption(args))
	switch {
	case err == errThreadYield:
		return t.yielded(), nil
	case isNeedsGoroutine(err):
		return t.moveToGoroutine(caller, err.(*needsGoroutine).c)
	}
	// The thread has run to its end or died of an error.
	t.die(err)
	return t.term.Etc(), err
}

// takeResumption returns the continuation that a resume of the light thread t
// with the given values makes run: the body of the thread, if it has not
// started, or else what comes after the yield it is suspended in.
func (t *Thread) takeResumption(args []Value) Cont {
	if yc := t.suspendedIn; yc != nil {
		t.suspendedIn = nil
		return yc.resumeWith(t, args)
	}
	body := t.body
	t.body = nil
	t.term = NewTerminationWith(t.CurrentCont(), 0, true)
	c := body.Continuation(t, t.term)
	t.Push(c, args...)
	return c
}

// runLight runs the continuation c in the loop of the light thread t, until
// it returns.  It returns the error with which the loop returned: nil if the
// body returned, errThreadYield if the thread yielded, a needsGoroutine error
// if it needs a goroutine.
func (t *Thread) runLight(c Cont) error {
	r := t.Runtime
	r.resumeDepth++
	returned := false
	defer func() {
		r.resumeDepth--
		if !returned {
			// A panic is on its way through: the context has run out of
			// resources.  The thread is dead and closes nothing.
			t.die(nil)
		}
	}()
	err := t.RunContinuation(c)
	returned = true
	return err
}

// suspendAt suspends the running light thread t in the call c of yield.
func (t *Thread) suspendAt(c *GoCont) {
	t.suspendedIn = c
	t.status = ThreadSuspended
	t.caller = nil
}

// yielded is called when the loop of the light thread t has returned
// because it yielded.  It returns the values yielded.
func (t *Thread) yielded() []Value {
	// The loop has put the current continuation back to where it found it
	// but the thread has stopped in the call to yield, which a traceback
	// should show.
	t.currentCont = t.suspendedIn
	return t.suspendedIn.Etc()
}

// moveToGoroutine gives the running light thread t, which was resumed by
// caller, a goroutine that starts by running c.  The caller waits for the
// thread to yield or end, as it does for any thread that has a goroutine.
func (t *Thread) moveToGoroutine(caller *Thread, c Cont) ([]Value, error) {
	t.startGoroutine(false, func([]Value) Cont { return c })
	return caller.getResumeValues()
}

// die ends the light thread t with the given error, if any.  It has nothing
// to close: a light thread holds no value with a __close metamethod.
func (t *Thread) die(err error) {
	t.status = ThreadDead
	t.caller = nil
	t.light = false
	t.body, t.suspendedIn = nil, nil
	t.closeStack.truncate(0)
	t.closeErr = err
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
	t.suspendAt(c)
	return nil, errThreadYield
}

// Status returns the status of a thread (suspended, running or dead).
func (t *Thread) Status() ThreadStatus {
	return t.status
}

// Resume execution of a suspended thread.  Its status switches to
// running while its caller's status switches to suspended.
func (t *Thread) Resume(caller *Thread, args []Value) ([]Value, error) {
	caller.ensureResumeCh() // It may have to wait for a goroutine
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
		t.die(nil)
		t.mux.Unlock()
		caller.mux.Unlock()
		return true, nil
	}
	caller.ensureResumeCh() // It has to wait for the goroutine
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
