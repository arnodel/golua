package runtime

import (
	"errors"
	"unsafe"
)

// A Continuation is a function that stands for the rest of a computation:
// calling it makes the call it was captured at return its arguments.  It can
// be called any number of times, before or after that call has returned.
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
type Continuation struct {
	thread *Thread
	loop   uint64 // The thread's loop at the capture, see reinstate
	cont   Cont   // A frozen frame, or a continuation that is not a frame

	goFunction *GoFunction // Calls k, for Go code; made when first needed
}

var _ Callable = (*Continuation)(nil)

// NewCallCC returns a function callcc(f), which calls f(k) and returns what
// f returns, where k is the continuation of the call of callcc.
func NewCallCC() *GoFunction {
	return &GoFunction{f: callcc, name: "callcc", nArgs: 1, keepsNext: true}
}

func callcc(t *Thread, c *GoCont) (Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	// The caller's frame is frozen from here on.  This function says it
	// keeps its next continuation as it finds it (keepsNext), so that a
	// frame that is frozen already is not copied to be frozen again.
	next := c.next
	if frame, ok := next.(*LuaCont); ok {
		next = (*frozenCont)(frame)
	}
	t.RequireSize(unsafe.Sizeof(Continuation{}))
	k := &Continuation{thread: t, loop: t.loop, cont: next}

	// f returns where callcc would: this is a tail call.
	fc, err := Continue(t, c.Arg(0), next)
	if err != nil {
		return nil, err
	}
	fc.Push(t.Runtime, Value{iface: k})
	return fc, nil
}

// reinstate returns a continuation that carries on the computation k was
// captured from, for its caller to push values into and run in place of the
// running one.
//
// A frame returns to the frame that called it, and so on down to a
// continuation that is not a frame: that is where the Go code that started
// them waits for them to finish, in RunContinuation, and that cannot be
// captured.  So a Continuation is of no use once that Go code has returned,
// or from within other Go code that has called back into Lua since, or in
// another thread.  All three are errors, and the first two are told by the
// thread not being in the loop it was in at the capture.
func (k *Continuation) reinstate(t *Thread) (Cont, error) {
	if t != k.thread {
		return nil, errors.New("attempt to call a continuation from another thread")
	}
	if t.loop != k.loop {
		return nil, errors.New("attempt to call a continuation across a Go call boundary")
	}
	if frame, ok := k.cont.(*frozenCont); ok {
		return frame.thaw(t), nil
	}
	return k.cont, nil
}

// Continuation implements Callable.Continuation.  This is how Go code calls
// a function, and it may well run what it gets in a loop of its own, so k is
// reinstated when the call runs and not here: the call is that of a Go
// function which does it.  Lua code calling k does not come this way, see
// continueCall.
func (k *Continuation) Continuation(t *Thread, next Cont) Cont {
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

func (k *Continuation) call(t *Thread, c *GoCont) (Cont, error) {
	cont, err := k.reinstate(t)
	if err != nil {
		return nil, err
	}
	cont.PushEtc(t.Runtime, c.Etc())
	return cont, nil
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
