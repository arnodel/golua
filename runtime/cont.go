package runtime

// Cont is an interface for things that can be run.
type Cont interface {
	Push(Value)
	RunInThread(*Thread) (Cont, *Error)
	Next() Cont
	DebugInfo() *DebugInfo
}

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
