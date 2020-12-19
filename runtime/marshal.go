package runtime

import (
	"encoding/binary"
	"io"

	"github.com/arnodel/golua/code"
)

// MarshalConst serializes a const value to the writer w.
func MarshalConst(w io.Writer, c Value) (err error) {
	bw := bwriter{w: w}
	return bw.writeConst(c)
}

// UnmarshalConst reads from r to deserialize a const value.
func UnmarshalConst(r io.Reader) (v Value, err error) {
	br := breader{r: r}
	err = br.readConst(&v)
	return
}

//
// bwriter: helper data struture to serialise values
//
type bwriter struct {
	w   io.Writer
	err error
}

func (w *bwriter) writeConst(c Value) (err error) {
	switch c.Type() {
	case IntType:
		w.write(IntType, c.AsInt())
	case FloatType:
		w.write(FloatType, c.AsFloat())
	case BoolType:
		w.write(BoolType, c.AsBool())
	case StringType:
		w.write(StringType, c.AsString())
	case NilType:
		w.write(NilType)
	case CodeType:
		w.writeCode(c.AsCode())
	default:
		panic("invalid value type")
	}
	return w.err
}

func (w *bwriter) writeCode(c *Code) (err error) {
	w.write(
		CodeType,
		c.source,
		c.name,
		int64(len(c.code)), c.code,
		int64(len(c.lines)), c.lines,
		int64(len(c.consts)),
	)
	for _, k := range c.consts {
		w.writeConst(k)
	}
	w.write(
		c.UpvalueCount,
		c.RegCount,
		c.CellCount,
		int64(len(c.UpNames)),
	)
	for _, n := range c.UpNames {
		w.writeString(n)
	}
	return w.err
}

func (w *bwriter) write(xs ...interface{}) (err error) {
	if w.err != nil {
		return w.err
	}
	for _, x := range xs {
		switch xx := x.(type) {
		case string:
			err = w.writeString(xx)
		default:
			err = binary.Write(w.w, binary.LittleEndian, x)
		}
		if err != nil {
			break
		}
	}
	w.err = err
	return
}

func (w *bwriter) writeString(s string) (err error) {
	if w.write(int64(len(s))) == nil {
		_, w.err = w.w.Write([]byte(s))
	}
	return w.err
}

//
// breader: helper datastructure to deserialize values
//

type breader struct {
	r   io.Reader
	err error
}

func (r *breader) readConst(v *Value) (err error) {
	var tp ValueType
	if r.read(&tp) != nil {
		return r.err
	}
	switch tp {
	case IntType:
		var x int64
		if r.read(&x) == nil {
			*v = IntValue(x)
		}
	case FloatType:
		var x float64
		if r.read(&x) == nil {
			*v = FloatValue(x)
		}
	case StringType:
		var s string
		if r.readString(&s) == nil {
			*v = StringValue(s)
		}
	case BoolType:
		var x bool
		if r.read(&x) == nil {
			*v = BoolValue(x)
		}
	case NilType:
		*v = NilValue
	case CodeType:
		x := new(Code)
		if r.readCode(x) == nil {
			*v = CodeValue(x)
		}
	default:
		panic("invalid value type")
	}
	return r.err
}

func (r *breader) readCode(c *Code) (err error) {
	var sz int64
	r.read(
		&c.source,
		&c.name,
		&sz,
	)
	c.code = make([]code.Opcode, sz)
	r.read(c.code,
		&sz,
	)
	c.lines = make([]int32, sz)
	r.read(
		c.lines,
		&sz,
	)
	c.consts = make([]Value, sz)
	for i := range c.consts {
		r.readConst(&c.consts[i])
	}
	r.read(
		&c.UpvalueCount,
		&c.RegCount,
		&c.CellCount,
		&sz,
	)
	c.UpNames = make([]string, sz)
	for i := range c.UpNames {
		r.readString(&c.UpNames[i])
	}
	return r.err
}

func (r *breader) read(xs ...interface{}) (err error) {
	if r.err != nil {
		return r.err
	}
	for _, x := range xs {
		switch xx := x.(type) {
		case *string:
			err = r.readString(xx)
		default:
			err = binary.Read(r.r, binary.LittleEndian, x)
		}
		if err != nil {
			break
		}
	}
	r.err = err
	return
}

func (r *breader) readString(s *string) (err error) {
	var sl int64
	if err = r.read(&sl); err == nil {
		b := make([]byte, sl)
		_, err = r.r.Read(b)
		if err == nil {
			*s = string(b)
		} else {
			r.err = err
		}
	}
	return
}
