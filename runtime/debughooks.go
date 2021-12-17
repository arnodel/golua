package runtime

/*
Debug hooks.  For call/return, it's a bit complicated.  The logic is split
between

- LuaCont.RunInThread(): when exiting, a call / tailcall / return event is
  emitted
- GoCont.RuntInThread(): when exiting, a return event is emitted
- Thread.RunContinuation: at the start (before the loop), a call event is
  emitted

It's unfortunate it has to be split like this but I cannot find a better
approach.
*/

type DebugHookFlags uint8

const (
	hookFlagInHook DebugHookFlags = 1 << iota
	HookFlagCall
	HookFlagReturn
	HookFlagLine
	HookFlagCount
)

type DebugHooks struct {
	DebugHookFlags DebugHookFlags
	HookLineCount  int
	Hook           Value
}

func (h *DebugHooks) callHook(t *Thread, c Cont, args ...Value) *Error {
	if h.DebugHookFlags&hookFlagInHook != 0 || c == nil {
		return nil
	}
	h.DebugHookFlags |= hookFlagInHook
	defer func() { h.DebugHookFlags &= ^hookFlagInHook }()
	term := NewTerminationWith(c, 0, false)
	return Call(t, h.Hook, args, term)
}

func (h *DebugHooks) SetupHooks(newHooks DebugHooks) {
	if h.DebugHookFlags&hookFlagInHook != 0 {
		return
	}
	*h = newHooks
}

var (
	callHookString     = StringValue("call")
	tailCallHookString = StringValue("tail call")
	returnHookString   = StringValue("return")
)

// Important for this function to inline
func (h *DebugHooks) triggerCall(t *Thread, c Cont) *Error {
	if h.DebugHookFlags&HookFlagCall == 0 {
		return nil
	}
	return h.callHook(t, c, callHookString)
}

// Important for this function to inline
func (h *DebugHooks) triggerTailCall(t *Thread, c Cont) *Error {
	if h.DebugHookFlags&HookFlagCall == 0 {
		return nil
	}
	return h.callHook(t, c, tailCallHookString)
}

// Important for this function to inline
func (h *DebugHooks) triggerReturn(t *Thread, c Cont) *Error {
	if h.DebugHookFlags&HookFlagReturn == 0 {
		return nil
	}
	return h.callHook(t, c, returnHookString)
}

// Important for this function to inline
func (h *DebugHooks) areFlagsEnabled(flags DebugHookFlags) bool {
	return h.DebugHookFlags&hookFlagInHook == 0 && h.DebugHookFlags&flags != 0
}
