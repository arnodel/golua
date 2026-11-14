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

	// The RunContinuation loop that is running, for the continuations
	// captured in it; nil until there is one.
	loop *runLoop

	// Depth of __call metamethod chain (Lua 5.5).  This should not exceed
	// maxCallChainLength to prevent infinite loops through chained __call metamethods.
	metacallChainDepth int

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
func (t *Thread) RunContinuation(c Cont) (err error) {
	if t.runContinuationDepth >= maxRunContinuationDepth {
		return errors.New("stack overflow")
	}
	t.runContinuationDepth++
	outer, closeStackBase := t.loop, t.closeStack.size()
	t.loop = nil
	defer func() {
		t.runContinuationDepth--
		if t.loop != nil {
			t.loop.done = true
		}
		t.loop = outer
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
			if call, ok := err.(*continuationCall); ok {
				// A continuation was called under Go code that this loop
				// was waiting for, which has now returned.  If it was
				// captured in this loop, it carries on here.  If not, this
				// loop is itself under Go code that the continuation's
				// loop is waiting for: it finishes, closing the variables
				// to be closed of the frames it leaves behind.
				if call.k.loop != t.loop {
					return t.cleanupCloseStack(c, closeStackBase, err)
				}
				if next, err = call.k.resume(t); err == nil {
					next.PushEtc(t.Runtime, call.args)
					c = next
					continue
				}
				// Closing a variable failed: that is an error like any.
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
func (t *Thread) Resume(caller *Thread, args []Value) ([]Value, error) {
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
func (t *Thread) Yield(args []Value) ([]Value, error) {
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

// A closeStack is the stack of the values to be closed that are pending in a
// thread.  Its height counts every value pushed, and it only holds those
// that are to be closed in fact, not nil or false: each knows the height of
// the stack it is the top of.
//
// The values are linked from the top down rather than held in an array, so
// that a continuation can keep the stack as it was when it was captured (see
// switchCloseStack).  With continuations the pending values of a thread form
// a tree, of which the thread's closeStack is the branch that is running.
type closeStack struct {
	top    *closeNode
	height int
}

// A closeNode is a value to be closed and the ones pushed before it.
type closeNode struct {
	value  Value
	height int // Height of the stack with this value on top
	next   *closeNode

	// A continuation was captured with this value pending: the block it is
	// in can be entered again.
	captured bool

	// The value was closed.  Copies of a frame share it, and it is closed
	// once.
	closed bool
}

func (s closeStack) size() int {
	return s.height
}

// truncate brings the stack down to height h without closing anything.
func (s *closeStack) truncate(h int) {
	for s.top != nil && s.top.height > h {
		s.top = s.top.next
	}
	if s.height > h {
		s.height = h
	}
}

// pushClose pushes v on the thread's close stack.
func (t *Thread) pushClose(v Value) {
	s := &t.closeStack
	s.height++
	if Truth(v) {
		t.RequireSize(unsafe.Sizeof(closeNode{}))
		s.top = &closeNode{value: v, height: s.height, next: s.top}
	}
}

// Truncate the close stack to size h, calling the __close metamethods in the
// context of the given continuation c and feeding them with the given error.
func (t *Thread) cleanupCloseStack(c Cont, h int, err error) error {
	s := &t.closeStack
	if s.height <= h {
		return err
	}
	for s.top != nil && s.top.height > h {
		n := s.top
		s.top, s.height = n.next, n.height-1
		closeErr, ok := t.closeValue(c, n, err)
		if !ok {
			return errors.New("to be closed value missing a __close metamethod")
		}
		if closeErr != nil {
			err = closeErr
		}
	}
	s.height = h
	return err
}

// closeValue calls the __close metamethod of the value of n, unless that
// was done already.  It returns the error of the call, and false if there is
// no metamethod to call.
func (t *Thread) closeValue(c Cont, n *closeNode, err error) (error, bool) {
	if n.closed {
		return nil, true
	}
	n.closed = true
	// Lua 5.5: __close receives 1 arg (the object) when closing normally,
	// and 2 args (the object and the error) when closing due to an error.
	var args []Value
	if err == nil || IsContinuationCall(err) {
		args = []Value{n.value}
	} else {
		args = []Value{n.value, ErrorValue(err)}
	}
	return metacall(t, n.value, metaClose, args, NewTerminationWith(c, 0, false))
}

// capture returns the close stack for a continuation to keep, with its
// pending values marked as captured.
func (s *closeStack) capture() closeStack {
	// A captured value has only captured values under it.
	for n := s.top; n != nil && !n.captured; n = n.next {
		n.captured = true
	}
	return *s
}

// switchCloseStack makes to, the close stack a continuation has kept, the
// thread's, as the continuation is about to run.
//
// The two stacks are branches of one tree: they have the values in common
// that were pending before they parted, and the blocks of the others are
// left by the switch.  A block that is left is over if no continuation was
// captured in it, as nothing can run it again, and then its value is closed.
// If one was, the block may be entered again, by calling that continuation:
// its value stays pending, and is closed when the block ends, as that of a
// coroutine which has yielded.
func (t *Thread) switchCloseStack(to closeStack) error {
	s := &t.closeStack
	if s.top != to.top {
		// Where the branches part: the first value they have in common.
		a, b := s.top, to.top
		for a != b {
			switch {
			case b == nil || (a != nil && a.height > b.height):
				a = a.next
			case a == nil || b.height > a.height:
				b = b.next
			default:
				a, b = a.next, b.next
			}
		}
		for s.top != a && !s.top.captured {
			n := s.top
			s.top, s.height = n.next, n.height-1
			closeErr, ok := t.closeValue(t.currentCont, n, nil)
			if !ok {
				return errors.New("to be closed value missing a __close metamethod")
			}
			if closeErr != nil {
				return closeErr
			}
		}
	}
	*s = to
	return nil
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
