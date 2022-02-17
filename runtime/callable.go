package runtime

// Callable is the interface for callable values.
type Callable interface {
	Continuation(*Thread, Cont) Cont
}
