package runtime

import (
	"os"
	"reflect"
	"testing"
)

func Test_runChunk(t *testing.T) {

	if !QuotasAvailable {
		t.Skip("Skipping as build does not enforce quotas")
		return
	}

	type args struct {
		source string
		rtCtx  RuntimeContextDef
	}
	tests := []struct {
		name           string
		args           args
		want           Value
		wantErr        bool
		skipIfNoQuotas bool
	}{
		{
			name: "run out of cpu",
			args: args{
				source: `while true do end`,
				rtCtx:  RuntimeContextDef{HardLimits: RuntimeResources{Cpu: 10000}},
			},
			want:    NilValue,
			wantErr: true,
		},
		{
			name: "return value",
			args: args{
				source: `return 42`,
				rtCtx:  RuntimeContextDef{HardLimits: RuntimeResources{Cpu: 10000}},
			},
			want:    IntValue(42),
			wantErr: false,
		},
		{
			name: "error in execution",
			args: args{
				source: `return {} + 1`,
				rtCtx:  RuntimeContextDef{HardLimits: RuntimeResources{Cpu: 10000}},
			},
			want:    NilValue,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RunChunk1([]byte(tt.args.source), tt.args.rtCtx, os.Stdout)
			if (err != nil) != tt.wantErr {
				t.Errorf("runChunk() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("runChunk() = %v, want %v", got, tt.want)
			}
		})
	}
}
