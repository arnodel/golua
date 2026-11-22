//go:build !noquotas
// +build !noquotas

package runtime

import (
	"testing"
)

func Test_runtimeContextManager_LinearUnused(t *testing.T) {
	type fields struct {
		hardLimits    RuntimeResources
		usedResources RuntimeResources
	}
	type args struct {
		cpuFactor uint64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   uint64
	}{
		{
			name: "no limits",
			args: args{cpuFactor: 10},
			want: 0,
		},
		{
			name: "no mem limit",
			fields: fields{
				hardLimits: RuntimeResources{Cpu: 1000},
			},
			args: args{cpuFactor: 5},
			want: 5000,
		},
		{
			name: "no cpu limit",
			fields: fields{
				hardLimits: RuntimeResources{Memory: 1000},
			},
			args: args{cpuFactor: 5},
			want: 1000,
		},
		{
			name: "cpu wins",
			fields: fields{
				hardLimits: RuntimeResources{Memory: 1000, Cpu: 100},
			},
			args: args{cpuFactor: 5},
			want: 500,
		},
		{
			name: "mem wins",
			fields: fields{
				hardLimits: RuntimeResources{Memory: 1000, Cpu: 500},
			},
			args: args{cpuFactor: 10},
			want: 1000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &runtimeContextManager{
				hardLimits:    tt.fields.hardLimits,
				usedResources: tt.fields.usedResources,
			}
			if got := m.LinearUnused(tt.args.cpuFactor); got != tt.want {
				t.Errorf("runtimeContextManager.LinearUnused() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_runtimeContextManager_TerminateContext(t *testing.T) {
	type fields struct {
		status RuntimeContextStatus
	}
	tests := []struct {
		name      string
		fields    fields
		wantPanic bool
	}{
		{
			name:      "live context",
			fields:    fields{status: StatusLive},
			wantPanic: true,
		},
		{
			name:      "done context",
			fields:    fields{status: StatusDone},
			wantPanic: false,
		},
		{
			name:      "error context",
			fields:    fields{status: StatusError},
			wantPanic: false,
		},
		{
			name:      "killed context",
			fields:    fields{status: StatusKilled},
			wantPanic: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &runtimeContextManager{
				status: tt.fields.status,
			}
			defer func() {
				switch recover() {
				case nil:
					if tt.wantPanic {
						t.Error("should panic")
					}
				default:
					if !tt.wantPanic {
						t.Error("should not panic")
					} else if m.status != StatusKilled {
						t.Error("context status should be StatusKilled")
					}
				}
			}()
			m.TerminateContext("error")
		})
	}
}

// terminationOf runs f and returns the context termination that it panics
// with, if any.
func terminationOf(f func()) (termErr *ContextTerminationError) {
	defer func() {
		if r := recover(); r != nil {
			e := r.(ContextTerminationError)
			termErr = &e
		}
	}()
	f()
	return nil
}

// A request for CPU that terminates the context is recorded: the work it is
// for may have been done already.
func Test_runtimeContextManager_RequireCPU_recordsTerminatingRequest(t *testing.T) {
	var m runtimeContextManager
	m.initRoot()
	m.PushContext(RuntimeContextDef{HardLimits: RuntimeResources{Cpu: 1000}})
	m.RequireCPU(10)
	if terminationOf(func() { m.RequireCPU(1000) }) == nil {
		t.Fatal("context should be terminated")
	}
	if used := m.UsedResources().Cpu; used != 1010 {
		t.Errorf("used CPU = %d, want 1010", used)
	}
}

// A request for memory that terminates the context is recorded: the memory may
// be in use already.
func Test_runtimeContextManager_RequireMem_recordsTerminatingRequest(t *testing.T) {
	var m runtimeContextManager
	m.initRoot()
	m.PushContext(RuntimeContextDef{HardLimits: RuntimeResources{Memory: 1000}})
	m.RequireMem(10)
	if terminationOf(func() { m.RequireMem(1000) }) == nil {
		t.Fatal("context should be terminated")
	}
	if used := m.UsedResources().Memory; used != 1010 {
		t.Errorf("used memory = %d, want 1010", used)
	}
}

// LinearRequire records both the CPU and the memory of a request, whichever
// of the two terminates the context.
func Test_runtimeContextManager_LinearRequire_recordsBoth(t *testing.T) {
	tests := []struct {
		name   string
		limits RuntimeResources
	}{
		{name: "memory terminates", limits: RuntimeResources{Cpu: 1000, Memory: 1000}},
		{name: "CPU terminates", limits: RuntimeResources{Cpu: 100, Memory: 10000}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m runtimeContextManager
			m.initRoot()
			m.PushContext(RuntimeContextDef{HardLimits: tt.limits})
			if terminationOf(func() { m.LinearRequire(10, 2000) }) == nil {
				t.Fatal("context should be terminated")
			}
			if used := m.UsedResources().Cpu; used != 200 {
				t.Errorf("used CPU = %d, want 200", used)
			}
			if used := m.UsedResources().Memory; used != 2000 {
				t.Errorf("used memory = %d, want 2000", used)
			}
		})
	}
}

// A context terminated for using up the memory that it got from its parent
// terminates its parent when it is popped, and that one its own parent.
func Test_runtimeContextManager_PopContext_memoryTerminatesParents(t *testing.T) {
	var m runtimeContextManager
	m.initRoot()
	m.PushContext(RuntimeContextDef{HardLimits: RuntimeResources{Memory: 1000}})
	m.RequireMem(10)
	m.PushContext(RuntimeContextDef{})
	m.PushContext(RuntimeContextDef{})
	if terminationOf(func() { m.RequireMem(5000) }) == nil {
		t.Fatal("innermost context should be terminated")
	}
	if terminationOf(func() { m.PopContext() }) == nil {
		t.Fatal("middle context should be terminated")
	}
	if terminationOf(func() { m.PopContext() }) == nil {
		t.Fatal("outer context should be terminated")
	}
	if limit := m.HardLimits().Memory; limit != 1000 {
		t.Errorf("current context has memory limit %d, want 1000 (the outer one)", limit)
	}
	if used := m.UsedResources().Memory; used != 5010 {
		t.Errorf("used memory = %d, want 5010", used)
	}
}

// A context terminated for using up the CPU that it got from its parent
// terminates its parent when it is popped, and the parent is the current
// context by then.
func Test_runtimeContextManager_PopContext_terminatesParent(t *testing.T) {
	var m runtimeContextManager
	m.initRoot()
	m.PushContext(RuntimeContextDef{HardLimits: RuntimeResources{Cpu: 1000}})
	m.RequireCPU(10)
	m.PushContext(RuntimeContextDef{})
	if terminationOf(func() { m.RequireCPU(1000) }) == nil {
		t.Fatal("inner context should be terminated")
	}
	if terminationOf(func() { m.PopContext() }) == nil {
		t.Fatal("outer context should be terminated")
	}
	if m.Status() != StatusKilled {
		t.Errorf("status = %v, want killed", m.Status())
	}
	if limit := m.HardLimits().Cpu; limit != 1000 {
		t.Errorf("current context has CPU limit %d, want 1000 (the outer one)", limit)
	}
	if used := m.UsedResources().Cpu; used != 1010 {
		t.Errorf("used CPU = %d, want 1010", used)
	}
	// Popping the outer context leaves the root one.
	if outer := m.PopContext(); outer == nil || outer.Status() != StatusKilled {
		t.Errorf("popped %v, want the killed outer context", outer)
	}
	if m.PopContext() != nil {
		t.Error("the root context should be left")
	}
}

// A context with a CPU limit of its own charges its parent what it used when
// it is terminated, and no more.
func Test_runtimeContextManager_PopContext_chargesOwnLimit(t *testing.T) {
	var m runtimeContextManager
	m.initRoot()
	m.PushContext(RuntimeContextDef{HardLimits: RuntimeResources{Cpu: 1000}})
	m.PushContext(RuntimeContextDef{HardLimits: RuntimeResources{Cpu: 100}})
	// The request that a pattern match makes when it runs out: the CPU that
	// was left and one more.
	if terminationOf(func() { m.RequireCPU(m.UnusedCPU() + 1) }) == nil {
		t.Fatal("inner context should be terminated")
	}
	if terminationOf(func() { m.PopContext() }) != nil {
		t.Fatal("outer context should not be terminated")
	}
	if used := m.UsedResources().Cpu; used != 101 {
		t.Errorf("used CPU = %d, want 101", used)
	}
}
