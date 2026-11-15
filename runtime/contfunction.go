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
type ContFunction struct {
	thread *Thread
	loop   *runLoop // The thread's loop at the capture, see call
	cont   Cont     // A frozen frame, or a continuation that is not a frame

	// The values to be closed that were pending at the capture.
	closeStack closeStack

	goFunction *GoFunction // Calls k, for Go code; made when first needed
}

var _ Callable = (*ContFunction)(nil)

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
// k carries on in the RunContinuation loop that is running, so next must be
// part of that loop, as the continuation a Go function is given is: not one
// made to start a loop of its own, such as a Termination.  k also keeps the
// values to be closed that are pending, which calling k makes pending again.
func (t *Thread) CaptureCont(next Cont) (Value, Cont) {
	if frame, ok := next.(*LuaCont); ok {
		next = (*frozenCont)(frame)
	}
	if t.loop == nil {
		t.RequireSize(unsafe.Sizeof(runLoop{}))
		t.loop = new(runLoop)
	}
	t.RequireSize(unsafe.Sizeof(ContFunction{}))
	k := &ContFunction{thread: t, loop: t.loop, cont: next, closeStack: t.closeStack.capture()}
	return Value{iface: k}, next
}

// A runLoop is what a ContFunction keeps of the RunContinuation loop it was
// captured in: whether it has finished.  A thread makes one for a loop when
// a continuation is first captured in it (Thread.loop).
type runLoop struct {
	done bool
}

// resume returns a continuation that carries on the computation k was
// captured from, for its caller to push values into and run in place of the
// running one.  The thread must be in the loop k was captured in.  The
// values to be closed that are pending become those of k, which closes those
// of the blocks that are left for good (see switchCloseStack), and that can
// fail.
func (k *ContFunction) resume(t *Thread) (Cont, error) {
	if err := t.switchCloseStack(k.closeStack); err != nil {
		return nil, err
	}
	if frame, ok := k.cont.(*frozenCont); ok {
		return frame.thaw(t), nil
	}
	return k.cont, nil
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
			keepsNext:   true,
			safetyFlags: ComplyCpuSafe | ComplyMemSafe | ComplyTimeSafe | ComplyIoSafe,
		}
	}
	return NewGoCont(t, k.goFunction, next)
}

// call calls k with the arguments of c.
//
// A frame returns to the frame that called it, and so on down to a
// continuation that is not a frame: that is where the Go code that started
// them waits for them to finish, in a RunContinuation loop.  That Go code
// is not part of what k holds, so k carries on in the loop it was captured
// in and nowhere else.
//
//   - In that loop, the frame k holds runs next.
//   - Under Go code which that loop is waiting for (a pcall, a metamethod
//     call, a function given to table.sort), that Go code has to return
//     first.  The call returns as an error does, a continuationCall, which
//     Go code passes on as it does errors and the loop takes up.
//   - Once that loop has finished there is nowhere to carry on, and in
//     another thread it is not this thread's to run: both are errors.
func (k *ContFunction) call(t *Thread, c *GoCont) (Cont, error) {
	switch {
	case t.loop == k.loop:
		cont, err := k.resume(t)
		if err != nil {
			return nil, err
		}
		cont.PushEtc(t.Runtime, c.Etc())
		return cont, nil
	case t != k.thread:
		return nil, errors.New("attempt to call a continuation from another thread")
	case k.loop.done:
		return nil, errors.New("attempt to call a continuation captured in a Go call that has returned")
	}
	// The arguments are the function's until it returns.
	t.RequireArrSize(unsafe.Sizeof(Value{}), len(c.Etc()))
	return nil, &continuationCall{k: k, args: append([]Value(nil), c.Etc()...)}
}

// A continuationCall is the call of a continuation on its way to the loop
// that can run it, through the Go code that loop is waiting for.  It travels
// as an error so that this Go code returns, and it is not one.
type continuationCall struct {
	k    *ContFunction
	args []Value
}

// Error implements error.  This is what a Go function gets if it makes a
// message of the errors of the Lua code it calls, instead of returning them.
func (*continuationCall) Error() string {
	return "attempt to call a continuation across a Go function that does not let it through"
}

