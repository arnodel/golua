package stringlib

import (
	"bytes"
	"encoding/binary"
	"io"

	rt "github.com/arnodel/golua/runtime"
)

type unpacker struct {
	packFormatReader
	pack   []byte     // The packed data
	j      int        // Current index in the packed data
	values []rt.Value // Values unpacked so far
	intVal rt.Int     // Last unpacked integral value (for options 'i' and 'I')
	strVal rt.String  // Last unpacked string value
}

func UnpackString(format, pack string) ([]rt.Value, int, error) {
	u := &unpacker{
		packFormatReader: packFormatReader{
			format:       format,
			byteOrder:    nativeEndian,
			maxAlignment: defaultMaxAlignement,
		},
		pack: []byte(pack),
	}
	for u.hasNext() {
		switch u.nextOption() {
		case '<':
			u.byteOrder = binary.LittleEndian
		case '>':
			u.byteOrder = binary.BigEndian
		case '=':
			u.byteOrder = nativeEndian
		case '!':
			if u.smallOptSize(defaultMaxAlignement) {
				u.maxAlignment = u.optSize
			}
		case 'b':
			var x byte
			_ = u.align(0) &&
				u.read(&x) &&
				u.add(rt.Int(x))
		case 'B':
			var x uint8
			_ = u.align(0) &&
				u.read(&x) &&
				u.add(rt.Int(x))
		case 'h':
			var x int16
			_ = u.align(2) &&
				u.read(&x) &&
				u.add(rt.Int(x))
		case 'H':
			var x uint16
			_ = u.align(2) &&
				u.read(&x) &&
				u.add(rt.Int(x))
		case 'l', 'j':
			var x int64
			_ = u.align(8) &&
				u.read(&x) &&
				u.add(rt.Int(x))
		case 'L', 'J', 'T':
			var x uint64
			_ = u.align(8) &&
				u.read(&x) &&
				u.add(rt.Int(x))
		case 'i':
			_ = u.smallOptSize(4) &&
				u.align(u.optSize) &&
				u.readVarInt() &&
				u.add(u.intVal)
		case 'I':
			_ = u.smallOptSize(4) &&
				u.align(u.optSize) &&
				u.readVarUint() &&
				u.add(u.intVal)
		case 'f':
			var x float32
			_ = u.align(4) && u.read(&x) && u.add(rt.Float(x))
		case 'd', 'n':
			var x float64
			_ = u.align(8) && u.read(&x) && u.add(rt.Float(x))
		case 'c':
			_ = u.align(0) &&
				u.mustGetOptSize() &&
				u.readStr(int(u.optSize)) &&
				u.add(u.strVal)
		case 'z':
			if !u.align(0) {
				break
			}
			var zi = u.j
			for {
				if zi >= len(u.pack) {
					return nil, 0, errUnexpectedPackEnd
				}
				if u.pack[zi] == 0 {
					break
				}
				zi++
			}
			b := make([]byte, zi-u.j)
			_ = u.read(b) && u.add(rt.String(b))
		case 's':
			_ = u.smallOptSize(8) &&
				u.align(u.optSize) &&
				u.readVarUint() &&
				u.readStr(int(u.intVal)) &&
				u.add(u.strVal)
		case 'x':
			_ = u.skip(1)
		case 'X':
			u.alignOnly = true
		default:
			u.err = errBadFormatString
		}
		if u.err != nil {
			return nil, 0, u.err
		}
	}
	if u.alignOnly {
		return nil, 0, errExpectedOption
	}
	return u.values, u.j, nil
}

// Read implements io.Read
func (u *unpacker) Read(b []byte) (n int, err error) {
	if u.j >= len(u.pack) {
		return 0, io.EOF
	}
	n = copy(b, u.pack[u.j:])
	u.j += n
	return
}

func (u *unpacker) align(n uint) bool {
	if n != 0 {
		if n > u.maxAlignment {
			n = u.maxAlignment
		}
		if r := uint(u.j) % n; r != 0 {
			if !u.skip(n - r) {
				return false
			}
		}
	}
	if u.alignOnly {
		u.alignOnly = false
		return false
	}
	return true
}

func (u *unpacker) read(x interface{}) bool {
	if err := binary.Read(u, u.byteOrder, x); err != nil {
		u.err = err
		return false
	}
	return true
}

func (u *unpacker) readStr(n int) (ok bool) {
	b := make([]byte, n)
	ok = u.read(b)
	if ok {
		u.strVal = rt.String(b)
	}
	return
}

func (u *unpacker) readVarUint() (ok bool) {
	switch n := u.optSize; {
	case n == 4:
		var x uint32
		ok = u.read(&x) &&
			u.setIntVal(rt.Int(x))
	case n == 8:
		var x uint64
		ok = u.read(&x) &&
			u.setIntVal(rt.Int(x))
	case n > 8:
		var x uint64
		ok = (u.byteOrder == binary.LittleEndian || u.skip(n-8)) &&
			u.read(&x) &&
			u.setIntVal(rt.Int(x)) &&
			(u.byteOrder == binary.BigEndian || u.skip(n-8))
	default:
		// n < 8 so truncated
		var b [8]byte
		switch u.byteOrder {
		case binary.LittleEndian:
			_, u.err = u.Read(b[:n])
		default:
			_, u.err = u.Read(b[8-n:])
		}
		if u.err != nil {
			return false
		}
		r := bytes.NewReader(b[:])
		var x uint64
		_ = binary.Read(r, u.byteOrder, &x) // There should be no error!
		u.intVal = rt.Int(x)
		return true
	}
	return
}

func (u *unpacker) readVarInt() (ok bool) {
	switch n := u.optSize; {
	case n == 4:
		var x int32
		ok = u.read(&x) &&
			u.setIntVal(rt.Int(x))
	case n == 8:
		var x int64
		ok = u.read(&x) &&
			u.setIntVal(rt.Int(x))
	case n > 8:
		var x int64
		ok = (u.byteOrder == binary.LittleEndian || u.skip(n-8)) &&
			u.read(&x) &&
			u.setIntVal(rt.Int(x)) &&
			(u.byteOrder == binary.BigEndian || u.skip(n-8))
	default:
		// n < 8 so truncated
		var b [8]byte
		switch u.byteOrder {
		case binary.LittleEndian:
			_, u.err = u.Read(b[:n])
			if u.err != nil {
				return false
			}
			if b[n-1]&(1<<7) != 0 {
				for i := n; i < 8; i++ {
					b[i] = 0xff
				}
			}
		default:
			_, u.err = u.Read(b[8-n:])
			if u.err != nil {
				return false
			}
			if b[8-n]&(1<<7) != 0 {
				for i := uint(0); i < 8-n; i++ {
					b[i] = 0xff
				}
			}
		}
		r := bytes.NewReader(b[:])
		var x int64
		_ = binary.Read(r, u.byteOrder, &x) // There should be no error!
		u.intVal = rt.Int(x)
		return true
	}
	return
}

func (u *unpacker) add(v rt.Value) bool {
	u.values = append(u.values, v)
	return true
}

func (u *unpacker) skip(n uint) (ok bool) {
	u.j += int(n)
	ok = u.j <= len(u.pack)
	if !ok {
		u.err = errUnexpectedPackEnd
	}
	return
}

func (u *unpacker) setIntVal(v rt.Int) bool {
	u.intVal = v
	return true
}
