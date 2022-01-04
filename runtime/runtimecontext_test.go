package runtime

import (
	"reflect"
	"testing"
)

func TestRuntimeResources_Dominates(t *testing.T) {
	tests := []struct {
		name string
		r    RuntimeResources
		v    RuntimeResources
		want bool
	}{
		{
			name: "both 0",
			want: true,
		},
		{
			name: "v > 0",
			v: RuntimeResources{
				Cpu:    1,
				Mem:    2,
				Millis: 3,
			},
			want: true,
		},
		{
			name: "r > 0",
			r: RuntimeResources{
				Cpu:    1,
				Mem:    2,
				Millis: 3,
			},
			want: true,
		},
		{
			name: "r > v > 0",
			r: RuntimeResources{
				Cpu:    10,
				Mem:    20,
				Millis: 30,
			},
			v: RuntimeResources{
				Cpu:    5,
				Mem:    10,
				Millis: 10,
			},
			want: true,
		},
		{
			name: "r.Cpu < v.Cpu",
			r: RuntimeResources{
				Cpu: 10,
			},
			v: RuntimeResources{
				Cpu: 11,
			},
			want: false,
		},
		{
			name: "r.Mem == v.Mem",
			r: RuntimeResources{
				Mem: 10,
			},
			v: RuntimeResources{
				Mem: 10,
			},
			want: false,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.Dominates(tt.v); got != tt.want {
				t.Errorf("RuntimeResources.Dominates() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRuntimeResources_Merge(t *testing.T) {
	tests := []struct {
		name string
		r    RuntimeResources
		r1   RuntimeResources
		want RuntimeResources
	}{
		{
			name: "all 0",
		},
		{
			name: "Mix",
			r: RuntimeResources{
				Cpu:    10,
				Mem:    20,
				Millis: 30,
			},
			r1: RuntimeResources{
				Cpu: 20,
				Mem: 10,
			},
			want: RuntimeResources{
				Cpu:    10,
				Mem:    10,
				Millis: 30,
			},
		},
		{
			name: "r == 0",
			r1: RuntimeResources{
				Cpu:    10,
				Mem:    20,
				Millis: 30,
			},
			want: RuntimeResources{
				Cpu:    10,
				Mem:    20,
				Millis: 30,
			},
		},
		{
			name: "r1 > r",
			r: RuntimeResources{
				Cpu:    10,
				Mem:    20,
				Millis: 30,
			},
			r1: RuntimeResources{
				Cpu:    100,
				Mem:    200,
				Millis: 300,
			},
			want: RuntimeResources{
				Cpu:    10,
				Mem:    20,
				Millis: 30,
			},
		},

		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if got := tt.r.Merge(tt.r1); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RuntimeResources.Merge() = %v, want %v", got, tt.want)
			}
			if got := tt.r1.Merge(tt.r); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("symmetric RuntimeResources.Merge() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRuntimeResources_Remove(t *testing.T) {
	tests := []struct {
		name string
		r    RuntimeResources
		v    RuntimeResources
		want RuntimeResources
	}{
		{
			name: "all 0",
		},
		{
			name: "r == 0",
			v: RuntimeResources{
				Cpu: 10,
				Mem: 100,
			},
		},
		{
			name: "r > 0",
			r: RuntimeResources{
				Cpu:    100,
				Mem:    10,
				Millis: 50,
			},
			v: RuntimeResources{
				Cpu:    50,
				Millis: 100,
			},
			want: RuntimeResources{
				Cpu: 50,
				Mem: 10,
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.Remove(tt.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RuntimeResources.Remove() = %v, want %v", got, tt.want)
			}
		})
	}
}
