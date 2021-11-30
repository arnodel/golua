//go:build !noquotas
// +build !noquotas

package runtime

import (
	"fmt"
	"strings"
	"time"
)

const QuotasAvailable = true

const cpuThresholdIncrement = 1000

type runtimeContextManager struct {
	hardLimits    RuntimeResources
	softLimits    RuntimeResources
	usedResources RuntimeResources

	requiredFlags ComplianceFlags

	status RuntimeContextStatus

	parent *runtimeContextManager

	messageHandler Callable

	trackCpu         bool
	trackMem         bool
	trackTime        bool
	startTime        uint64
	nextCpuThreshold uint64
}

var _ RuntimeContext = (*runtimeContextManager)(nil)

func (m *runtimeContextManager) HardLimits() RuntimeResources {
	return m.hardLimits
}

func (m *runtimeContextManager) SoftLimits() RuntimeResources {
	return m.softLimits
}

func (m *runtimeContextManager) UsedResources() RuntimeResources {
	return m.usedResources
}

func (m *runtimeContextManager) Status() RuntimeContextStatus {
	return m.status
}

func (m *runtimeContextManager) RequiredFlags() ComplianceFlags {
	return m.requiredFlags
}

func (m *runtimeContextManager) CheckRequiredFlags(flags ComplianceFlags) *Error {
	missingFlags := m.requiredFlags &^ flags
	if missingFlags != 0 {
		return NewErrorF("missing flags: %s", strings.Join(missingFlags.Names(), " "))
	}
	return nil
}

func (m *runtimeContextManager) Parent() RuntimeContext {
	return m.parent
}

func (m *runtimeContextManager) ShouldCancel() bool {
	return !m.softLimits.Dominates(m.usedResources)
}

func (m *runtimeContextManager) RuntimeContext() RuntimeContext {
	return m
}

func (m *runtimeContextManager) PushContext(ctx RuntimeContextDef) {
	parent := *m
	m.startTime = now()
	m.hardLimits = m.hardLimits.Remove(m.usedResources).Merge(ctx.HardLimits)
	m.softLimits = m.softLimits.Remove(m.usedResources).Merge(ctx.SoftLimits)
	m.requiredFlags |= ctx.RequiredFlags

	if ctx.HardLimits.Cpu > 0 {
		m.requiredFlags |= ComplyCpuSafe
	}
	if ctx.HardLimits.Mem > 0 {
		m.requiredFlags |= ComplyMemSafe
	}

	m.trackTime = m.hardLimits.Time > 0 || m.softLimits.Time > 0
	m.trackCpu = m.hardLimits.Cpu > 0 || m.softLimits.Cpu > 0 || m.trackTime
	m.trackMem = m.hardLimits.Mem > 0 || m.softLimits.Mem > 0
	m.status = StatusLive
	m.messageHandler = ctx.MessageHandler
	m.parent = &parent
}

func (m *runtimeContextManager) PopContext() RuntimeContext {
	if m == nil || m.parent == nil {
		return nil
	}
	mCopy := *m
	if mCopy.status == StatusLive {
		mCopy.status = StatusDone
	}
	m.parent.RequireCPU(m.usedResources.Cpu)
	m.parent.RequireMem(m.usedResources.Mem)
	*m = *m.parent
	return &mCopy
}

func (m *runtimeContextManager) CallContext(def RuntimeContextDef, f func() *Error) (ctx RuntimeContext, err *Error) {
	m.PushContext(def)
	defer func() {
		ctx = m.PopContext()
		if r := recover(); r != nil {
			_, ok := r.(ContextTerminationError)
			if !ok {
				panic(r)
			}
		}
	}()
	err = f()
	if err != nil {
		m.status = StatusError
	}
	return
}

func (m *runtimeContextManager) RequireCPU(cpuAmount uint64) {
	if m.trackCpu {
		// The path with limit is "outlined" so RequireCPU can be inlined,
		// minimising the overhead when there is no limit.
		m.requireCPU(cpuAmount)
	}
}

