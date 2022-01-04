//go:build noquotas
// +build noquotas

package runtime

import (
	"testing"
)

func Test_runtimeContextManager_RuntimeContext(t *testing.T) {
	var m runtimeContextManager
	if m.HardLimits() != (RuntimeResources{}) {
		t.Fail()
	}
	if m.SoftLimits() != (RuntimeResources{}) {
		t.Fail()
	}
	if m.UsedResources() != (RuntimeResources{}) {
		t.Fail()
	}
	if m.Status() != StatusLive {
		t.Fail()
	}
	if m.RequiredFlags() != 0 {
		t.Fail()
	}
	if m.CheckRequiredFlags(0) != nil {
		t.Fail()
	}
	if m.Parent() != nil {
		t.Fail()
	}
	if m.Due() {
		t.Fail()
	}
	if m.RuntimeContext() != &m {
		t.Fail()
	}
	m2 := m
	m2.SetStopLevel(HardStop | SoftStop)
	if m != m2 {
		t.Fail()
	}
}

func Test_runtimeContextManager_TerminateContext(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("No panic")
		}
	}()
	var m runtimeContextManager
	m.TerminateContext("foo")
}

func Test_runtimeContextManager_UnusedResources(t *testing.T) {
	var m runtimeContextManager
	if m.UnusedMem() != 0 {
		t.Fail()
	}
	if m.UnusedCPU() != 0 {
		t.Fail()
	}
}

func Test_runtimeContextManager_ResetQuota(t *testing.T) {
	var m runtimeContextManager
	var m1 = m
	m.ResetQuota()
	if m != m1 {
		t.Fail()
	}
}
