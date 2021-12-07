package stringlib

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"unsafe"

	rt "github.com/arnodel/golua/runtime"
)

type unpacker struct {
	packFormatReader
	pack   []byte     // The packed data
	j      int        // Current index in the packed data
	values []rt.Value // Values unpacked so far
	intVal int64      // Last unpacked integral value (for options 'i' and 'I')
	strVal string     // Last unpacked string value

	budget uint64 // Number of bytes we are allowed to write before stopping (0 means unbounded)
	used   uint64 // Number of bytes written so far
}

func UnpackString(format, pack string, j int, budget uint64) (vals []rt.Value, nextPos int, used uint64, err error) {
	u := &unpacker{
		packFormatReader: packFormatReader{
			format:       format,
			byteOrder:    nativeEndian,
			maxAlignment: defaultMaxAlignement,
		},
		pack: []byte(pack),
		j:    j,

		budget: budget,
	}
	for u.hasNext() {
		switch c := u.nextOption(); c {
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
			var x int8
			_ = u.align(0) &&
				u.read(1, &x) &&
				u.add(rt.IntValue(int64(x)))
		case 'B':
			var x uint8
			_ = u.align(0) &&
				u.read(1, &x) &&
				u.add(rt.IntValue(int64(x)))
		case 'h':
			var x int16
			_ = u.align(2) &&
				u.read(2, &x) &&
				u.add(rt.IntValue(int64(x)))
		case 'H':
			var x uint16
			_ = u.align(2) &&
				u.read(2, &x) &&
				u.add(rt.IntValue(int64(x)))
		case 'l', 'j':
			var x int64
			_ = u.align(8) &&
				u.read(8, &x) &&
				u.add(rt.IntValue(x))
		case 'L', 'J', 'T':
			var x uint64
			_ = u.align(8) &&
				u.read(8, &x) &&
				u.add(rt.IntValue(int64(x)))
		case 'i':
			_ = u.smallOptSize(8) &&
				u.align(u.optSize) &&
				u.readVarInt() &&
				u.add(rt.IntValue(u.intVal))
		case 'I':
			_ = u.smallOptSize(8) &&
				u.align(u.optSize) &&
				u.readVarUint() &&
				u.add(rt.IntValue(u.intVal))
		case 'f':
			var x float32
			_ = u.align(4) &&
				u.read(4, &x) &&
				u.add(rt.FloatValue(float64(x)))
		case 'd', 'n':
			var x float64
			_ = u.align(8) &&
				u.read(8, &x) &&
				u.add(rt.FloatValue(x))
		case 'c':
			_ = u.align(0) &&
				u.mustGetOptSize() &&
				u.readStr(int(u.optSize)) &&
				u.add(rt.StringValue(u.strVal))
		case 'z':
			if !u.align(0) {
				u.err = errExpectedOption
				break
			}
			var zi = u.j
			for {
				if zi >= len(u.pack) {
					return nil, 0, u.used, errUnexpectedPackEnd
				}
				if u.pack[zi] == 0 {
					break
				}
				zi++
				if !u.consumeBudget(1) {
					return nil, 0, u.used, u.err
				}
			}
			b := make([]byte, zi-u.j)
			_ = u.read(0, b) &&
				u.add(rt.StringValue(string(b))) &&
				u.skip(1)
		case 's':
			_ = u.smallOptSize(8) &&
				u.align(u.optSize) &&
				u.readVarUint() &&
				u.readStr(int(u.intVal)) &&
				u.add(rt.StringValue(u.strVal))
		case 'x':
			_ = u.skip(1)
		case 'X':
			if u.alignOnly {
				u.err = errExpectedOption
			} else {
				u.alignOnly = true
			}
		case ' ':
			if u.alignOnly {
				u.err = errExpectedOption
			}
		default:
			u.err = errBadFormatString(c)
		}
		if u.err != nil {
			return nil, 0, u.used, u.err
		}
	}
	if u.alignOnly {
		return nil, 0, u.used, errExpectedOption
	}
	return u.values, u.j, u.used, nil
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

func (u *unpacker) read(sz uint64, x interface{}) bool {
	if sz > 0 && !u.consumeBudget(sz) {
		return false
	}
	if err := binary.Read(u, u.byteOrder, x); err != nil {
		if err == io.ErrUnexpectedEOF {
			err = errUnexpectedPackEnd
		}
		u.err = err
		return false
	}
	return true
}

func (u *unpacker) readStr(n int) (ok bool) {
	if !u.consumeBudget(uint64(n)) {
		return false
	}
	b := make([]byte, n)
	ok = u.read(0, b)
	if ok {
		u.strVal = string(b)
	}
	return
}

func (u *unpacker) readVarUint() (ok bool) {
	if !u.consumeBudget(8) {
		return false
	}
	switch n := u.optSize; {
	case n == 4:
		var x uint32
		ok = u.read(0, &x) &&
			u.setIntVal(int64(x))
	case n == 8:
		var x uint64
		ok = u.read(0, &x) &&
			u.setIntVal(int64(x))
	case n > 8:
		var x uint64
		ok = (u.byteOrder == binary.LittleEndian || u.skip0(n-8)) &&
			u.read(0, &x) &&
			u.setIntVal(int64(x)) &&
			(u.byteOrder == binary.BigEndian || u.skip0(n-8))
	default:
		// n < 8 so truncated
		var b [8]byte
		var rn int
		switch u.byteOrder {
		case binary.LittleEndian:
			rn, u.err = u.Read(b[:n])
		default:
			rn, u.err = u.Read(b[8-n:])
		}
		if rn < int(n) {
			u.err = errUnexpectedPackEnd
		}
		if u.err != nil {
			return false
		}
		r := bytes.NewReader(b[:])
		var x uint64
		_ = binary.Read(r, u.byteOrder, &x) // There should be no error!
		u.intVal = int64(x)
		return true
	}
	return
}

func (u *unpacker) readVarInt() (ok bool) {
	if !u.consumeBudget(8) {
		return false
	}
	switch n := u.optSize; {
	case n == 4:
		var x int32
		ok = u.read(0, &x) &&
			u.setIntVal(int64(x))
	case n == 8:
		var x int64
		ok = u.read(0, &x) &&
			u.setIntVal(x)
	case n > 8:
		var x uint64
		var signExt uint8
		if u.byteOrder == binary.BigEndian {
			ok = u.readSignExt(n-8, &signExt) && u.read(0, &x)
		} else {
			ok = u.read(0, &x) && u.readSignExt(n-8, &signExt)
		}
		if !ok {
			return
		}
		if signExt == 0 {
			ok = x <= math.MaxInt64
			if !ok {
				u.err = errDoesNotFit
				return
			}
			u.intVal = int64(x)
		} else {
			if ok = x > math.MaxInt64; ok {
				xx := *(*int64)(unsafe.Pointer(&x))
				u.intVal = xx
			} else {
				u.err = errDoesNotFit
			}
		}
	default:
		// n < 8 so truncated
		var b [8]byte
		var rn int
		switch u.byteOrder {
		case binary.LittleEndian:
			rn, u.err = u.Read(b[:n])
			if rn < int(n) {
				u.err = errUnexpectedPackEnd
			}
			if u.err != nil {
				return false
			}
			if b[n-1]&(1<<7) != 0 {
				for i := n; i < 8; i++ {
					b[i] = 0xff
				}
			}
		default:
			rn, u.err = u.Read(b[8-n:])
			if rn < int(n) {
				u.err = errUnexpectedPackEnd
			}
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
		u.intVal = x
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

func (u *unpacker) skip0(n uint) (ok bool) {
	j := u.j
	u.j += int(n)
	ok = u.j <= len(u.pack)
	if !ok {
		u.err = errUnexpectedPackEnd
		return
	}
	for j < u.j {
		if ok = u.pack[j] == 0; !ok {
			u.err = errDoesNotFit
			return
		}
		j++
	}
	return
}

func (u *unpacker) readSignExt(n uint, sign *uint8) (ok bool) {
	// No need to consume budget, that's alrady been done in the calling
	// function.
	j := u.j
	u.j += int(n)
	ok = n > 0 && u.j <= len(u.pack)
	if !ok {
		u.err = errUnexpectedPackEnd
		return
	}
	*sign = u.pack[j]
	ok = *sign == 0 || *sign == 0xff
	for j++; ok && j < u.j; j++ {
		ok = u.pack[j] == *sign
	}
	if !ok {
		u.err = errDoesNotFit
	}
	return
}

func (u *unpacker) setIntVal(v int64) bool {
	u.intVal = v
	return true
}

func (u *unpacker) consumeBudget(amount uint64) bool {
	if u.budget == 0 {
		return true
	}
	u.used += amount
	if u.used > u.budget {
		u.err = errBudgetConsumed
		u.used = u.budget
		return false
	}
	return true
}
