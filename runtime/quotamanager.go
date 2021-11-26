//go:build !noquotas
// +build !noquotas

package runtime

import (
	"fmt"
	"strings"
)

const QuotasAvailable = true

type quotaManager struct {
	hardLimits    RuntimeResources
	softLimits    RuntimeResources
	usedResources RuntimeResources

	safetyFlags ComplianceFlags

	status RuntimeContextStatus

	parent *quotaManager
}

var _ RuntimeContext = (*quotaManager)(nil)

func (m *quotaManager) HardResourceLimits() RuntimeResources {
	return m.hardLimits
}

func (m *quotaManager) SoftResourceLimits() RuntimeResources {
	return m.softLimits
}

func (m *quotaManager) UsedResources() RuntimeResources {
	return m.usedResources
}

func (m *quotaManager) Status() RuntimeContextStatus {
	return m.status
}

func (m *quotaManager) SafetyFlags() ComplianceFlags {
	return m.safetyFlags
}

func (m *quotaManager) CheckSafetyFlags(flags ComplianceFlags) *Error {
	missingFlags := m.safetyFlags &^ flags
	if missingFlags != 0 {
		return NewErrorF("missing flags: %s", strings.Join(missingFlags.Names(), " "))
	}
	return nil
}

func (m *quotaManager) Parent() RuntimeContext {
	return m.parent
}

func (m *quotaManager) ShouldCancel() bool {
	return !m.softLimits.Dominates(m.usedResources)
}

func (m *quotaManager) RuntimeContext() RuntimeContext {
	return m
}

func (m *quotaManager) PushContext(ctx RuntimeContextDef) {
	parent := *m
	m.hardLimits = m.hardLimits.Remove(m.usedResources).Merge(ctx.HardLimits)
	m.softLimits = m.softLimits.Remove(m.usedResources).Merge(ctx.SoftLimits)
	m.safetyFlags |= ctx.SafetyFlags

	if ctx.HardLimits.Cpu > 0 {
		m.safetyFlags |= ComplyCpuSafe
	}
	if ctx.HardLimits.Mem > 0 {
		m.safetyFlags |= ComplyMemSafe
	}

	m.status = RCS_Live
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
			_, ok := r.(ContextTerminationError)
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
	m.parent.RequireCPU(m.usedResources.Cpu)
	m.parent.RequireMem(m.usedResources.Mem)
	*m = *m.parent
}

func (m *quotaManager) RequireCPU(cpuAmount uint64) {
	if m.hardLimits.Cpu > 0 {
		// The path with quota is "outlined" so RequireCPU can be inlined,
		// minimising the overhead when there is no quota.
		m.requireCPU(cpuAmount)
	}
}

//go:noinline
func (m *quotaManager) requireCPU(cpuAmount uint64) {
	cpuUsed := m.usedResources.Cpu + cpuAmount
	if cpuUsed >= m.hardLimits.Cpu {
		m.TerminateContext("CPU limit of %d exceeded", m.hardLimits.Cpu)
	}
	m.usedResources.Cpu = cpuUsed
}

func (m *quotaManager) UnusedCPU() uint64 {
	return m.hardLimits.Cpu - m.usedResources.Cpu
}

func (m *quotaManager) RequireMem(memAmount uint64) {
	if m.hardLimits.Mem > 0 {
		// The path with quota is "outlined" so RequireMem can be inlined,
		// minimising the overhead when there is no quota.
		m.requireMem(memAmount)
	}
}

//go:noinline
func (m *quotaManager) requireMem(memAmount uint64) {
	memUsed := m.usedResources.Mem + memAmount
	if memUsed >= m.hardLimits.Mem {
		m.TerminateContext("mem limit of %d exceeded", m.hardLimits.Mem)
	}
	m.usedResources.Mem = memUsed
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
	if m.hardLimits.Mem > 0 {
		if memAmount <= m.usedResources.Mem {
			m.usedResources.Mem -= memAmount
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
	return m.hardLimits.Mem - m.usedResources.Mem
}

func (m *quotaManager) ResetQuota() {
	m.hardLimits = RuntimeResources{}
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

func (m *quotaManager) TerminateContext(format string, args ...interface{}) {
	if m.status != RCS_Live {
		return
	}
	m.status = RCS_Killed
	panic(ContextTerminationError{
		message: fmt.Sprintf(format, args...),
	})
}
