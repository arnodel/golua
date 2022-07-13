//go:build noquotas
// +build noquotas

package runtime

import (
	"fmt"

	"github.com/arnodel/golua/runtime/internal/luagc"
)

const QuotasAvailable = false

type runtimeContextManager struct {
	messageHandler Callable
	parent         *runtimeContextManager
	weakRefPool    luagc.Pool
}

var _ RuntimeContext = (*runtimeContextManager)(nil)

func (m *runtimeContextManager) initRoot() {
	m.weakRefPool = luagc.NewDefaultPool()
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
	parent := *m
	m.messageHandler = ctx.MessageHandler
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

func (m *runtimeContextManager) ResetQuota() {
}

func (m *runtimeContextManager) TerminateContext(format string, args ...interface{}) {
	// I don't know if it should do it?
	panic(ContextTerminationError{
		message: fmt.Sprintf(format, args...),
	})
}
