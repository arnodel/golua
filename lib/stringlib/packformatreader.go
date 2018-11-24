package stringlib

import (
	"encoding/binary"
	"math"
)

type packFormatReader struct {
	format       string           // Specifies the packing format
	i            int              // Current index in the format string
	byteOrder    binary.ByteOrder // Current byteOrder of outputting numbers
	maxAlignment uint             // Current max alignment (used in pack.align())
	err          error            // if non-nil, the error encountered while packing
	optSize      uint             // Value of current option size
	alignOnly    bool             // true after "X" option is parsed
}

func (p *packFormatReader) hasNext() bool {
	return p.i < len(p.format)
}

func (p *packFormatReader) nextOption() byte {
	opt := p.format[p.i]
	p.i++
	return opt
}

func (p *packFormatReader) smallOptSize(defaultSize uint) (ok bool) {
	if p.getOptSize() {
		if ok = p.optSize >= 1 && p.optSize <= 16; !ok {
			p.err = errBadOptionArg
		}
		return
	}
	p.optSize = defaultSize
	if ok = defaultSize != 0; !ok {
		p.err = errMissingSize // TODO: check this condition occurs
	}
	return
}

const maxDecuplable = math.MaxUint64 / 10

func (p *packFormatReader) getOptSize() bool {
	var n uint
	ok := false
	for ; p.i < len(p.format); p.i++ {
		c := p.format[p.i]
		if c >= '0' && c <= '9' {
			ok = true
			if n > maxDecuplable {
				p.err = errOverflow
				return false
			}
			cc := uint(c - '0')
			n = n*10 + cc
			if n < cc {
				p.err = errOverflow
				return false
			}
		} else {
			break
		}
	}
	p.optSize = n
	return ok
}

func (p *packFormatReader) mustGetOptSize() bool {
	ok := p.getOptSize()
	if !ok && p.err == nil {
		p.err = errMissingSize
	}
	return ok
}