// IsContinuationCall reports whether err is the call of a continuation on
// its way through Go code (see Thread.RunContinuation).  A Go function that
// turns the errors of the Lua code it calls into values, as pcall does, must
// return this one as it is.
func IsContinuationCall(err error) bool {
	_, ok := err.(*continuationCall)
	return ok
}

// A frozenCont is a frame that was captured, or that a captured frame
// returns to.  It is a LuaCont that does not change any more: it has the
// same memory, and another type so that the places which push values into
// a continuation, and tell a LuaCont from the others already, find it among
// the others.  There they make a copy of it with thaw and use that.
//
// Only the frame at the capture is given this type then.  The frames it
// returns through are frozen just the same, because the captured frame was
// the only way to them, and they get the type one at a time: a copy refers
// to the frame it returns to as a frozenCont.
type frozenCont LuaCont

var _ Cont = (*frozenCont)(nil)

// thaw returns a copy of the frame that can run: the same function at the
// same place with the same values in its registers.  It is made as
// NewLuaCont makes a frame, so it is released as any frame is.
func (f *frozenCont) thaw(t *Thread) *LuaCont {
	src := (*LuaCont)(f)
	n := int(src.RegCount)
	t.RequireCPU(uint64(n))
	t.RequireArrSize(unsafe.Sizeof(Value{}), n)
	registers := t.regPool.getIn(src.regClass, src.regSize, n)
	copy(registers, src.registers)
	c := t.luaContPool.get()
	t.RequireSize(unsafe.Sizeof(LuaCont{}))
	c.Closure = src.Closure
	c.registers = registers
	c.pc = src.pc
	c.closeStackBase = src.closeStackBase
	if len(src.acc) > 0 {
		t.RequireArrSize(unsafe.Sizeof(Value{}), len(src.acc))
		c.acc = append([]Value(nil), src.acc...)
	}

	// The frame this one returns to is frozen too.
	if parent, ok := registers[0].iface.(*LuaCont); ok {
		registers[0] = Value{iface: (*frozenCont)(parent)}
	}

	// The other frames a frame can hold are those it has made for calls
	// whose arguments it is still computing ("f(x, g())" makes the frame of
	// f before it calls g).  They have not run but they hold arguments, and
	// they return to this frame or where it returns: they are copied too.
	for i := 1; i < n; i++ {
		switch callee := registers[i].iface.(type) {
		case *LuaCont:
			d := (*frozenCont)(callee).thaw(t)
			d.registers[0] = calleeNext(callee.registers[0].iface, src, c)
			registers[i] = Value{iface: d}
		case *GoCont:
			next, _ := calleeNext(callee.next, src, c).TryCont()
			d := NewGoCont(t, callee.GoFunction, next)
			d.PushEtc(t.Runtime, callee.args[:callee.nArgs])
			d.PushEtc(t.Runtime, callee.etc)
			registers[i] = Value{iface: d}
		}
	}
	return c
}

// calleeNext returns what a copy of a callee of the frame src returns to,
// given what the callee returns to: the copy c of src, or where c returns.
func calleeNext(next interface{}, src, c *LuaCont) Value {
	if next == interface{}(src) {
		return Value{iface: c}
	}
	return c.registers[0]
}

// A frozen frame is not pushed into: what is about to push thaws it first
// (contToPush in the run loop, GoCont.RunInThread for a Go function).

// Push implements Cont.Push.
func (f *frozenCont) Push(*Runtime, Value) {
	panic("push into a frozen frame")
}

// PushEtc implements Cont.PushEtc.
func (f *frozenCont) PushEtc(*Runtime, []Value) {
	panic("push into a frozen frame")
}

// RunInThread implements Cont.RunInThread.
func (f *frozenCont) RunInThread(t *Thread) (Cont, error) {
	return f.thaw(t), nil
}

// Next implements Cont.Next.
func (f *frozenCont) Next() Cont {
	return (*LuaCont)(f).Next()
}

// Parent implements Cont.Parent.
func (f *frozenCont) Parent() Cont {
	return (*LuaCont)(f).Parent()
}

// DebugInfo implements Cont.DebugInfo.
func (f *frozenCont) DebugInfo() *DebugInfo {
	return (*LuaCont)(f).DebugInfo()
}
