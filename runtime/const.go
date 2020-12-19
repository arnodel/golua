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
		_, err = w.Write([]byte{constTypeInt})
		if err == nil {
			err = bwrite(w, c.AsInt())
		}
	case FloatType:
		_, err := w.Write([]byte{constTypeFloat})
		if err == nil {
			err = bwrite(w, c.AsFloat())
		}
	case BoolType:
		_, err = w.Write([]byte{constTypeBool})
		if err == nil {
			err = bwrite(w, c.AsBool())
		}
	case StringType:
		_, err = w.Write([]byte{constTypeString})
		if err == nil {
			err = swrite(w, c.AsString())
		}
	case NilType:
		_, err = w.Write([]byte{constTypeNil})
	case CodeType:
		err = c.AsCode().write(w)
	default:
		panic("invalid value type")
	}
	return
}

// LoadConst reads from r to deserialize a const value.
func LoadConst(r io.Reader) (Value, error) {
	var tp = make([]byte, 1)
	_, err := r.Read(tp)
	if err != nil {
		return Value{}, err
	}
	switch tp[0] {
	case constTypeInt:
		var x int64
		if err := bread(r, &x); err != nil {
			return Value{}, err
		}
		return IntValue(x), nil
	case constTypeFloat:
		var x float64
		if err := bread(r, &x); err != nil {
			return Value{}, err
		}
		return FloatValue(x), nil
	case constTypeString:
		var l uint64
		if err := bread(r, &l); err != nil {
			return Value{}, err
		}
		var b = make([]byte, l)
		if _, err := r.Read(b); err != nil {
			return Value{}, err
		}
		return StringValue(string(b)), nil
	case constTypeBool:
		var x bool
		if err := bread(r, &x); err != nil {
			return Value{}, err
		}
		return BoolValue(x), nil
	case constTypeNil:
		return NilValue, nil
	case constTypeCode:
		x := new(Code)
		if err := x.load(r); err != nil {
			return Value{}, err
		}
		return CodeValue(x), nil
	}
	return Value{}, nil
}

func (c *Code) write(w io.Writer) (err error) {
	_, err = w.Write([]byte{constTypeCode})
	if err != nil {
		return
	}
	swrite(w, c.source)
	swrite(w, c.name)
	bwrite(w, int64(len(c.code)))
	for _, opcode := range c.code {
		bwrite(w, int32(opcode))
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

func (c *Code) load(r io.Reader) (err error) {
	sread(r, &c.source)
	sread(r, &c.name)
	var sz int64
	bread(r, &sz)
	c.code = make([]code.Opcode, sz)
	for i := range c.code {
		var op int32
		bread(r, &op)
		c.code[i] = code.Opcode(op)
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

const (
	constTypeInt = iota
	constTypeFloat
	constTypeString
	constTypeBool
	constTypeNil
	constTypeCode

	// ConstTypeMaj is bigger than any const type
	ConstTypeMaj
)

func bwrite(w io.Writer, x interface{}) (err error) {
	err = binary.Write(w, binary.LittleEndian, x)
	return
}

func bread(r io.Reader, x interface{}) error {
	return binary.Read(r, binary.LittleEndian, x)
}
