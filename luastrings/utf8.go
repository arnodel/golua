package luastrings

import "unicode/utf8"

const (
	UTFMax = 6

	t1 = 0b00000000
	tx = 0b10000000
	t2 = 0b11000000
	t3 = 0b11100000
	t4 = 0b11110000
	t5 = 0b11111000
	t6 = 0b11111100

	maskx = 0b00111111
	mask2 = 0b00011111
	mask3 = 0b00001111
	mask4 = 0b00000111
	mask5 = 0b00000011
	mask6 = 0b00000001

	rune1Max = 1<<7 - 1
	rune2Max = 1<<11 - 1
	rune3Max = 1<<16 - 1
	rune4Max = 1<<21 - 1
	rune5Max = 1<<26 - 1
	// rune6Max = 1<<31 - 1

	// The default lowest and highest continuation byte.
	locb = 0b10000000
	hicb = 0b10111111

	// These names of these constants are chosen to give nice alignment in the
	// table below. The first nibble is an index into acceptRanges or F for
	// special one-byte cases. The second nibble is the Rune length or the
	// Status for the special one-byte case.
	xx = 0xF1 // invalid: size 1
	as = 0xF0 // ASCII: size 1
	s1 = 0x02 // accept 0, size 2

	s2 = 0x13 // accept 1, size 3
	s3 = 0x03 // accept 0, size 3
	s4 = 0x23 // accept 2, size 3

	s5 = 0x34 // accept 3, size 4
	s6 = 0x04 // accept 0, size 4
	s7 = 0x44 // accept 4, size 4

	// Added for Lua
	s8 = 0x05 // accept 0, size 5
	s9 = 0x06 // accept 0, size 6
)

// first is information about the first byte in a UTF-8 sequence.  This table is
// copied from the utf8 std library.
var first = [256]uint8{
	//   1   2   3   4   5   6   7   8   9   A   B   C   D   E   F
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x00-0x0F
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x10-0x1F
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x20-0x2F
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x30-0x3F
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x40-0x4F
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x50-0x5F
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x60-0x6F
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x70-0x7F
	//   1   2   3   4   5   6   7   8   9   A   B   C   D   E   F
	xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, // 0x80-0x8F
	xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, // 0x90-0x9F
	xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, // 0xA0-0xAF
	xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, // 0xB0-0xBF
	xx, xx, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, // 0xC0-0xCF
	s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, // 0xD0-0xDF
	s2, s3, s3, s3, s3, s3, s3, s3, s3, s3, s3, s3, s3, s4, s3, s3, // 0xE0-0xEF
	s5, s6, s6, s6, s7, s6, s6, s6, s8, s8, s8, s8, s9, s9, xx, xx, // 0xF0-0xFF
}

// 1111 0xxx: 0xF0-0xF7
// 1111 10xx: 0xF8-0xFB
// 1111 110x: 0xFC-0xFD

const RuneError = utf8.RuneError

// Encode a unicode point with value i into a sequence of bytes, writing into p.
// p must be big enough (length 6 accomodates all values).  Returns the number
// of bytes written. A non-positive value means an error.
//
// Any non-negative int32 can be encoded, that is why the golang utf8 package
// cannot be used.
func UTF8EncodeInt32(p []byte, i int32) int {
	switch {
	case i < 0:
		return 0
	case i <= rune1Max:
		p[0] = t1 | byte(i)
		return 1
	case i <= rune2Max:
		_ = p[1]
		p[0] = t2 | byte(i>>6)
		p[1] = tx | byte(i)&maskx
		return 2
	case i <= rune3Max:
		_ = p[2]
		p[0] = t3 | byte(i>>12)
		p[1] = tx | byte(i>>6)&maskx
		p[2] = tx | byte(i)&maskx
		return 3
	case i <= rune4Max:
		_ = p[3]
		p[0] = t4 | byte(i>>18)
		p[1] = tx | byte(i>>12)&maskx
		p[2] = tx | byte(i>>6)&maskx
		p[3] = tx | byte(i)&maskx
		return 4
	case i <= rune5Max:
		_ = p[4]
		p[0] = t5 | byte(i>>24)
		p[1] = tx | byte(i>>18)&maskx
		p[2] = tx | byte(i>>12)&maskx
		p[3] = tx | byte(i>>6)&maskx
		p[4] = tx | byte(i)&maskx
		return 5
	default: // i <= rune6Max:
		_ = p[5]
		p[0] = t6 | byte(i>>30)
		p[1] = tx | byte(i>>24)&maskx
		p[2] = tx | byte(i>>18)&maskx
		p[3] = tx | byte(i>>12)&maskx
		p[4] = tx | byte(i>>6)&maskx
		p[5] = tx | byte(i)&maskx
		return 6
	}
}

// GetDecodeRuneInString return a decode function that is strict or lax about
// the utf8 encoding depending on the value of lax.  For details see the UTF-8
// support section in the Lua 5.4 manual.
func GetDecodeRuneInString(lax bool) func(string) (rune, int) {
	if lax {
		return DecodeRuneInString
	} else {
		return utf8.DecodeRuneInString
	}
}

// DecodeRuneInString is like DecodeRune but its input is a string. If s is
// empty it returns (RuneError, 0). Otherwise, if the encoding is invalid, it
// returns (RuneError, 1). Both are impossible results for correct, non-empty
// UTF-8.
//
// An encoding is invalid if it is incorrect UTF-8, encodes a rune that is
// out of range, or is not the shortest possible UTF-8 encoding for the
// value. No other validation is performed.
func DecodeRuneInString(s string) (r rune, size int) {
	n := len(s)
	if n < 1 {
		return RuneError, 0
	}
	s0 := s[0]
	x := first[s0]
	if x >= as {
		// The following code simulates an additional check for x == xx and
		// handling the ASCII and invalid cases accordingly. This mask-and-or
		// approach prevents an additional branch.
		mask := rune(x) << 31 >> 31 // Create 0x0000 or 0xFFFF.
		return rune(s[0])&^mask | RuneError&mask, 1
	}
	sz := int(x & 7)
	if n < sz {
		return RuneError, 1
	}
	s1 := s[1]
	if s1 < locb || hicb < s1 {
		return RuneError, 1
	}
	if sz <= 2 { // <= instead of == to help the compiler eliminate some bounds checks
		return rune(s0&mask2)<<6 | rune(s1&maskx), 2
	}
	s2 := s[2]
	if s2 < locb || hicb < s2 {
		return RuneError, 1
	}
	if sz <= 3 {
		return rune(s0&mask3)<<12 | rune(s1&maskx)<<6 | rune(s2&maskx), 3
	}
	s3 := s[3]
	if s3 < locb || hicb < s3 {
		return RuneError, 1
	}
	if sz <= 4 {
		return rune(s0&mask4)<<18 | rune(s1&maskx)<<12 | rune(s2&maskx)<<6 | rune(s3&maskx), 4
	}
	s4 := s[4]
	if s4 < locb || hicb < s4 {
		return RuneError, 1
	}
	if sz <= 5 {
		return rune(s0&mask5)<<24 | rune(s1&maskx)<<18 | rune(s2&maskx)<<12 | rune(s3&maskx)<<6 | rune(s4&maskx), 5
	}
	s5 := s[5]
	if s5 < locb || hicb < s5 {
		return RuneError, 1
	}
	return rune(s0&mask6)<<30 | rune(s1&maskx)<<24 | rune(s2&maskx)<<18 | rune(s3&maskx)<<12 | rune(s4&maskx)<<6 | rune(s5&maskx), 6
}
