//go:build !noquotas
// +build !noquotas

package runtime

// A contextHandle is a RuntimeContext that keeps referring to the same context
// while other contexts are pushed and popped.  This is needed because the
// runtimeContextManager's value is swapped when contexts are pushed and popped
// (so a pointer to it always refers to the current context), and ancestors of
// the current context are copies of what it was when they were last current.
type contextHandle struct {
	live  *runtimeContextManager // the live manager
	final *runtimeContextManager // the context's final state once it has finished
}

var _ RuntimeContext = (*contextHandle)(nil)

// finishedContext stands in for a context that cannot be found any more, which
// should not happen.
var finishedContext = &runtimeContextManager{status: StatusDone}

// get returns the current state of the context: the live manager if the
// context is current, its copy in the parent chain if it is an ancestor of the
// current context, or its final state if it has finished.
func (h *contextHandle) get() *runtimeContextManager {
	if h.final != nil {
		return h.final
	}
	for m := h.live; m != nil; m = m.parent {
		if m.handle == h {
			return m
		}
	}
	return finishedContext
}

func (h *contextHandle) HardLimits() RuntimeResources     { return h.get().HardLimits() }
func (h *contextHandle) SoftLimits() RuntimeResources     { return h.get().SoftLimits() }
func (h *contextHandle) UsedResources() RuntimeResources  { return h.get().UsedResources() }
func (h *contextHandle) Status() RuntimeContextStatus     { return h.get().Status() }
func (h *contextHandle) Parent() RuntimeContext           { return h.get().Parent() }
func (h *contextHandle) RequiredFlags() ComplianceFlags   { return h.get().RequiredFlags() }
func (h *contextHandle) SetStopLevel(stopLevel StopLevel) { h.get().SetStopLevel(stopLevel) }
func (h *contextHandle) Due() bool                        { return h.get().Due() }
func (h *contextHandle) GCPolicy() GCPolicy               { return h.get().GCPolicy() }
