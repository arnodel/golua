package runtime

import "testing"

func Test_floordivInt(t *testing.T) {
	type args struct {
		x int64
		y int64
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "-10//10",
			args: args{
				x: -10,
				y: 10,
			},
			want: -1,
		},
		{
			name: "-10//-10",
			args: args{
				x: -10,
				y: -10,
			},
			want: 1,
		},
		{
			name: "-10//-3",
			args: args{
				x: -10,
				y: -3,
			},
			want: 3,
		},

		{
			name: "-10//100",
			args: args{
				x: -10,
				y: 100,
			},
			want: -1,
		},
		{
			name: "-10//8",
			args: args{
				x: -10,
				y: 8,
			},
			want: -2,
		},
		{
			name: "-10//-100",
			args: args{
				x: -10,
				y: -100,
			},
			want: 0,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := floordivInt(tt.args.x, tt.args.y); got != tt.want {
				t.Errorf("floordivInt() = %v, want %v", got, tt.want)
			}
		})
	}
}
