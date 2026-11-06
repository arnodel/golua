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

// Memory released by releaseMemIn goes back to the context it was charged
// to, or to wherever that context's usage went when it finished, and to no
// other context.
func Test_runtimeContextManager_releaseMemIn(t *testing.T) {
	limited := RuntimeContextDef{HardLimits: RuntimeResources{Memory: 1 << 20}}
	used := func(m *runtimeContextManager) uint64 { return m.usedResources.Memory }
	newManager := func() *runtimeContextManager {
		m := &runtimeContextManager{}
		m.initRoot()
		m.PushContext(limited) // outer
		m.RequireMem(1000)
		return m
	}

	t.Run("charged and released in the same context", func(t *testing.T) {
		m := newManager()
		id := m.contextID()
		m.RequireMem(2048)
		m.releaseMemIn(id, 2048)
		if used(m) != 1000 {
			t.Errorf("used %d, want 1000", used(m))
		}
	})

	t.Run("charged outside, released inside", func(t *testing.T) {
		m := newManager()
		id := m.contextID()
		m.RequireMem(2048)
		m.PushContext(limited) // inner
		m.RequireMem(100)
		m.releaseMemIn(id, 2048)
		if used(m) != 100 || used(m.parent) != 1000 {
			t.Errorf("inner used %d, outer used %d, want 100 and 1000", used(m), used(m.parent))
		}
		m.PopContext()
		if used(m) != 1100 {
			t.Errorf("outer used %d after inner ended, want 1100", used(m))
		}
	})

	t.Run("charged in a context that has finished", func(t *testing.T) {
		m := newManager()
		m.PushContext(limited) // finishes before the release
		id := m.contextID()
		m.RequireMem(2048)
		m.PopContext()         // outer now holds the 2048
		m.PushContext(limited) // a sibling, which didn't pay for it
		m.RequireMem(100)
		m.releaseMemIn(id, 2048)
		if used(m) != 100 || used(m.parent) != 1000 {
			t.Errorf("sibling used %d, outer used %d, want 100 and 1000", used(m), used(m.parent))
		}
	})

	t.Run("charged two levels out", func(t *testing.T) {
		m := newManager()
		id := m.contextID()
		m.RequireMem(2048)
		m.PushContext(limited)
		m.RequireMem(10)
		m.PushContext(limited)
		m.RequireMem(100)
		m.releaseMemIn(id, 2048)
		if used(m) != 100 || used(m.parent) != 10 || used(m.parent.parent) != 1000 {
			t.Errorf("used %d, %d, %d from the inside out, want 100, 10 and 1000",
				used(m), used(m.parent), used(m.parent.parent))
		}
	})
}
