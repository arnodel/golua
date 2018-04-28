package runtime

// Cont is an interface for things that can be run.
type Cont interface {
	Push(Value)
	RunInThread(*Thread) (Cont, error)
}

func Push(c Cont, vals ...Value) {
	for _, v := range vals {
		c.Push(v)
	}
}
