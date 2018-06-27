package stringlib

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"unsafe"

	rt "github.com/arnodel/golua/runtime"
)

/*
Notes:

<: sets little endian
>: sets big endian

=: sets native endian

![n]: sets maximum alignment to n (default is native alignment)
   between 1 and 16
   Does not need to be a power of 2 but if it's not it seems things break?

b: a signed byte (char)
   int8

B: an unsigned byte (char)
   uint8

h: a signed short (native size)
   int16

H: an unsigned short (native size)
   uint16

l: a signed long (native size)
   int64

L: an unsigned long (native size)
   uint64

j: a lua_Integer
   int64

J: a lua_Unsigned
   uint64

T: a size_t (native size)
   uint64

i[n]: a signed int with n bytes (default is native size)
   default int32
   n between 1 and 16 errors if an overflow

I[n]: an unsigned int with n bytes (default is native size)
   default uint32
   n between 1 and 16
f: a float (native size)
   float32
d: a double (native size)
   float64

n: a lua_Number
   float64

cn: a fixed-sized string with n bytes
   Not aligned

z: a zero-terminated string
   Not aligned

s[n]: a string preceded by its length coded as an unsigned integer with n bytes (default is a size_t)
   Aligned like I[n]

x: one byte of padding
   That is one zero byte

Xop: an empty item that aligns according to option op (which is otherwise ignored)
   That is, add padding for alignment but do not add a value

' ': (empty space) ignored
*/

func isLittleEndian() bool {
	var i int32 = 0x01020304
	u := unsafe.Pointer(&i)
	pb := (*byte)(u)
	b := *pb
	return (b == 0x04)
}

var nativeEndian binary.ByteOrder

const defaultMaxAlignement uint = 1

func init() {
	if isLittleEndian() {
		nativeEndian = binary.LittleEndian
	} else {
		nativeEndian = binary.BigEndian
	}
}

func optionArg(format string, i int) (int, int) {
	arg := 0
	for ; i < len(format); i++ {
		c := format[i]
		if c >= '0' && c <= '9' {
			arg = arg*10 + int(c-'0')
		} else {
			break
		}
	}
	return i, arg
}

func nextValue(values []rt.Value) (rt.Value, []rt.Value) {
	if len(values) > 0 {
		return values[0], values[1:]
	}
	return nil, nil
}

