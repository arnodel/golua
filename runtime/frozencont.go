package runtime

// A frozenCont is a frame that does not change any more: one that was
// captured (see Thread.CaptureCont), or that a captured frame returns to.
// It is a LuaCont under another type, with the same memory.
//
// The type is what keeps a frozen frame from being pushed into.  The fast
// paths for a *LuaCont do not take it, and its Push panics.  What is about
// to push into a continuation tests for the type, and replaces a frozen
// frame with a copy that can run, made by thaw, which gets the values: see
// contToPush in the run loop, and GoCont.RunInThread for the results of a
// Go function.  The test is on the type of an interface value that is at
// hand, so a frame that is not frozen pays a comparison for this.
//
// Only the captured frame is given the type at the capture.  The frames it
// returns through are frozen as well, because the captured frame was the
// only way to reach them, and they get the type one at a time: a copy made
// by thaw refers to the frame it returns to as a frozenCont.  So a
// frozenCont is found in a ContFunction and where a continuation keeps the
// one it returns to (register 0 of a frame, the next of a GoCont or of a
// protected call), and nowhere else.
type frozenCont LuaCont

var _ Cont = (*frozenCont)(nil)

// frozen returns c as a captured continuation keeps it, c being what a
// captured frame, or the capture itself, returns to.  A frame is frozen.  A
// protected call, which a function returns into as it does into a frame, is
// replaced with a copy that nothing will be pushed into (see
// protectedCall.clone).  Any other continuation is not changed by being
// run, and is kept as it is.
func frozen(t *Thread, c Cont) Cont {
	switch c := c.(type) {
	case *LuaCont:
		return (*frozenCont)(c)
	case *protectedCall:
		return c.clone(t)
	}
	return c
}

// isLuaFrame reports whether c is the frame of a Lua function, frozen or
// not.
func isLuaFrame(c Cont) bool {
	switch c.(type) {
	case *LuaCont, *frozenCont:
		return true
	}
	return false
}

// thaw returns a copy of the frame that can run (see LuaCont.clone), linked
// so that running it changes nothing that is frozen.
func (f *frozenCont) thaw(t *Thread) *LuaCont {
	src := (*LuaCont)(f)
	c := src.clone(t)
	registers := c.registers

	// What this frame returns to is frozen too.
	if next, ok := registers[0].TryCont(); ok {
		registers[0] = ContValue(frozen(t, next))
	}

	// The other frames a frame can hold are those it has made for calls
	// whose arguments it is still computing ("f(x, g())" makes the frame of
	// f before it calls g).  They have not run but they hold arguments, and
	// they return to this frame or where it returns: they are copied too,
	// to return to the copy or where it returns.
	for i := 1; i < len(registers); i++ {
		switch callee := registers[i].iface.(type) {
		case *LuaCont:
			d := (*frozenCont)(callee).thaw(t)
			d.registers[0] = ContValue(calleeNext(callee.Next(), src, c))
			registers[i] = ContValue(d)
		case *GoCont:
			registers[i] = ContValue(callee.clone(t, calleeNext(callee.next, src, c)))
		}
	}
	return c
}

// calleeNext returns what the copy of a callee of the frame src returns to,
// given what the callee returns to: the copy c of src, or where c returns.
func calleeNext(next Cont, src, c *LuaCont) Cont {
	if next == Cont(src) {
		return c
	}
	return c.Next()
}

// A frozen frame is not pushed into: what is about to push thaws it first.

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
