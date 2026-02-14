package luagc

import (
	"os"
	"sync"
)

var (
	poolFactories = make(map[string]func() Pool)
	registryMu    sync.RWMutex
)

// RegisterPool adds a pool factory to the registry.
// This is typically called from init() functions.
func RegisterPool(name string, factory func() Pool) {
	registryMu.Lock()
	defer registryMu.Unlock()
	poolFactories[name] = factory
}

// DefaultPoolFactory returns a factory function that creates the best
// available pool.
//
// The GOLUA_POOL environment variable can override the default selection
// (useful for testing different pool implementations).
//
// Otherwise, it tries "weak" first (available on Go 1.24+), then falls back
// to "clone".
func DefaultPoolFactory() func() Pool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	// Allow override via environment variable (mainly for testing)
	if name := os.Getenv("GOLUA_POOL"); name != "" {
		if factory, ok := poolFactories[name]; ok {
			return factory
		}
	}
	// Default: try weak first (Go 1.24+), fall back to clone
	if factory, ok := poolFactories["weak"]; ok {
		return factory
	}
	return poolFactories["clone"]
}

// PoolNames returns all registered pool names.
func PoolNames() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(poolFactories))
	for name := range poolFactories {
		names = append(names, name)
	}
	return names
}