func pack(format string, values []rt.Value) ([]byte, error) {
	byteOrder := nativeEndian
	maxAlignment := defaultMaxAlignement
	alignOnly := false
	vIdx := 0
	var n uint
	var err error
	i := 0
	var w bytes.Buffer
	var optionArg = func() bool {
		n = 0
		ok := false
		for ; i < len(format); i++ {
			c := format[i]
			if c >= '0' && c <= '9' {
				ok = true
				n = n*10 + uint(c-'0')
			} else {
				break
			}
		}
		return ok
	}
	var optOptionArg = func() bool {
		optionArg()
		ok := 0 <= n && n <= 16
		if !ok {
			err = errBadOptionArg
		}
		return ok
	}
	var nextValue = func() rt.Value {
		var v rt.Value
		if len(values) > vIdx {
			v = values[vIdx]
		}
		vIdx++
		return v
	}
	var nextIntValue = func() (rt.Int, bool) {
		v := nextValue()
		n, tp := rt.ToInt(v)
		ok := tp == rt.IsInt
		if !ok {
			err = errBadType
		}
		return n, ok
	}
	var nextFloatValue = func() (rt.Float, bool) {
		v := nextValue()
		f, ok := rt.ToFloat(v)
		if !ok {
			err = errBadType
		}
		return f, ok
	}
	var nextStringValue = func() (rt.String, bool) {
		v := nextValue()
		s, ok := rt.AsString(v)
		if !ok {
			err = errBadType
		}
		return s, ok
	}
	var checkBounds = func(n rt.Int, min, max rt.Int) bool {
		ok := n >= min && n <= max
		if !ok {
			err = errOutOfBounds
		}
		return ok
	}
	var write = func(x interface{}) bool {
		err = binary.Write(&w, byteOrder, x)
		return err == nil
	}
	var align = func(n uint) bool {

		if alignOnly {
			alignOnly = false
			return false
		}
		return true

	}
	for ; i < len(format); i++ {
		switch format[i] {
		case '<':
			byteOrder = binary.LittleEndian
		case '>':
			byteOrder = binary.BigEndian
		case '=':
			byteOrder = nativeEndian
		case '!':
			if !optOptionArg() {
				return nil, err
			}
			if n > 0 {
				maxAlignment = n
			} else {
				maxAlignment = defaultMaxAlignement
			}
		case 'b':
			n, ok := nextIntValue()
			ok = ok && checkBounds(n, 0, math.MaxUint8) && write(uint8(n))
		case 'B':
			n, ok := nextIntValue()
			ok = ok && checkBounds(n, math.MinInt8, math.MaxInt8) && write(int8(n))
		case 'h':
			n, ok := nextIntValue()
			ok = ok && align(2) && checkBounds(n, 0, math.MaxUint16) && write(uint16(n))
		case 'H':
			n, ok := nextIntValue()
			ok = ok && align(2) && checkBounds(n, math.MinInt16, math.MaxInt16) && write(uint16(n))
		case 'l', 'j':
			n, ok := nextIntValue()
			ok = ok && align(8) && write(int64(n))
		case 'L', 'J', 'T':
			n, ok := nextIntValue()
			ok = ok && align(8) && checkBounds(n, 0, math.MaxInt64) && write(uint16(n))
		case 'i':
			if !optOptionArg() {
				return nil, err
			}
			m, ok := nextIntValue()
			if n == 0 {
				ok = ok && align(4) && checkBounds(m, math.MinInt32, math.MaxInt32) && write(int32(m))
			} else {
				ok = ok && align(n)
				if n >= 8 {
					fill := byte(0)
					if m < 0 {
						fill = 255
					}

					if byteOrder == binary.BigEndian {
						for j := n; j > 8; j-- {
							w.WriteByte(fill)
						}
					}

					ok = ok && write(int64(n))

					if byteOrder == binary.LittleEndian {
						for j := n; j > 8; j-- {
							w.WriteByte(fill)
						}
					}
				} else {
					max := 1 << (n<<3 - 1)
					ok = ok && checkBounds(m, rt.Int(-max), rt.Int(max-1))
					var ww bytes.Buffer
					err = binary.Write(&ww, byteOrder, uint64(m))
					if err != nil {
						return nil, err
					}
					if byteOrder == binary.LittleEndian {
						w.Write(ww.Bytes()[:n])
					} else {
						w.Write(ww.Bytes()[8-n:])
					}
				}
			}
		case 'I':
			if !optOptionArg() {
				return nil, err
			}
			m, ok := nextIntValue()
			if n == 0 {
				ok = ok && align(4) && checkBounds(m, 0, math.MaxInt32) && write(uint32(m))
			} else {
				ok = ok && align(n)
				if n >= 8 {
					ok = checkBounds(m, 0, math.MaxInt64)
					if byteOrder == binary.BigEndian {
						for j := n; j > 8; j-- {
							w.WriteByte(0)
						}
					}

					ok = ok && write(uint64(n))

					if byteOrder == binary.LittleEndian {
						for j := n; j > 8; j-- {
							w.WriteByte(0)
						}
					}
				} else {
					max := 1 << (n << 3)
					ok = ok && checkBounds(m, 0, rt.Int(max-1))
					var ww bytes.Buffer
					err = binary.Write(&ww, byteOrder, uint64(m))
					if err != nil {
						return nil, err
					}
					if byteOrder == binary.LittleEndian {
						w.Write(ww.Bytes()[:n])
					} else {
						w.Write(ww.Bytes()[8-n:])
					}
				}
			}
		case 'f':
			f, ok := nextFloatValue()
			if ok && (f < -math.MaxFloat32 || f > math.MaxFloat32) {
				return nil, errOutOfBounds
			}
			ok = ok && write(float32(f))
		case 'd', 'n':
			f, ok := nextFloatValue()
			ok = ok && write(float64(f))
		case 'c':
			if !optionArg() {
				return nil, errMissingSize
			}
			s, ok := nextStringValue()
			if !ok {
				return nil, err
			}
			if len(s) > int(n) {
				return nil, errOutOfBounds
			}
			w.Write([]byte(s))
			for j := len(s); j < int(n); j++ {
				w.WriteByte(0)
			}
		case 'z':
			s, ok := nextStringValue()
			if !ok {
				return nil, err
			}
			w.Write([]byte(s))
			w.WriteByte(0)
		case 's':
			// TODO OMG this is dreadful
		case 'x':
			w.WriteByte(0)
		case 'X':
			alignOnly = true
		case ' ':
			// Ignored
		default:
			return nil, errBadFormatString
		}
		if err != nil {
			return nil, err
		}
	}
	if alignOnly {
		return nil, errExpectedOption
	}
	return w.Bytes(), nil
}

var errBadOptionArg = errors.New("arg must be between 1 and 16")
var errMissingSize = errors.New("missing string length")
var errBadType = errors.New("bad value type")          // TODO: better error
var errOutOfBounds = errors.New("Value out of bounds") // TODO: better error
var errBadFormatString = errors.New("Bad syntax in format string")
var errExpectedOption = errors.New("Expected option after 'X'")
