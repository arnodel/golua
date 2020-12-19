package runtime

import (
	"encoding/binary"
	"io"

	"github.com/arnodel/golua/code"
)

// WriteConst serializes a const value to the writer w.
func WriteConst(w io.Writer, c Value) (err error) {
	switch c.Type() {
	case IntType:
		err = bwrite(w, IntType, c.AsInt())
	case FloatType:
		err = bwrite(w, FloatType, c.AsFloat())
	case BoolType:
		err = bwrite(w, BoolType, c.AsBool())
	case StringType:
		err = bwrite(w, StringType)
		if err == nil {
			err = swrite(w, c.AsString())
		}
	case NilType:
		err = bwrite(w, NilType)
	case CodeType:
		err = writeCode(w, c.AsCode())
	default:
		panic("invalid value type")
	}
	return
}

// LoadConst reads from r to deserialize a const value.
func LoadConst(r io.Reader) (v Value, err error) {
	var tp ValueType
	err = bread(r, &tp)
	if err != nil {
		return
	}
	switch tp {
	case IntType:
		var x int64
		err = bread(r, &x)
		if err == nil {
			v = IntValue(x)
		}
	case FloatType:
		var x float64
		err = bread(r, &x)
		if err == nil {
			v = FloatValue(x)
		}
	case StringType:
		var s string
		err = sread(r, &s)
		if err == nil {
			v = StringValue(s)
		}
	case BoolType:
		var x bool
		err = bread(r, x)
		if err == nil {
			v = BoolValue(x)
		}
	case NilType:
		return NilValue, nil
	case CodeType:
		x := new(Code)
		err = loadCode(r, x)
		if err == nil {
			v = CodeValue(x)
		}
	default:
		panic("invalid value type")
	}
	return
}

func writeCode(w io.Writer, c *Code) (err error) {
	bwrite(w, CodeType)
	swrite(w, c.source)
	swrite(w, c.name)
	bwrite(w, int64(len(c.code)))
	for _, opcode := range c.code {
		bwrite(w, opcode)
	}
	bwrite(w, int64(len(c.lines)))
	bwrite(w, c.lines)
	bwrite(w, int64(len(c.consts)))
	for _, k := range c.consts {
		WriteConst(w, k)
	}
	bwrite(w, c.UpvalueCount)
	bwrite(w, c.RegCount)
	bwrite(w, c.CellCount)
	bwrite(w, int64(len(c.UpNames)))
	for _, n := range c.UpNames {
		swrite(w, n)
	}
	return
}

func loadCode(r io.Reader, c *Code) (err error) {
	sread(r, &c.source)
	sread(r, &c.name)
	var sz int64
	bread(r, &sz)
	c.code = make([]code.Opcode, sz)
	for i := range c.code {
		bread(r, &c.code[i])
	}
	bread(r, &sz)
	c.lines = make([]int32, sz)
	bread(r, c.lines)
	bread(r, &sz)
	c.consts = make([]Value, sz)
	for i := range c.consts {
		c.consts[i], err = LoadConst(r)
		if err != nil {
			return
		}
	}
	bread(r, &c.UpvalueCount)
	bread(r, &c.RegCount)
	bread(r, &c.CellCount)
	bread(r, &sz)
	c.UpNames = make([]string, sz)
	for i := range c.UpNames {
		sread(r, &c.UpNames[i])
	}
	return
}

//
//  Helper functions
//

func swrite(w io.Writer, s string) (err error) {
	err = bwrite(w, int64(len(s)))
	if err == nil {
		err = bwrite(w, []byte(s))
	}
	return
}

func sread(r io.Reader, s *string) (err error) {
	var sl int64
	err = bread(r, &sl)
	if err == nil {
		b := make([]byte, sl)
		err = bread(r, b)
		if err == nil {
			*s = string(b)
		}
	}
	return
}

func bwrite(w io.Writer, xs ...interface{}) (err error) {
	for _, x := range xs {
		err = binary.Write(w, binary.LittleEndian, x)
		if err != nil {
			break
		}
	}
	return
}

func bread(r io.Reader, xs ...interface{}) (err error) {
	for _, x := range xs {
		err = binary.Read(r, binary.LittleEndian, x)
		if err != nil {
			break
		}
	}
	return
}
