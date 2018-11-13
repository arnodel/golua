package runtime

// Cont is an interface for things that can be run in a Thread.  Implementations
// of Cont a typically an invocation of a Lua function or an invocation of a Go
// function.
//
// TODO: document the methods.
type Cont interface {
	Push(Value)
	RunInThread(*Thread) (Cont, *Error)
	Next() Cont
	DebugInfo() *DebugInfo
}

// Push is a convenience method that pushes a number of values to the
// continuation c.
func Push(c Cont, vals ...Value) {
	for _, v := range vals {
		c.Push(v)
	}
}

func appendTraceback(tb []*DebugInfo, c Cont) []*DebugInfo {
	for c != nil {
		info := c.DebugInfo()
		if info != nil {
			tb = append(tb, info)
		}
		c = c.Next()
	}
	return tb
}
