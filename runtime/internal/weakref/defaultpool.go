//go:build !safepool

package weakref

// NewDefaultPool returns a new WeakRefPool with an appropriate implementation.
func NewDefaultPool() Pool {
	return NewClonePool()
}
