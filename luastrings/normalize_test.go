package luastrings

import (
	"testing"
)

func TestNormalizeNewLines(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want string
	}{
		{
			name: "no line endings",
			arg:  "hello there",
			want: "hello there",
		},
		{
			name: "only LF",
			arg:  "line\nline\nline",
			want: "line\nline\nline",
		},
		{
			name: "CR LF CR",
			arg:  "\r\nline\r\n\rline",
			want: "\nline\n\nline",
		},
		{
			name: "LF CR LF",
			arg:  "line\n\r\nline\n\r",
			want: "line\n\nline\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeNewLines([]byte(tt.arg)); string(got) != tt.want {
				t.Errorf("NormalizeNewLines() = %q, want %q", got, tt.want)
			}
		})
	}
}
