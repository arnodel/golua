package luastrings

import "testing"

func TestQuote(t *testing.T) {
	type args struct {
		s     string
		quote byte
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "single quoted",
			args: args{
				s:     `"hi" 'there'`,
				quote: '\'',
			},
			want: `'"hi" \'there\''`,
		},
		{
			name: "double quoted",
			args: args{
				s:     `"hi" 'there'`,
				quote: '"',
			},
			want: `"\"hi\" 'there'"`,
		},
		{
			name: "named escapes",
			args: args{
				s:     "\a\b\t\n\v\f",
				quote: '"',
			},
			want: `"\a\b\t\n\v\f"`,
		},
		{
			name: "ascii escapes",
			args: args{
				s:     "\x00\x01\x7f",
				quote: '"',
			},
			want: `"\0\1\127"`,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Quote(tt.args.s, tt.args.quote); got != tt.want {
				t.Errorf("Quote() = %v, want %v", got, tt.want)
			}
		})
	}
}
