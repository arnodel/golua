//go:build !noquotas
// +build !noquotas

package runtime

import (
	"fmt"
)

const QuotasAvailable = true

type quotaManager struct {
	cpuQuota uint64
	cpuUsed  uint64

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

func (m *quotaManager) Parent() RuntimeContext {
	return m.parent
}

func (m *quotaManager) RuntimeContext() RuntimeContext {
	return m
}

func (m *quotaManager) PushContext(ctx RuntimeContext) {
	m.PushQuota(ctx.CpuLimit(), ctx.MemLimit())
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

func (m *quotaManager) PushQuota(cpuQuota, memQuota uint64) {
	parent := *m
	m.cpuQuota, m.cpuUsed = m.UnusedCPU(), 0
	m.memQuota, m.memUsed = m.UnusedMem(), 0
	if cpuQuota > 0 && (m.cpuQuota == 0 || m.cpuQuota > cpuQuota) {
		m.cpuQuota = cpuQuota
	}
	if memQuota > 0 && (m.memQuota == 0 || m.memQuota > memQuota) {
		m.memQuota = memQuota
	}
	m.status = RCS_Live
	m.parent = &parent
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
		m.cpuUsed += cpuAmount
		if m.cpuUsed >= m.cpuQuota {
			m.cpuUsed = m.cpuQuota
			m.status = RCS_Killed
			panicWithQuotaExceded("CPU quota of %d exceeded", m.cpuQuota)
		}
	}
}

func (m *quotaManager) UpdateCPUQuota(newQuota uint64) {
	m.cpuQuota = newQuota
}

func (m *quotaManager) UnusedCPU() uint64 {
	return m.cpuQuota - m.cpuUsed
}

func (m *quotaManager) CPUQuotaStatus() (uint64, uint64) {
	return m.cpuUsed, m.cpuQuota
}

func (m *quotaManager) RequireMem(memAmount uint64) {
	if m.memQuota > 0 {
		m.memUsed += memAmount
		if m.memUsed >= m.memQuota {
			m.memUsed = m.memQuota
			m.status = RCS_Killed
			panicWithQuotaExceded("mem quota of %d exceeded", m.memQuota)
		}
	}
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

func (m *quotaManager) UpdateMemQuota(newQuota uint64) {
	m.memQuota = newQuota
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
