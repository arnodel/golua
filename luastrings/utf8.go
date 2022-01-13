package luastrings

const (
	t1 = 0b00000000
	tx = 0b10000000
	t2 = 0b11000000
	t3 = 0b11100000
	t4 = 0b11110000
	t5 = 0b11111000
	t6 = 0b11111100

	maskx = 0b00111111
	// mask2 = 0b00011111
	// mask3 = 0b00001111
	// mask4 = 0b00000111
	// mask5 = 0b00000011
	// mask6 = 0b00000001

	rune1Max = 1<<7 - 1
	rune2Max = 1<<11 - 1
	rune3Max = 1<<16 - 1
	rune4Max = 1<<21 - 1
	rune5Max = 1<<26 - 1
	// rune6Max = 1<<31 - 1
)

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
