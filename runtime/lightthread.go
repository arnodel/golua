package runtime

import (
	"errors"
	"fmt"
)

// # Light threads
//
// A thread that Start has given something to run is light: it has no
// goroutine.  Resuming it runs its continuations in a loop (runLight) on the
// goroutine of the thread that resumes it, and when it yields the loop
// returns.  So a resume and a yield cost a function call and its return, and
// a suspended thread is no more than its current continuation, which goes
// away when nothing refers to the thread.
//
// # What a light thread cannot do
//
// When the thread yields its loop returns, so at that point nothing of the
// thread can be left on the Go stack.  Go code that is waiting for Lua code
// it has called would be, if that Lua code yields.  So a light thread never
// has Go code calling Lua, in any of its four forms:
//
//   - a Go function, unless it was declared NoYieldAcross;
//   - a metamethod called by an opcode;
//   - the __close metamethod of a value to be closed;
//   - a debug hook.
//
// # Getting a goroutine
//
// Before a light thread does one of these it is given a goroutine
// (moveToGoroutine), for good: from then on it is resumed and it yields
// through channels, as thread.go has it.  The loop sees the first and the last
// of the four coming before it runs a continuation (lightCannotRun).  The
// other two are found while a continuation runs: it returns
// errNeedsGoroutine, having done nothing that running it again would repeat.
//
// # Stopping and going on
//
// The loop stops in one of three ways (lightStop), and in the first two the
// thread goes on by running its current continuation again:
//
//   - the thread yields: the current continuation is the call of yield.  A
//     resume runs it again, and this time it returns the values of the resume
//     (YieldFrom);
//   - the thread needs a goroutine: the current continuation is the one that
//     could not run, and the goroutine starts with it;
//   - the thread ends, because its body has returned or failed.

// lightStop is why the loop of a light thread has stopped.
type lightStop int

const (
	lightEnded          lightStop = iota // The body has returned or failed
	lightYielded                         // The thread is suspended in a call of yield
	lightNeedsGoroutine                  // The current continuation cannot run without one
)

var (
	// errLightYield is what a call of yield returns in a light thread, to
	// make the thread's loop return.
	errLightYield = errors.New("yield")

	// errNeedsGoroutine is what a continuation of a light thread returns when
	// it comes to something the thread cannot do.  It has not done it, and
	// can be run again.
	errNeedsGoroutine = errors.New("needs a goroutine")
)

// maxResumeDepth is how deep resumes of light threads can nest: each is a
// loop on the Go stack of the one that resumed it.
const maxResumeDepth = 200

// Start gives the thread the callable c to run when it is first resumed.  The
// t.Resume() method needs to be called to provide arguments to the callable.
// The thread is light: it has no goroutine until it needs one.
func (t *Thread) Start(c Callable) {
	t.light = true
	t.body = c
}

// HasGoroutine is true if the thread runs in a goroutine of its own, false
// if it is light and runs in the goroutine of whoever resumes it.
func (t *Thread) HasGoroutine() bool {
	return !t.light
}

//
// What a light thread cannot do
//

// lightCannotRun is true if the light thread t needs a goroutine before it
// runs c.
func (t *Thread) lightCannotRun(c Cont) bool {
	if t.DebugHookFlags != 0 {
		// The hook is called from Go.
		return true
	}
	// A Go function may call Lua, unless it says that no yield crosses it.
	g, isGo := c.(*GoCont)
	return isGo && !g.noYieldAcross
}

// The other two things are found where they happen: Go code calling Lua in
// runContinuation, and a value to be closed in pushToBeClosed.

//
// Resuming, running, yielding, ending
//

// resumeLight runs the light thread t, which caller resumes with the given
// values, until it yields or ends.  If it comes to need a goroutine on the
// way it gets one, and the wait is for that.
func (t *Thread) resumeLight(caller *Thread, args []Value) ([]Value, error) {
	if t.status != ThreadSuspended {
		return nil, errors.New("cannot resume running thread")
	}
	if t.resumeDepth >= maxResumeDepth {
		return nil, errors.New("stack overflow")
	}
	c, isCall := t.lightResumption(args)
	t.caller = caller
	t.status = ThreadOK
	if isCall && t.lightCannotRun(c) {
		// The thread cannot even start without a goroutine.
		return t.moveToGoroutine(caller, c, true)
	}
	switch stop, err := t.runLight(c); stop {
	case lightYielded:
		// What the call of yield was given.
		return t.currentCont.(*GoCont).Etc(), nil
	case lightNeedsGoroutine:
		return t.moveToGoroutine(caller, t.currentCont, false)
	default:
		t.endLight(err)
		return t.term.Etc(), err
	}
}

