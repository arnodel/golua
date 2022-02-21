//go:build safepool

package weakref

// NewPool returns a new WeakRefPool with an appropriate implementation.
func NewDefaultPool() Pool {
	return NewUnsafePool()
}
