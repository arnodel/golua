//go:build safepool

package luagc

// NewPool returns a new Pool with an appropriate implementation.
func NewDefaultPool() Pool {
	return NewUnsafePool()
}
