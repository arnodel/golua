package runtime

// GoCont implements Cont for functions written in Go.
type GoCont struct {
	f    func(*Thread, []Value, Cont) (Cont, error)
	next Cont
	args []Value
}

// Push implements Cont.Push.
func (c *GoCont) Push(v Value) {
	if c.next == nil {
		var ok bool
		c.next, ok = v.(Cont)
		if !ok {
			panic("First push must be a continuation")
		}
	} else {
		c.args = append(c.args, v)
	}
}

// RunInThread implements Cont.RunInThread
func (c *GoCont) RunInThread(t *Thread) (Cont, error) {
	return c.f(t, c.args, c.next)
}
