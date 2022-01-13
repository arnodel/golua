package luastrings

import (
	"reflect"
	"testing"
)

func TestUTF8EncodeInt32(t *testing.T) {
	type want struct {
		n     int
		bytes []byte
	}
	tests := []struct {
		name string
		i    int32
		want want
	}{
		{
			name: "1 byte",
			i:    0x55,
			want: want{
				n:     1,
				bytes: []byte{0x55},
			},
		},
		{
			name: "2 bytes low",
			i:    0x80,
			want: want{
				n:     2,
				bytes: []byte{0xc2, 0x80},
			},
		},
		{
			name: "2 bytes high",
			i:    0x7ff,
			want: want{
				n:     2,
				bytes: []byte{0xdf, 0xbf},
			},
		},
		{
			name: "3 bytes low",
			i:    0x800,
			want: want{
				n:     3,
				bytes: []byte{0xe0, 0xa0, 0x80},
			},
		},
		{
			name: "3 bytes high",
			i:    0xffff,
			want: want{
				n:     3,
				bytes: []byte{0xef, 0xbf, 0xbf},
			},
		},
		{
			name: "4 bytes low",
			i:    0x10000,
			want: want{
				n:     4,
				bytes: []byte{0xf0, 0x90, 0x80, 0x80},
			},
		},
		{
			name: "4 bytes high",
			i:    0x1fffff,
			want: want{
				n:     4,
				bytes: []byte{0xf7, 0xbf, 0xbf, 0xbf},
			},
		},
		{
			name: "5 bytes low",
			i:    0x200000,
			want: want{
				n:     5,
				bytes: []byte{0xf8, 0x88, 0x80, 0x80, 0x80},
			},
		},
		{
			name: "5 bytes high",
			i:    0x3ffffff,
			want: want{
				n:     5,
				bytes: []byte{0xfb, 0xbf, 0xbf, 0xbf, 0xbf},
			},
		},
		{
			name: "6 bytes low",
			i:    0x4000000,
			want: want{
				n:     6,
				bytes: []byte{0xfc, 0x84, 0x80, 0x80, 0x80, 0x80},
			},
		},
		{
			name: "6 bytes high",
			i:    0x7fffffff,
			want: want{
				n:     6,
				bytes: []byte{0xfd, 0xbf, 0xbf, 0xbf, 0xbf, 0xbf},
			},
		},
		{
			name: "negative i",
			i:    -3,
			want: want{
				n: 0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p [6]byte
			n := UTF8EncodeInt32(p[:], tt.i)
			if n != tt.want.n {
				t.Errorf("UTF8EncodeInt32() = %v, want %v", n, tt.want.n)
			}
			if n > 0 {
				if !reflect.DeepEqual(p[:n], tt.want.bytes) {
					t.Errorf("UTF8EncodeInt32() bytes = %v, want %v", p[:n], tt.want.bytes)

				}
			}
		})
	}
}
