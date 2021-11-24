//go:build !noquotas
// +build !noquotas

package runtime

import (
	"fmt"
	"strings"
)

const QuotasAvailable = true

type quotaManager struct {
	cpuQuota    uint64
	cpuUsed     uint64
	safetyFlags RuntimeSafetyFlags

	memQuota uint64
	memUsed  uint64

	status RuntimeContextStatus

	parent *quotaManager
}

var _ RuntimeContext = (*quotaManager)(nil)

func (m *quotaManager) CpuLimit() uint64 {
	return m.cpuQuota
}

func (m *quotaManager) CpuUsed() uint64 {
	return m.cpuUsed
}

func (m *quotaManager) MemLimit() uint64 {
	return m.memQuota
}

func (m *quotaManager) MemUsed() uint64 {
	return m.memUsed
}

func (m *quotaManager) Status() RuntimeContextStatus {
	return m.status
}

func (m *quotaManager) SafetyFlags() RuntimeSafetyFlags {
	return m.safetyFlags
}

func (m *quotaManager) CheckSafetyFlags(flags RuntimeSafetyFlags) *Error {
	missingFlags := m.safetyFlags &^ flags
	if missingFlags != 0 {
		return NewErrorF("missing flags: %s", strings.Join(missingFlags.Names(), " "))
	}
	return nil
}

func (m *quotaManager) Parent() RuntimeContext {
	return m.parent
}

func (m *quotaManager) RuntimeContext() RuntimeContext {
	return m
}

func (m *quotaManager) PushContext(ctx RuntimeContextDef) {
	parent := *m
	m.cpuQuota, m.cpuUsed = m.UnusedCPU(), 0
	m.memQuota, m.memUsed = m.UnusedMem(), 0
	if ctx.CpuLimit > 0 && (m.cpuQuota == 0 || m.cpuQuota > ctx.CpuLimit) {
		m.cpuQuota = ctx.CpuLimit
		m.safetyFlags |= RCS_CpuSafe
	}
	if ctx.MemLimit > 0 && (m.memQuota == 0 || m.memQuota > ctx.MemLimit) {
		m.memQuota = ctx.MemLimit
		m.safetyFlags |= RCS_MemSafe
	}
	m.status = RCS_Live
	m.safetyFlags |= ctx.SafetyFlags
	m.parent = &parent
}

func (m *quotaManager) PopContext() RuntimeContext {
	if m == nil {
		return nil
	}
	mCopy := *m
	if mCopy.status == RCS_Live {
		mCopy.status = RCS_Done
	}
	m.PopQuota()
	return &mCopy
}

func (m *quotaManager) CallContext(def RuntimeContextDef, f func()) (ctx RuntimeContext) {
	m.PushContext(def)
	defer func() {
		ctx = m.PopContext()
		if r := recover(); r != nil {
			_, ok := r.(QuotaExceededError)
			if !ok {
				panic(r)
			}
		}
	}()
	f()
	return
}

func (m *quotaManager) PopQuota() {
	if m.parent == nil {
		return
	}
	m.parent.RequireCPU(m.cpuUsed)
	m.parent.RequireMem(m.memUsed)
	*m = *m.parent
}

func (m *quotaManager) RequireCPU(cpuAmount uint64) {
	if m.cpuQuota > 0 {
		// The path with quota is "outlined" so RequireCPU can be inlined,
		// minimising the overhead when there is no quota.
		m.requireCPU(cpuAmount)
	}
}

//go:noinline
func (m *quotaManager) requireCPU(cpuAmount uint64) {
	cpuUsed := m.cpuUsed + cpuAmount
	if cpuUsed >= m.cpuQuota {
		m.status = RCS_Killed
		panicWithQuotaExceded("CPU limit of %d exceeded", m.cpuQuota)
	}
	m.cpuUsed = cpuUsed
}

func (m *quotaManager) UnusedCPU() uint64 {
	return m.cpuQuota - m.cpuUsed
}

func (m *quotaManager) CPUQuotaStatus() (uint64, uint64) {
	return m.cpuUsed, m.cpuQuota
}

func (m *quotaManager) RequireMem(memAmount uint64) {
	if m.memQuota > 0 {
		// The path with quota is "outlined" so RequireMem can be inlined,
		// minimising the overhead when there is no quota.
		m.requireMem(memAmount)
	}
}

//go:noinline
func (m *quotaManager) requireMem(memAmount uint64) {
	memUsed := m.memUsed + memAmount
	if memUsed >= m.memQuota {
		m.status = RCS_Killed
		panicWithQuotaExceded("mem limit of %d exceeded", m.memQuota)
	}
	m.memUsed = memUsed
}

func (m *quotaManager) RequireSize(sz uintptr) (mem uint64) {
	mem = uint64(sz)
	m.RequireMem(mem)
	return
}

func (m *quotaManager) RequireArrSize(sz uintptr, n int) (mem uint64) {
	mem = uint64(sz) * uint64(n)
	m.RequireMem(mem)
	return
}

func (m *quotaManager) RequireBytes(n int) (mem uint64) {
	mem = uint64(n)
	m.RequireMem(mem)
	return
}

func (m *quotaManager) ReleaseMem(memAmount uint64) {
	// TODO: think about what to do when memory is released when unwinding from
	// a quota exceeded error
	if m.memQuota > 0 {
		if memAmount <= m.memUsed {
			m.memUsed -= memAmount
		} else {
			panic("Too much mem released")
		}
	}
}

func (m *quotaManager) ReleaseSize(sz uintptr) {
	m.ReleaseMem(uint64(sz))
}

func (m *quotaManager) ReleaseArrSize(sz uintptr, n int) {
	m.ReleaseMem(uint64(sz) * uint64(n))
}

func (m *quotaManager) ReleaseBytes(n int) {
	m.ReleaseMem(uint64(n))
}

func (m *quotaManager) UnusedMem() uint64 {
	return m.memQuota - m.memUsed
}

func (m *quotaManager) MemQuotaStatus() (uint64, uint64) {
	return m.memUsed, m.memQuota
}

func (m *quotaManager) ResetQuota() {
	m.memUsed = 0
	m.cpuUsed = 0
}

// LinearUnused returns an amount of resource combining memory and cpu.  It is
// useful when calling functions whose time complexity is a linear function of
// the size of their output.  As cpu ticks are "smaller" than memory ticks, the
// cpuFactor arguments allows specifying an increased "weight" for cpu ticks.
func (m *quotaManager) LinearUnused(cpuFactor uint64) uint64 {
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
func (m *quotaManager) LinearRequire(cpuFactor uint64, amt uint64) {
	m.RequireMem(amt)
	m.RequireCPU(amt / cpuFactor)
}

func panicWithQuotaExceded(format string, args ...interface{}) {
	panic(QuotaExceededError{
		message: fmt.Sprintf(format, args...),
	})
}
