package runtime

import (
	"sync"
	"sync/atomic"
)

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
// A context can have several interrupts at once, one installed by the code that
// started it and others installed by code running inside it. Triggering any of
// them stops the context it was installed in.
//
// The same Interrupt can be installed in contexts of several runtimes.
//
// Trigger is safe to call from any goroutine, any number of times; the first
// reason given is kept. An Interrupt cannot be reset: use a new one for the
// next context.
//
// A running script is stopped at its next interpreter step, so only code that
// runs Lua is interrupted. A Go function that blocks (for example reading from
// a pipe) is not interrupted: the context stops once that call returns. A host
// that needs a hard deadline should also set a time limit and require the
// timesafe flag, so that such functions cannot be called.
//
// Interrupts need the quotas build. With the noquotas build tag they are
// accepted and ignored, as the other restrictions are.
type Interrupt struct {
	reason atomic.Pointer[string]

	// mu guards installed and orders Trigger with install, so that an
	// interrupt triggered while it is being installed is not missed.
	mu sync.Mutex
	// installed counts, for each runtime flag, the contexts the interrupt is
	// installed in. Trigger raises each of these flags.
	installed map[*atomic.Bool]int
}

// defaultInterruptReason is the termination message when Trigger is given
// none.
const defaultInterruptReason = "interrupted"

// NewInterrupt returns an Interrupt that has not been triggered.
func NewInterrupt() *Interrupt {
	return &Interrupt{}
}

// Trigger requests termination of the contexts the Interrupt is installed in.
// The reason becomes the termination error message ("interrupted" if it is
// empty).
func (i *Interrupt) Trigger(reason string) {
	if reason == "" {
		reason = defaultInterruptReason
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if !i.reason.CompareAndSwap(nil, &reason) {
		return
	}
	for flag := range i.installed {
		flag.Store(true)
	}
}

// Triggered reports whether Trigger has been called, and with which reason.
func (i *Interrupt) Triggered() (reason string, ok bool) {
	if r := i.reason.Load(); r != nil {
		return *r, true
	}
	return "", false
}

// install records that a context of the runtime whose pending flag is given
// runs under the interrupt. If the interrupt has already been triggered, the
// flag is raised at once.
func (i *Interrupt) install(pending *atomic.Bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.installed == nil {
		i.installed = make(map[*atomic.Bool]int)
	}
	i.installed[pending]++
	if i.reason.Load() != nil {
		pending.Store(true)
	}
}

// uninstall undoes one install for the same flag.
func (i *Interrupt) uninstall(pending *atomic.Bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.installed[pending] <= 1 {
		delete(i.installed, pending)
	} else {
		i.installed[pending]--
	}
}

// interruptChain is the list of the interrupts a context runs under, the most
// recently installed first. It is immutable: pushing a context makes a new
// head and leaves the parent's list as it was.
type interruptChain struct {
	interrupt *Interrupt
	next      *interruptChain
}

// triggered returns the reason of the first triggered interrupt of the chain.
func (c *interruptChain) triggered() (reason string, ok bool) {
	for ; c != nil; c = c.next {
		if reason, ok = c.interrupt.Triggered(); ok {
			return
		}
	}
	return "", false
}
