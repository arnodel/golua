package stringlib

import (
	"encoding/binary"
	"errors"
	"fmt"
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

func pack(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	format, err := c.StringArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	res, perr := PackValues(string(format), c.Etc())
	if perr != nil {
		return nil, rt.NewErrorE(perr).AddContext(c)
	}
	return c.PushingNext(rt.String(res)), nil
}

func unpack(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err.AddContext(c)
	}
	var format, pack rt.String
	var n rt.Int = 1
	var err *rt.Error
	format, err = c.StringArg(0)
	if err == nil {
		pack, err = c.StringArg(1)
	}
	if err == nil && c.NArgs() >= 3 {
		n, err = c.IntArg(2)
	}
	i := pos(pack, n) - 1
	if i < 0 || i > len(pack) {
		err = rt.NewErrorS("#3 out of string")
	}
	if err != nil {
		return nil, err.AddContext(c)
	}
	vals, m, uerr := UnpackString(string(format), string(pack), i)
	if uerr != nil {
		return nil, rt.NewErrorE(uerr).AddContext(c)
	}
	next := c.Next()
	rt.Push(next, vals...)
	next.Push(rt.Int(m) + 1)
	return next, nil
}

func packsize(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	format, err := c.StringArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	size, serr := PackSize(string(format))
	if serr != nil {
		return nil, rt.NewErrorE(serr).AddContext(c)
	}
	return c.PushingNext(rt.Int(size)), nil
}

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

var errBadOptionArg = errors.New("arg out of limits: must be between 1 and 16")
var errMissingSize = errors.New("missing size")
var errBadType = errors.New("bad value type") // TODO: better error
var errOutOfBounds = errors.New("overflow")   // TODO: better error
var errExpectedOption = errors.New("invalid next option after 'X'")
var errBadAlignment = errors.New("alignment not power of 2")
var errUnexpectedPackEnd = errors.New("packed string too short: unexpected end")
var errDoesNotFit = errors.New("does not fit into Lua integer")
var errStringLongerThanFormat = errors.New("string longer than format spec")
var errStringDoesNotFit = errors.New("string does not fit")
var errVariableLength = errors.New("variable-length format") // For packsize only
var errOverflow = errors.New("invalid format: option size overflow")
var errStringContainsZeros = errors.New("string contains zeros")

func errBadFormatString(c byte) error {
	return fmt.Errorf("invalid format option %q", c)
}
