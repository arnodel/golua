package stringlib

import (
	"encoding/binary"
	"errors"
	"fmt"
	"unsafe"

	"github.com/arnodel/golua/luastrings"
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

func pack(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	format, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	res, used, perr := PackValues(string(format), c.Etc(), t.LinearUnused(10))
	t.LinearRequire(10, used)
	if perr != nil {
		return nil, perr
	}
	return c.PushingNext1(t.Runtime, rt.StringValue(res)), nil
}

func unpack(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}
	var (
		format, pack string
		n            int64 = 1
		err          error
	)
	format, err = c.StringArg(0)
	if err == nil {
		pack, err = c.StringArg(1)
	}
	if err == nil && c.NArgs() >= 3 {
		n, err = c.IntArg(2)
	}
	i := luastrings.StringNormPos(pack, int(n)) - 1
	if i < 0 || i > len(pack) {
		err = errors.New("#3 out of string")
	}
	if err != nil {
		return nil, err
	}
	vals, m, used, uerr := UnpackString(string(format), string(pack), i, t.LinearUnused(10))
	t.LinearRequire(10, used)
	if uerr != nil {
		return nil, uerr
	}
	next := c.Next()
	t.Push(next, vals...)
	t.Push1(next, rt.IntValue(int64(m+1)))
	return next, nil
}

func packsize(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	format, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	size, serr := PackSize(string(format))
	if serr != nil {
		return nil, serr
	}
	return c.PushingNext1(t.Runtime, rt.IntValue(int64(size))), nil
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

var (
	errBadOptionArg           = errors.New("arg out of limits [1,16]")
	errMissingSize            = errors.New("missing size")
	errBadType                = errors.New("bad value type") // TODO: better error
	errOutOfBounds            = errors.New("overflow")       // TODO: better error
	errExpectedOption         = errors.New("invalid next option after 'X'")
	errBadAlignment           = errors.New("alignment not power of 2")
	errUnexpectedPackEnd      = errors.New("packed string too short: unexpected end")
	errDoesNotFit             = errors.New("does not fit into Lua integer")
	errStringLongerThanFormat = errors.New("string longer than format spec")
	errStringDoesNotFit       = errors.New("string does not fit")
	errVariableLength         = errors.New("variable-length format") // For packsize only
	errOverflow               = errors.New("invalid format: option size overflow")
	errStringContainsZeros    = errors.New("string contains zeros")

	errBudgetConsumed = errors.New("Packing memory budget consumed")
)

func errBadFormatString(c byte) error {
	return fmt.Errorf("invalid format option %q", c)
}
