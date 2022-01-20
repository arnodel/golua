package luastrings

import (
	"fmt"
	"reflect"
	"testing"
)

type Utf8Map struct {
	r   rune
	str string
}

var utf8map = []Utf8Map{
	{0x00, "\x00"},
	{0x55, "\x55"},
	{0x7f, "\x7f"},
	{0x80, "\xc2\x80"},
	{0x7ff, "\xdf\xbf"},
	{0x800, "\xe0\xa0\x80"},
	{0xffff, "\xef\xbf\xbf"},
	{0x10000, "\xf0\x90\x80\x80"},
	{0x1fffff, "\xf7\xbf\xbf\xbf"},
	{0x200000, "\xf8\x88\x80\x80\x80"},
	{0x3ffffff, "\xfb\xbf\xbf\xbf\xbf"},
	{0x4000000, "\xfc\x84\x80\x80\x80\x80"},
	{0x7fffffff, "\xfd\xbf\xbf\xbf\xbf\xbf"},
}

func TestUTF8EncodeInt32(t *testing.T) {
	for _, tt := range utf8map {
		t.Run(fmt.Sprintf("%x", tt.r), func(t *testing.T) {
			var p [6]byte
			n := UTF8EncodeInt32(p[:], tt.r)
			if n != len(tt.str) {
				t.Errorf("UTF8EncodeInt32() = %v, want %v", n, len(tt.str))
			}
			if n > 0 {
				if !reflect.DeepEqual(p[:n], []byte(tt.str)) {
					t.Errorf("UTF8EncodeInt32() bytes = %v, want %v", p[:n], []byte(tt.str))

				}
			}
		})
	}
	t.Run(fmt.Sprintf("%x", -1), func(t *testing.T) {
		var p [6]byte
		n := UTF8EncodeInt32(p[:], -1)
		if n != 0 {
			t.Errorf("UTF8EncodeInt32() = %v, want %v", n, 0)
		}
	})

}

func TestDecodeRuneInString(t *testing.T) {
	for _, tt := range utf8map {
		t.Run(fmt.Sprintf("%x", tt.r), func(t *testing.T) {
			gotR, gotSize := DecodeRuneInString(tt.str)
			if gotR != tt.r {
				t.Errorf("DecodeRuneInString() gotR = %v, want %v", gotR, tt.r)
			}
			if gotSize != len(tt.str) {
				t.Errorf("DecodeRuneInString() gotSize = %v, want %v", gotSize, len(tt.str))
			}
		})
		t.Run(fmt.Sprintf("byte missing %x", tt.r), func(t *testing.T) {
			gotR, gotSize := DecodeRuneInString(tt.str[:len(tt.str)-1])
			if gotR != RuneError {
				t.Errorf("DecodeRuneInString() gotR = %v, want %v", gotR, tt.r)
			}
			wantSize := 1
			if len(tt.str) == 1 {
				wantSize = 0
			}
			if gotSize != wantSize {
				t.Errorf("DecodeRuneInString() gotSize = %v, want %v", gotSize, wantSize)
			}
		})
		t.Run(fmt.Sprintf("out of range %x", tt.r), func(t *testing.T) {
			gotR, gotSize := DecodeRuneInString(tt.str[:len(tt.str)-1] + "\xff")
			if gotR != RuneError {
				t.Errorf("DecodeRuneInString() gotR = %v, want %v", gotR, tt.r)
			}
			if gotSize != 1 {
				t.Errorf("DecodeRuneInString() gotSize = %v, want %v", gotSize, 1)
			}
		})
	}
}
