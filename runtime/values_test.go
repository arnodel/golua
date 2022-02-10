package runtime

import (
	"testing"

	"github.com/arnodel/golua/luastrings"
)

func TestStringNormPos(t *testing.T) {
	type args struct {
		s string
		p int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "start from left",
			args: args{
				s: "hello",
				p: 1,
			},
			want: 1,
		},
		{
			name: "end from left",
			args: args{
				s: "hello",
				p: 5,
			},
			want: 5,
		},
		{
			name: "start from right",
			args: args{
				s: "hello",
				p: -1,
			},
			want: 5,
		},
		{
			name: "end from right",
			args: args{
				s: "hello",
				p: -5,
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := luastrings.StringNormPos(tt.args.s, tt.args.p); got != tt.want {
				t.Errorf("StringNormPos() = %v, want %v", got, tt.want)
			}
		})
	}
}
