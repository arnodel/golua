package pattern

type byteSet [4]uint64

func (s *byteSet) merge(t byteSet) {
	s[0] |= t[0]
	s[1] |= t[1]
	s[2] |= t[2]
	s[3] |= t[3]
}

func (s *byteSet) add(b byte) {
	s[b>>6] |= uint64(1) << (b & 0x3F)
}

func (s *byteSet) complement() {
	s[0] ^= full64
	s[1] ^= full64
	s[2] ^= full64
	s[3] ^= full64
}

func (s byteSet) contains(b byte) bool {
	return s[b>>6]>>(b&0x3F)&1 != 0
}

// byte sets below are built with this function
//
func byteRange(a, b byte) (s byteSet) {
	for i := a; i < b; i++ {
		s.add(i)
	}
	s.add(b)
	return
}

func complement(b byteSet) byteSet {
	b.complement()
	return b
}

const full64 uint64 = ^uint64(0)

var (
	letterSet    = byteSet{0x0, 0x7fffffe07fffffe, 0x0, 0x0}                 // r('a', 'z', 'A', 'Z')
	controlSet   = byteSet{0xffffffff, 0x8000000000000000, 0x0, 0x0}         // r(0, 32, 127, 127)
	digitSet     = byteSet{0x3ff000000000000, 0x0, 0x0, 0x0}                 // r('0', '9')
	printableSet = byteSet{0xfffffffe00000000, 0x7fffffffffffffff, 0x0, 0x0} // r(33, 126)
	lowerSet     = byteSet{0x0, 0x7fffffe00000000, 0x0, 0x0}                 // r('a', z')
	punctSet     = byteSet{0xfc00fffe00000000, 0x78000001f8000001, 0x0, 0x0} // r('!', '/', ':', '@', '[', '`', '{', '~')
	spaceSet     = byteSet{0x100003e00, 0x0, 0x0, 0x0}                       // r(9, 13, 32, 32)
	upperSet     = byteSet{0x0, 0x7fffffe, 0x0, 0x0}                         // r('A', 'Z')
	alphanumSet  = byteSet{0x3ff000000000000, 0x7fffffe07fffffe, 0x0, 0x0}   // r('A', 'Z', 'a', 'z', '0', '9')
	hexSet       = byteSet{0x3ff000000000000, 0x7e0000007e, 0x0, 0x0}
	zeroSet      = byteSet{0x1, 0x0, 0x0, 0x0}

	fullSet = byteSet{full64, full64, full64, full64}
)

var namedByteSet = map[byte]byteSet{
	'a': letterSet,
	'c': controlSet,
	'd': digitSet,
	'g': printableSet,
	'l': lowerSet,
	'p': punctSet,
	's': spaceSet,
	'u': upperSet,
	'w': alphanumSet,
	'x': hexSet,
	'z': zeroSet,
	'A': complement(letterSet),
	'C': complement(controlSet),
	'D': complement(digitSet),
	'G': complement(printableSet),
	'L': complement(lowerSet),
	'P': complement(punctSet),
	'S': complement(spaceSet),
	'U': complement(upperSet),
	'W': complement(alphanumSet),
	'X': complement(hexSet),
	'Z': complement(zeroSet),
}
