package runtime

import "unsafe"

// Termination is a 'dead-end' continuation: it cannot be run.
type Termination struct {
	args      []Value
	pushIndex int
	etc       *[]Value
}

var _ Cont = (*Termination)(nil)

// NewTermination returns a new pointer to Termination where the first len(args)
// values will be pushed into args and the remaining ones will be added to etc
// if it is non nil, dropped otherwise.
func NewTermination(args []Value, etc *[]Value) *Termination {
	return &Termination{args: args, etc: etc}
}

// NewTerminationWith creates a new Termination expecting nArgs args and
// possibly gathering extra args into an etc if hasEtc is true.
func NewTerminationWith(nArgs int, hasEtc bool) *Termination {
	var args []Value
	var etc *[]Value
	if nArgs > 0 {
		args = make([]Value, nArgs)
	}
	if hasEtc {
		etc = new([]Value)
	}
	return NewTermination(args, etc)
}

// Push implements Cont.Push.  It just accumulates values into
// a slice.
func (c *Termination) Push(r *Runtime, v Value) {
	if c.pushIndex < len(c.args) {
		c.args[c.pushIndex] = v
		c.pushIndex++
	} else if c.etc != nil {
		r.RequireMem(uint64(unsafe.Sizeof(Value{})))
		*c.etc = append(*c.etc, v)
	}
}

// PushEtc implements Cont.PushEtc.
func (c *Termination) PushEtc(r *Runtime, etc []Value) {
	if c.pushIndex < len(c.args) {
		for i, v := range etc {
			c.args[c.pushIndex] = v
			c.pushIndex++
			if c.pushIndex == len(c.args) {
				etc = etc[i+1:]
				goto FillEtc
			}
		}
		return
	}
FillEtc:
	if c.etc == nil {
		return
	}
	r.RequireMem(uint64(len(etc)) * uint64(unsafe.Sizeof(Value{})))
	*c.etc = append(*c.etc, etc...)
}

// RunInThread implements Cont.RunInThread. A termination exits
// immediately so it always returns nil.
func (c *Termination) RunInThread(t *Thread) (Cont, *Error) {
	return nil, nil
}

// Next implmements Cont.Next.
func (c *Termination) Next() Cont {
	return nil
}

// DebugInfo implements Cont.DebugInfo.
func (c *Termination) DebugInfo() *DebugInfo {
	return nil
}

// Get returns the n-th arg pushed to the termination.
func (c *Termination) Get(n int) Value {
	if n >= c.pushIndex {
		return NilValue
	}
	return c.args[n]
}

// Etc returns all the extra args pushed to the termination.
func (c *Termination) Etc() []Value {
	if c.etc == nil {
		return nil
	}
	return *c.etc
}

// Reset erases all the args pushed to the termination.
func (c *Termination) Reset() {
	c.pushIndex = 0
	if c.etc != nil {
		c.etc = new([]Value)
	}
}