//go:noinline
func (m *runtimeContextManager) requireCPU(cpuAmount uint64) {
	cpuUsed := m.usedResources.Cpu + cpuAmount
	if m.hardLimits.Cpu > 0 && cpuUsed >= m.hardLimits.Cpu {
		m.TerminateContext("CPU limit of %d exceeded", m.hardLimits.Cpu)
	}
	if m.trackTime && m.nextCpuThreshold <= cpuUsed {
		m.nextCpuThreshold = cpuUsed + cpuThresholdIncrement
		m.updateTimeUsed()
	}
	m.usedResources.Cpu = cpuUsed
}

func (m *runtimeContextManager) UnusedCPU() uint64 {
	return m.hardLimits.Cpu - m.usedResources.Cpu
}

func (m *runtimeContextManager) RequireMem(memAmount uint64) {
	if m.trackMem {
		// The path with limit is "outlined" so RequireMem can be inlined,
		// minimising the overhead when there is no limit.
		m.requireMem(memAmount)
	}
}

//go:noinline
func (m *runtimeContextManager) requireMem(memAmount uint64) {
	memUsed := m.usedResources.Mem + memAmount
	if memUsed >= m.hardLimits.Mem {
		m.TerminateContext("mem limit of %d exceeded", m.hardLimits.Mem)
	}
	m.usedResources.Mem = memUsed
}

func (m *runtimeContextManager) RequireSize(sz uintptr) (mem uint64) {
	mem = uint64(sz)
	m.RequireMem(mem)
	return
}

func (m *runtimeContextManager) RequireArrSize(sz uintptr, n int) (mem uint64) {
	mem = uint64(sz) * uint64(n)
	m.RequireMem(mem)
	return
}

func (m *runtimeContextManager) RequireBytes(n int) (mem uint64) {
	mem = uint64(n)
	m.RequireMem(mem)
	return
}

func (m *runtimeContextManager) ReleaseMem(memAmount uint64) {
	// TODO: think about what to do when memory is released when unwinding from
	// a quota exceeded error
	if m.hardLimits.Mem > 0 {
		if memAmount <= m.usedResources.Mem {
			m.usedResources.Mem -= memAmount
		} else {
			panic("Too much mem released")
		}
	}
}

func (m *runtimeContextManager) ReleaseSize(sz uintptr) {
	m.ReleaseMem(uint64(sz))
}

func (m *runtimeContextManager) ReleaseArrSize(sz uintptr, n int) {
	m.ReleaseMem(uint64(sz) * uint64(n))
}

func (m *runtimeContextManager) ReleaseBytes(n int) {
	m.ReleaseMem(uint64(n))
}

func (m *runtimeContextManager) UnusedMem() uint64 {
	return m.hardLimits.Mem - m.usedResources.Mem
}

func (m *runtimeContextManager) updateTimeUsed() {
	m.usedResources.Time = now() - m.startTime
	if m.usedResources.Time >= m.hardLimits.Time {
		m.TerminateContext("time limit of %d exceeded", m.hardLimits.Time)
	}
}

func (m *runtimeContextManager) ResetQuota() {
	m.hardLimits = RuntimeResources{}
}

// LinearUnused returns an amount of resource combining memory and cpu.  It is
// useful when calling functions whose time complexity is a linear function of
// the size of their output.  As cpu ticks are "smaller" than memory ticks, the
// cpuFactor arguments allows specifying an increased "weight" for cpu ticks.
func (m *runtimeContextManager) LinearUnused(cpuFactor uint64) uint64 {
	mem := m.UnusedMem()
	cpu := m.UnusedCPU() * cpuFactor
	switch {
	case cpu == 0:
		return mem
	case mem == 0:
		return cpu
	case cpu > mem:
		return mem
	default:
		return cpu
	}
}

// LinearRequire can be used to actually consume (part of) the resource budget
// returned by LinearUnused (with the same cpuFactor).
func (m *runtimeContextManager) LinearRequire(cpuFactor uint64, amt uint64) {
	m.RequireMem(amt)
	m.RequireCPU(amt / cpuFactor)
}

func (m *runtimeContextManager) TerminateContext(format string, args ...interface{}) {
	if m.status != StatusLive {
		return
	}
	m.status = StatusKilled
	panic(ContextTerminationError{
		message: fmt.Sprintf(format, args...),
	})
}

func now() uint64 {
	return uint64(time.Now().UnixNano())
}