// lightResumption returns the continuation that a resume of the light thread
// t with the given values runs: the call of its body if the thread has not
// started (isCall is then true), or else the call of yield it is suspended
// in, which is made to return the values.
func (t *Thread) lightResumption(args []Value) (c Cont, isCall bool) {
	if body := t.body; body != nil {
		t.body = nil
		t.term = NewTerminationWith(nil, 0, true)
		c = body.Continuation(t, t.term)
		t.Push(c, args...)
		return c, true
	}
	t.resumed, t.resumedWith = true, args
	return t.currentCont, false
}

// runLight is the loop of the light thread t: it runs c and the
// continuations that follow until the thread stops.  The thread's current
// continuation is then where it stopped, and err the error it ended with if
// it has ended.
func (t *Thread) runLight(c Cont) (stop lightStop, err error) {
	t.resumeDepth++
	panicking := true
	defer func() {
		t.resumeDepth--
		if panicking {
			// The context has run out of resources.
			t.endLight(nil)
		}
	}()
	stop, err = t.runLightLoop(c)
	panicking = false
	return stop, err
}

func (t *Thread) runLightLoop(c Cont) (lightStop, error) {
	errCount := 0
	for c != nil {
		if t.weakRefPool.HasPending() {
			t.runPendingFinalizers()
		}
		t.currentCont = c
		if t.lightCannotRun(c) {
			return lightNeedsGoroutine, nil
		}
		next, err := c.RunInThread(t)
		switch err {
		case nil:
		case errLightYield:
			return lightYielded, nil
		case errNeedsGoroutine:
			if _, isLua := c.(*LuaCont); isLua {
				// The opcode that asks is the one c runs next.
				return lightNeedsGoroutine, nil
			}
			// A Go function has had its call of Lua refused, and running
			// it again would repeat what it did before that.
			err = fmt.Errorf("Go function %s was declared NoYieldAcross, but called Lua", c.DebugInfo().Name)
			fallthrough
		default:
			next, err = t.contAfterError(c, err, &errCount)
			if err != nil {
				return lightEnded, err
			}
		}
		c = next
	}
	return lightEnded, nil
}

// YieldFrom yields from the Go function whose continuation is c, called by
// the thread t, with the arguments of c.  The Go function must return what
// YieldFrom returns.
//
// In a light thread the function is called twice: the first time YieldFrom
// suspends the thread and stops its loop, and when the thread is resumed the
// call is run again, for YieldFrom to return the values of the resume.
func (t *Thread) YieldFrom(c *GoCont) (Cont, error) {
	if t.resumed {
		vals := t.resumedWith
		t.resumed, t.resumedWith = false, nil
		return c.PushingNext(t.Runtime, vals...), nil
	}
	if !t.light {
		res, err := t.Yield(c.Etc())
		if err != nil {
			return nil, err
		}
		return c.PushingNext(t.Runtime, res...), nil
	}
	t.status = ThreadSuspended
	t.caller = nil
	return nil, errLightYield
}

// endLight makes the light thread t dead, with the error it died of if any.
// It has nothing to close: a light thread holds no value with a __close
// metamethod.
func (t *Thread) endLight(err error) {
	t.status = ThreadDead
	t.caller = nil
	t.light = false
	t.body = nil
	t.closeStack.truncate(0)
	t.closeErr = err
}

//
// Getting a goroutine
//

// moveToGoroutine gives the running light thread t, which caller has resumed,
// a goroutine that goes on with c (see startGoroutine).  The caller waits for
// the thread to yield or end, as it does for any thread that has a goroutine.
func (t *Thread) moveToGoroutine(caller *Thread, c Cont, isCall bool) ([]Value, error) {
	t.light = false
	caller.ensureResumeCh()
	t.startGoroutine(c, isCall)
	return caller.getResumeValues()
}
