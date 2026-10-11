package runtime

import "sync/atomic"

// An Interrupt stops a running Lua context from another goroutine.
//
// Install it with RuntimeContextDef.Interrupt when calling CallContext (or
// PushContext). Once Trigger has been called, the context it was installed in,
// and every context running inside it, terminates at its next CPU check, as if
// a hard limit had been reached: protected calls (pcall) inside the context do
// not stop it, because each enclosing context terminates in turn as control
// returns to it. Contexts outside the one the Interrupt was installed in are
// not affected.
//
// Trigger is safe to call from any goroutine, any number of times; the first
// reason given is kept. An Interrupt cannot be reset: use a new one for the
// next context.
type Interrupt struct {
	reason atomic.Value // string, set once by Trigger
}

// NewInterrupt returns an Interrupt that has not been triggered.
func NewInterrupt() *Interrupt {
	return &Interrupt{}
}

// Trigger requests termination of the contexts the Interrupt is installed in.
// The reason becomes the termination error message.
func (i *Interrupt) Trigger(reason string) {
	i.reason.CompareAndSwap(nil, reason)
}

// Triggered reports whether Trigger has been called, and with which reason.
func (i *Interrupt) Triggered() (reason string, ok bool) {
	reason, ok = i.reason.Load().(string)
	return
}
