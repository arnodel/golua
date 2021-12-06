package stringlib

type packsizer struct {
	packFormatReader
	size uint
}

func PackSize(format string) (uint, error) {
	s := &packsizer{packFormatReader: packFormatReader{
		format:       format,
		byteOrder:    nativeEndian,
		maxAlignment: defaultMaxAlignement,
	}}
	for s.hasNext() {
		switch c := s.nextOption(); c {
		case '<', '>', '=', ' ':
			// Nothing to do
		case '!':
			if s.smallOptSize(defaultMaxAlignement) {
				s.maxAlignment = s.optSize
			}
		case 'b', 'B':
			_ = s.align(0) && s.inc(1)
		case 'h', 'H':
			_ = s.align(2) && s.inc(2)
		case 'l', 'j', 'L', 'J', 'T', 'd', 'n':
			_ = s.align(8) && s.inc(8)
		case 'f':
			_ = s.align(4) && s.inc(4)
		case 'i', 'I':
			_ = s.smallOptSize(8) && s.align(s.optSize) && s.inc(s.optSize)
		case 'c':
			_ = s.align(0) && s.mustGetOptSize() && s.inc(s.optSize)
		case 'x':
			_ = s.align(0) && s.inc(1)
		case 'X':
			s.alignOnly = true
		case 's', 'z':
			s.err = errVariableLength
		default:
			s.err = errBadFormatString(c)
		}
		if s.err != nil {
			return 0, s.err
		}
	}

	return s.size, nil
}

func (s *packsizer) align(n uint) bool {
	if n != 0 {
		if n > s.maxAlignment {
			n = s.maxAlignment
		}
		if (n-1)&n != 0 { // (n-1)&n == 0 iff n is a power of 2 (or 0)
			s.err = errBadAlignment
			return false
		}
		if r := s.size % n; r != 0 {
			s.size += n - r
		}
	}
	if s.alignOnly {
		s.alignOnly = false
		return false
	}
	return true

}

func (s *packsizer) inc(n uint) bool {
	s.size += n
	return true
}
