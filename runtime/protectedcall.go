package runtime

import (
	"errors"
	"unsafe"
)

// # Protected calls
//
// pcall(f, ...) calls f so that an error in it does not go further than the
// call: pcall returns false and the error.  No Go code waits for f to do
// that.  The continuation that f returns to is a protectedCall, which stays
// in the chain of continuations for as long as f runs, and marks the place
// where an error stops:
//
//   - if f returns, the protectedCall is run like any continuation that
//     receives results, and gives true and the results to the continuation
//     that pcall returns to;
//   - if a continuation fails, the loop that ran it looks down the chain from
//     there (protectedCallFor).  If it finds a protectedCall it carries on
//     from it, with false and the error (caught).
//
// So f runs in the loop that pcall was called in, and can yield like any
// Lua code, whether its thread has a goroutine or not.
//
// A protectedCall also has the message handler of xpcall, which is called
// where the error is, before anything is undone (messageHandlerFor).

// A protectedCall is the continuation that a function called in protected
// mode returns to.
type protectedCall struct {
	next    Cont     // Where the protected call returns
	handler Callable // The message handler, if there is one
	name    string   // The function that made the call, for tracebacks

	// The state to go back to if the function fails.
	closeStackHeight int
	depth            int

	results []Value // What the function returned
}

var _ Cont = (*protectedCall)(nil)

// maxProtectedCallDepth is how deep protected calls can nest in a thread.
// Lua frames are not limited, but a function that keeps calling itself
// through pcall would never see the error that stops it.
const maxProtectedCallDepth = 1000

// ProtectedCall returns the continuation that calls f with args in protected
// mode: next is given true and the results of f, or false and the error if
// there is one.  If handler is not nil it is the message handler, called
// with the error where it happens, whose result replaces the error.  The
// name is that of the function making the call, as tracebacks will show it
// (pcall and xpcall are written with this).
func (t *Thread) ProtectedCall(name string, f Value, args []Value, handler Callable, next Cont) Cont {
	t.RequireSize(unsafe.Sizeof(protectedCall{}))
	p := &protectedCall{
		next:             next,
		handler:          handler,
		name:             name,
		closeStackHeight: t.closeStack.size(),
		depth:            t.protectedCallDepth + 1,
	}
	if p.depth > maxProtectedCallDepth {
		return p.caught(t, errors.New("stack overflow"))
	}
	c, err := Continue(t, f, p)
	if err != nil {
		return p.caught(t, err)
	}
	t.protectedCallDepth = p.depth
	t.Push(c, args...)
	_ = t.triggerCall(t, c)
	return c
}

// Push implements Cont.Push: p keeps the results of the function until it
// runs.  They cannot go to the next continuation as they come, because
// closing a variable of the function can still fail after it has returned
// them.
func (p *protectedCall) Push(r *Runtime, v Value) {
	r.RequireSize(unsafe.Sizeof(Value{}))
	p.results = append(p.results, v)
}

// PushEtc implements Cont.PushEtc.
func (p *protectedCall) PushEtc(r *Runtime, etc []Value) {
	r.RequireArrSize(unsafe.Sizeof(Value{}), len(etc))
	p.results = append(p.results, etc...)
}

// RunInThread implements Cont.RunInThread: the function has returned.
func (p *protectedCall) RunInThread(t *Thread) (Cont, error) {
	t.protectedCallDepth = p.depth - 1
	_ = t.triggerReturn(t, p)
	p.next.Push(t.Runtime, BoolValue(true))
	p.next.PushEtc(t.Runtime, p.results)
	return p.next, nil
}

// caught is called when the function has failed with err.  It returns the
// continuation to carry on with, which has been given false and the error.
func (p *protectedCall) caught(t *Thread, err error) Cont {
	t.protectedCallDepth = p.depth - 1
	// The variables that the function and what it called had left to close
	// are closed, with the error.  If that fails the error is the new one.
	err = t.cleanupCloseStack(p, p.closeStackHeight, err)
	_ = t.triggerReturn(t, p)
	p.next.Push(t.Runtime, BoolValue(false))
	p.next.Push(t.Runtime, ErrorValue(err))
	return p.next
}

// Next implements Cont.Next.
func (p *protectedCall) Next() Cont {
	return p.next
}

// Parent implements Cont.Parent.
func (p *protectedCall) Parent() Cont {
	return p.next
}

// DebugInfo implements Cont.DebugInfo.  A protected call shows in a
// traceback as the Go function that made it.
func (p *protectedCall) DebugInfo() *DebugInfo {
	return &DebugInfo{
		Source: "[Go]",
		Name:   p.name,
	}
}

// protectedCallFor returns the protected call that an error in c stops at,
// if it is one made in the loop that runs c: the first one down the chain of
// continuations from c.  The chain of a loop ends with a Termination, which
// is where Go code waits for the loop to end, and the error is for that Go
// code if no protected call comes first.
func protectedCallFor(c Cont) *protectedCall {
	for c = c.Next(); c != nil; c = c.Next() {
		if p, ok := c.(*protectedCall); ok {
			return p
		}
	}
	return nil
}

// messageHandlerFor returns the message handler for an error in c, running
// in t: that of the innermost protected call that c is part of, or that of
// the runtime context if the context was started inside that call.  It is nil
// if there is none.
//
// It also returns how many calls of a message handler the error is in: a
// handler that fails is called with its own error, and so on, which has to
// stop somewhere.
//
// Unlike protectedCallFor this looks through the Go code that is waiting for
// c, a Go function that has called Lua or a thread that has resumed another:
// the handler is called where the error is, before the error gets to that Go
// code.
func (t *Thread) messageHandlerFor(c Cont) (handler Callable, handlerDepth int) {
	// Runtime contexts are the same for all threads.
	contextHandler, contextStart := t.messageHandler, t.startCont
	c = c.Next()
	for {
		switch x := c.(type) {
		case nil:
			// The end of what the thread is running: carry on with the
			// thread that resumed it, which is waiting in that call.
			t = t.caller
			if t == nil {
				return contextHandler, handlerDepth
			}
			c = t.currentCont
		case *protectedCall:
			return x.handler, handlerDepth
		case *messageHandlerCont:
			// What a call of a message handler returns to.
			handlerDepth++
			c = x.Next()
		case *Termination:
			// Go code is waiting for the loop that ends here: carry on
			// with the continuation that was running when it called Lua.
			c = x.parent
		default:
			if c == contextStart {
				// The context was started after any protected call
				// further down.
				return contextHandler, handlerDepth
			}
			c = c.Next()
		}
	}
}
