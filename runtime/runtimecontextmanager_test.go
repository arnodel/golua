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
