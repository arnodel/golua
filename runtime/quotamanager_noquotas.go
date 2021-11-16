//go:build noquotas
// +build noquotas

package runtime

const QuotasAvailable = false

type quotaManager struct{}

func (m *quotaManager) AllowQuotaModificationsInLua() {
}

func (m *quotaManager) QuotaModificationsInLuaAllowed() bool {
	return false
}

func (m *quotaManager) RequireCPU(cpuAmount uint64) {
}

func (m *quotaManager) UpdateCPUQuota(newQuota uint64) {
}

func (m *quotaManager) UnusedCPU() uint64 {
	return 0
}

func (m *quotaManager) CPUQuotaStatus() (uint64, uint64) {
	return 0, 0
}

func (m *quotaManager) RequireMem(memAmount uint64) {
}

func (m *quotaManager) RequireSize(sz uintptr) {
}

func (m *quotaManager) RequireArrSize(sz uintptr, n int) {
}

func (m *quotaManager) RequireBytes(n int) {
}

func (m *quotaManager) ReleaseMem(memAmount uint64) {
}

func (m *quotaManager) UpdateMemQuota(newQuota uint64) {
}

func (m *quotaManager) UnusedMem() uint64 {
	return 0
}

func (m *quotaManager) MemQuotaStatus() (uint64, uint64) {
	return 0, 0
}

func (m *quotaManager) ResetQuota() {
}
