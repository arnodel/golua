package runtime

import (
	"errors"
	"unsafe"
)

// A ContFunction is a function that stands for a continuation: calling it
// makes the continuation carry on with the arguments of the call as the
// values pushed into it.  It is made by Thread.CaptureCont and can be called
// any number of times.  To Lua it is a function, next to Closure and
// GoFunction.
//
// A Cont cannot do this by itself.  Running a frame changes its registers
// and its place in the code, so a frame runs from a given state once.  A
// captured frame is frozen instead: it never runs again, and what runs is a
// copy of it, made when something is about to push values into it (see
// frozenCont).  Nothing is copied at the capture.
//
// A copy of a frame has the frame's registers, so the same values.  A
// variable that lives in a register is therefore as it was at the capture
// each time the continuation is called, whereas what the frame shares with
// the rest of the program is not copied and keeps its changes: tables, and
// the boxes of captured variables.
//
// See notes/continuations.md for the design.
type ContFunction struct {
	loop *runLoop // The loop k was captured in, where it can carry on
	cont Cont     // A frozen frame, or a continuation that is not a frame

	goFunction *GoFunction // Calls k, for Go code; made when first needed
}

var _ Callable = (*ContFunction)(nil)

// A runLoop is one call of Thread.RunContinuation, which runs continuations
// one after the other until one of them ends the loop.
//
// A Lua call or return is one turn of the loop that is running, and so is
// the call of a Go function that returns the continuation to run next.  A Go
// function that needs the results of Lua code before it can go on (pcall, a
// metamethod call, the comparison function of table.sort, the body of a
// coroutine) starts a loop of its own and waits for it to end.  So the loops
// of a thread are nested as the Go calls that started them are, and a frame
// belongs to one loop: the one that ends when the frames it returns through
// have all returned.
//
// A ContFunction keeps the loop it was captured in because that is the only
// place where the frames it holds can carry on: the Go code that waits for
// them to return is not part of what it holds.  A thread makes the runLoop
// of a loop when a continuation is first captured in it (see Thread.loop).
type runLoop struct {
	thread *Thread
}

// CaptureCont returns a function k that stands for next, and the
// continuation to use in place of next from now on.
//
// If next is a frame, it is frozen: it must not run any more, because
// running it would change what k holds.  The continuation returned is the
// frozen frame, which is copied when values are pushed into it, as it is
// each time k is called.  This is how a Go function gives its caller's
// continuation to Lua code (call/cc):
//
//	k, next := t.CaptureCont(c.Next())
//	// then call f with k as its argument, returning to next
//
// k carries on in the loop that is running and in no other (see runLoop), so
// next must be a continuation of that loop.  The next continuation of the
// Go function that is running is one.
func (t *Thread) CaptureCont(next Cont) (Value, Cont) {
	if lc, ok := next.(*LuaCont); ok {
		next = (*frozenCont)(lc)
	}
	if t.loop == nil {
		t.RequireSize(unsafe.Sizeof(runLoop{}))
		t.loop = &runLoop{thread: t}
	}
	t.RequireSize(unsafe.Sizeof(ContFunction{}))
	k := &ContFunction{loop: t.loop, cont: next}
	return FunctionValue(k), next
}

// reinstate returns a continuation that carries on the computation k was
// captured from, for its caller to push values into and run in place of the
// running one.  The thread must be in the loop k was captured in.
func (k *ContFunction) reinstate(t *Thread) Cont {
	if fc, ok := k.cont.(*frozenCont); ok {
		return fc.thaw(t)
	}
	return k.cont
}

// Continuation implements Callable.Continuation.  The call is that of a Go
// function, k.call, which decides what to do when it runs: Go code may well
// run what it gets here in a loop of its own.  Lua code calling k in the
// loop k was captured in does not come this way, see continueCall.
func (k *ContFunction) Continuation(t *Thread, next Cont) Cont {
	if k.goFunction == nil {
		t.RequireSize(unsafe.Sizeof(GoFunction{}))
		k.goFunction = &GoFunction{
			f:           k.call,
			name:        "continuation",
			hasEtc:      true,
			safetyFlags: ComplyCpuSafe | ComplyMemSafe | ComplyTimeSafe | ComplyIoSafe,
		}
	}
	return NewGoCont(t, k.goFunction, next)
}

// call calls k with the arguments of c.  k carries on in the loop it was
// captured in and nowhere else (see runLoop):
//
//   - In that loop, the frame k holds runs next.
//   - In another thread it is not this thread's to run, which is an error.
//   - Anywhere else in its thread, the Go code that waits for the frames of
//     k to return has returned itself, or other Go code has called back into
//     Lua since and is waiting in its turn.  Both are errors.
func (k *ContFunction) call(t *Thread, c *GoCont) (Cont, error) {
	switch {
	case t.loop == k.loop:
		cont := k.reinstate(t)
		cont.PushEtc(t.Runtime, c.Etc())
		return cont, nil
	case t != k.loop.thread:
		return nil, errors.New("attempt to call a continuation from another thread")
	}
	return nil, errors.New("attempt to call a continuation across a Go call boundary")
}
