//go:build !safepool

package luagc

// NewDefaultPool returns a new WeakRefPool with an appropriate implementation.
func NewDefaultPool() Pool {
	return NewClonePool()
}
