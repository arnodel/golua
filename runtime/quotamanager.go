//go:build !noquotas
// +build !noquotas

package runtime

import "fmt"

const QuotasAvailable = true

type quotaManager struct {
	cpuQuota uint64
	cpuUsed  uint64

	memQuota uint64
	memUsed  uint64

	quotaModificationsInLuaAllowed bool
}

func (m *quotaManager) AllowQuotaModificationsInLua() {
	m.quotaModificationsInLuaAllowed = true
}

func (m *quotaManager) QuotaModificationsInLuaAllowed() bool {
	return m.quotaModificationsInLuaAllowed
}

func (m *quotaManager) RequireCPU(cpuAmount uint64) {
	if m.cpuQuota > 0 {
		m.cpuUsed += cpuAmount
		if m.cpuUsed >= m.cpuQuota {
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

func panicWithQuotaExceded(format string, args ...interface{}) {
	panic(QuotaExceededError{
		message: fmt.Sprintf(format, args...),
	})
}
