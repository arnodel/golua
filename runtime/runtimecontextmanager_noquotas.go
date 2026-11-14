//go:build noquotas
// +build noquotas

package runtime

import (
	"fmt"

	"github.com/arnodel/golua/safeio"
)

const QuotasAvailable = false

type runtimeContextManager struct {
	messageHandler Callable
	parent         *runtimeContextManager
	weakRefPool    *WeakPool
	workDir        string
}

var _ RuntimeContext = (*runtimeContextManager)(nil)

func (m *runtimeContextManager) initRoot() {
	m.weakRefPool = NewWeakPool()
}

func (m *runtimeContextManager) HardLimits() (r RuntimeResources) {
	return
}

func (m *runtimeContextManager) SoftLimits() (r RuntimeResources) {
	return
}

func (m *runtimeContextManager) UsedResources() (r RuntimeResources) {
	return
}

func (m *runtimeContextManager) setStatus(RuntimeContextStatus) {
}

func (m *runtimeContextManager) Status() RuntimeContextStatus {
	return StatusLive
}

func (m *runtimeContextManager) RequiredFlags() (f ComplianceFlags) {
	return
}

func (m *runtimeContextManager) CheckRequiredFlags(ComplianceFlags) error {
	return nil
}

func (m *runtimeContextManager) Parent() RuntimeContext {
	return nil
}

func (m *runtimeContextManager) Due() bool {
	return false
}

func (m *runtimeContextManager) SetStopLevel(StopLevel) {
}

func (m *runtimeContextManager) GCPolicy() GCPolicy {
	return ShareGCPolicy
}

func (m *runtimeContextManager) RuntimeContext() RuntimeContext {
	return m
}

func (m *runtimeContextManager) PushContext(ctx RuntimeContextDef) {
	// Restrictions are not enforced so the context's FSAccess is not needed.
	_ = ctx.FSAccess.Close()
	parent := *m
	m.messageHandler = ctx.MessageHandler
	// The working directory changes what paths mean, not what is allowed, so
	// it applies even though restrictions are not enforced.
	m.workDir = resolveWorkDir(m.workDir, ctx.WorkDir)
	m.parent = &parent
}

func (m *runtimeContextManager) PopContext() RuntimeContext {
	if m == nil || m.parent == nil {
		return nil
	}
	mCopy := *m
	*m = *m.parent
	return &mCopy
}

func (m *runtimeContextManager) CallContext(def RuntimeContextDef, f func() error) (ctx RuntimeContext, err error) {
	m.PushContext(def)
	defer m.PopContext()
	return nil, f()
}

func (m *runtimeContextManager) RequireCPU(cpuAmount uint64) {
}

func (m *runtimeContextManager) UnusedCPU() uint64 {
	return 0
}

func (m *runtimeContextManager) RequireMem(memAmount uint64) {
}

func (m *runtimeContextManager) RequireSize(sz uintptr) uint64 {
	return 0
}

func (m *runtimeContextManager) RequireArrSize(sz uintptr, n int) uint64 {
	return 0
}

func (m *runtimeContextManager) RequireBytes(n int) uint64 {
	return 0
}

func (m *runtimeContextManager) ReleaseMem(memAmount uint64) {
}

func (m *runtimeContextManager) ReleaseSize(sz uintptr) {
}

func (m *runtimeContextManager) ReleaseArrSize(sz uintptr, n int) {
}

func (m *runtimeContextManager) ReleaseBytes(n int) {
}

func (m *runtimeContextManager) UnusedMem() uint64 {
	return 0
}

func (m *runtimeContextManager) LinearUnused(cpuFactor uint64) uint64 {
	return 0
}

func (m *runtimeContextManager) LinearRequire(cpuFactor uint64, amt uint64) {
}

func (m *runtimeContextManager) WorkDir() string {
	return m.workDir
}

func (m *runtimeContextManager) FSAccess() *safeio.FSAccess {
	return safeio.Unrestricted
}

func (m *runtimeContextManager) TerminateContext(format string, args ...interface{}) {
	// I don't know if it should do it?
	panic(ContextTerminationError{
		message: fmt.Sprintf(format, args...),
	})
}

// Resource accounting does nothing without quotas, and these methods make
// sure it costs nothing either.  They hide the ones Runtime gets from the
// context manager embedded in it: calling one of those through a *Runtime
// or a *Thread first takes the address of the manager inside the Runtime,
// which has to check that the Runtime pointer isn't nil, so every call site
// kept a load and a check for a method with an empty body.  A method on
// *Runtime itself has nothing to check, and its call compiles to nothing.

func (r *Runtime) RequireCPU(cpuAmount uint64) {}

func (r *Runtime) UnusedCPU() uint64 { return 0 }

func (r *Runtime) RequireMem(memAmount uint64) {}

func (r *Runtime) RequireSize(sz uintptr) uint64 { return 0 }

func (r *Runtime) RequireArrSize(sz uintptr, n int) uint64 { return 0 }

func (r *Runtime) RequireBytes(n int) uint64 { return 0 }

func (r *Runtime) ReleaseMem(memAmount uint64) {}

func (r *Runtime) ReleaseSize(sz uintptr) {}

func (r *Runtime) ReleaseArrSize(sz uintptr, n int) {}

func (r *Runtime) ReleaseBytes(n int) {}

func (r *Runtime) UnusedMem() uint64 { return 0 }

func (r *Runtime) LinearUnused(cpuFactor uint64) uint64 { return 0 }

func (r *Runtime) LinearRequire(cpuFactor uint64, amt uint64) {}
