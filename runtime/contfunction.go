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
	cont Cont     // What k carries on with, as frozen returns it

	protectedCallDepth int // The thread's depth of protected calls at the capture

	goFunction *GoFunction // Calls k, for Go code; made when first needed
}

var _ Callable = (*ContFunction)(nil)

// A runLoop is a loop that runs continuations one after the other until one
// of them ends it: a call of Thread.RunContinuation, or the loop of a
// coroutine, which stops when the coroutine yields and goes on when it is
// resumed, if need be in a goroutine (see lightthread.go).
//
// A Lua call or return is one turn of the loop that is running, and so is
// the call of a Go function that returns the continuation to run next, as
// pcall does.  A Go function that needs the results of Lua code before it
// can go on (a metamethod call, the comparison function of table.sort)
// starts a loop of its own and waits for it to end.  So the loops of a
// thread are nested as the Go calls that started them are, and a frame
// belongs to one loop: the one that ends when the frames it returns through
// have all returned.
//
// A ContFunction keeps the loop it was captured in because that is the only
// place where the frames it holds can carry on: the Go code that waits for
// them to return is not part of what it holds.  A thread makes the runLoop
// of a loop when a continuation is first captured in it (see Thread.loop).
type runLoop struct {
	thread *Thread
	done   bool // The loop has ended
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
// If next is a protected call (pcall called the Go function), k holds one
// that nothing has returned into, and what next returns to is frozen.
//
// k carries on in the loop that is running and in no other (see runLoop), so
// next must be a continuation of that loop.  The next continuation of the
// Go function that is running is one.
func (t *Thread) CaptureCont(next Cont) (Value, Cont) {
	cont := frozen(t, next)
	switch c := next.(type) {
	case *LuaCont:
		next = cont
	case *protectedCall:
		c.freezeNext(t)
	}
	if t.loop == nil {
		t.RequireSize(unsafe.Sizeof(runLoop{}))
		t.loop = &runLoop{thread: t}
	}
	t.RequireSize(unsafe.Sizeof(ContFunction{}))
	k := &ContFunction{loop: t.loop, cont: cont, protectedCallDepth: t.protectedCallDepth}
	return FunctionValue(k), next
}

// reinstate returns a continuation that carries on the computation k was
// captured from, for its caller to push values into and run in place of the
// running one.  The thread must be in the loop k was captured in.  The
// protected calls under way are from now on those the computation is in.
func (k *ContFunction) reinstate(t *Thread) Cont {
	t.protectedCallDepth = k.protectedCallDepth
	switch c := k.cont.(type) {
	case *frozenCont:
		return c.thaw(t)
	case *protectedCall:
		return c.clone(t)
	}
	return k.cont
}

// Continuation implements Callable.Continuation.  The call is that of a Go
// function, k.call, and what the call does is decided when that runs and
// not here, where nothing of k is touched.  The arguments of the call are
// computed after this, by the computation that k is to replace.  And the
// loop that is running now need not be the one that runs what this returns:
// Go code may well run it in a loop of its own.
func (k *ContFunction) Continuation(t *Thread, next Cont) Cont {
	if k.goFunction == nil {
		t.RequireSize(unsafe.Sizeof(GoFunction{}))
		k.goFunction = &GoFunction{
			f:           k.call,
			name:        "continuation",
			hasEtc:      true,
			safetyFlags: ComplyCpuSafe | ComplyMemSafe | ComplyTimeSafe | ComplyIoSafe,

			noYieldAcross: true,
		}
	}
	return NewGoCont(t, k.goFunction, next)
}

// call calls k with the arguments of c.  k carries on in the loop it was
// captured in and nowhere else (see runLoop):
//
//   - In that loop, the frame k holds runs next.
//   - In another thread it is not this thread's to run, which is an error.
//   - Once that loop has ended, the Go code that waited for the frames of k
//     to return has returned itself, which is an error too.
//   - Otherwise the thread is under Go code which that loop is waiting for
//     (a metamethod call, table.sort calling its comparison function), and
//     that Go code has to return first.  The call returns as an error does,
//     a contFunctionCall, which Go code passes on as it does errors and the
//     loop takes up (see Thread.contAfterError).
func (k *ContFunction) call(t *Thread, c *GoCont) (Cont, error) {
	switch {
	case t.loop == k.loop:
		cont := k.reinstate(t)
		cont.PushEtc(t.Runtime, c.Etc())
		return cont, nil
	case t != k.loop.thread:
		return nil, errors.New("attempt to call a continuation from another thread")
	case k.loop.done:
		return nil, errors.New("attempt to call a continuation captured in a Go call that has returned")
	}
	// The call keeps a copy of the arguments, which are c's.
	t.RequireArrSize(unsafe.Sizeof(Value{}), len(c.Etc()))
	return nil, &contFunctionCall{k: k, args: append([]Value(nil), c.Etc()...)}
}

// A contFunctionCall is the call of a ContFunction on its way to the loop
// that can run it, through the Go code that loop is waiting for.  It is an
// error value so that this Go code returns as it does for an error, but
// nothing has failed.
type contFunctionCall struct {
	k    *ContFunction
	args []Value
}

// Error implements error.  This is what a Go function gets if it makes a
// message of the errors of the Lua code it calls, instead of returning them.
func (*contFunctionCall) Error() string {
	return "attempt to call a continuation across a Go function that does not let it through"
}

// IsContFunctionCall reports whether err is the call of a ContFunction on
// its way through Go code (see Thread.contAfterError).  A Go function that
// turns the errors of the Lua code it calls into values, as load does for
// its reader function, must return this one as it is.
func IsContFunctionCall(err error) bool {
	_, ok := err.(*contFunctionCall)
	return ok
}
