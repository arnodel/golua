package runtime

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/arnodel/golua/code"
)

// MarshalConst serializes a const value to the writer w.
func MarshalConst(w io.Writer, c Value, budget uint64) (used uint64, err error) {
	defer func() {
		if r := recover(); r == budgetConsumed {
			used = budget
		}
	}()
	bw := bwriter{w: w, budget: budget}
	bw.writeConst(c)
	return budget - bw.budget, bw.err
}

// UnmarshalConst reads from r to deserialize a const value.
func UnmarshalConst(r io.Reader, budget uint64) (v Value, used uint64, err error) {
	defer func() {
		if r := recover(); r == budgetConsumed {
			used = budget
		}
	}()
	br := breader{r: r, budget: budget}
	v = br.readConst()
	return v, budget - br.budget, br.err
}

//
// bwriter: helper data struture to serialise values
//
type bwriter struct {
	w   io.Writer
	err error

	budget uint64
}

func (w *bwriter) writeConst(c Value) {
	switch c.Type() {
	case IntType:
		w.consumeBudget(1 + 8)
		w.write(IntType, c.AsInt())
	case FloatType:
		w.consumeBudget(1 + 8)
		w.write(FloatType, c.AsFloat())
	case StringType:
		w.consumeBudget(1 + 0) // w.writeString will consume the string budget
		w.write(StringType, c.AsString())
	case CodeType:
		w.writeCode(c.AsCode())
	// Booleans and nil are inlined so this shouldn't be neeeded.  Keeping
	// around in case this is reversed
	//
	//  case BoolType:
	//  w.consumeBudget(1 + 1)
	//  w.write(BoolType, c.AsBool())
	// case NilType:
	//  w.consumeBudget(1)
	//  w.write(NilType)
	default:
		w.err = errInvalidValueType
	}
}

func (w *bwriter) writeCode(c *Code) {
	w.consumeBudget(1 + 0 + 0 + 8 + 8 + 8)
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
	w.consumeBudget(2 + 2 + 2 + 8)
	w.write(
		c.UpvalueCount,
		c.RegCount,
		c.CellCount,
		int64(len(c.UpNames)),
	)
	for _, n := range c.UpNames {
		w.writeString(n)
	}
}

func (w *bwriter) write(xs ...interface{}) {
	if w.err != nil {
		return
	}
	for _, x := range xs {
		switch xx := x.(type) {
		case string:
			w.writeString(xx)
		default:
			w.err = binary.Write(w.w, binary.LittleEndian, x)
		}
		if w.err != nil {
			return
		}
	}
}

func (w *bwriter) writeString(s string) {
	w.consumeBudget(uint64(8 + len(s)))
	w.write(int64(len(s)))
	if w.err == nil {
		_, w.err = w.w.Write([]byte(s))
	}
}

func (w *bwriter) consumeBudget(amount uint64) {
	if w.budget == 0 {
		return
	}
	if w.budget < amount {
		panic(budgetConsumed)
	}
	w.budget -= amount
}

var budgetConsumed interface{} = "budget consumed"

//
// breader: helper datastructure to deserialize values
//

type breader struct {
	r   io.Reader
	err error

	budget uint64
}

func (r *breader) readConst() (v Value) {
	var tp ValueType
	r.read(1, &tp)
	if r.err != nil {
		return
	}
	switch tp {
	case IntType:
		var x int64
		r.read(8, &x)
		v = IntValue(x)
	case FloatType:
		var x float64
		r.read(8, &x)
		v = FloatValue(x)
	case StringType:
		s := r.readString()
		v = StringValue(s)
	case CodeType:
		x := new(Code)
		r.readCode(x)
		v = CodeValue(x)
	// Booleans and nil are inlined so this shouldn't be needed.  Keeping around
	// in case this is reversed.
	//
	// case BoolType:
	// 	var x bool
	// 	r.read(1, &x)
	// 	v = BoolValue(x)
	// case NilType:
	// 	v = NilValue
	default:
		r.err = errInvalidValueType
	}
	if r.err != nil {
		return NilValue
	}
	return v
}

func (r *breader) readCode(c *Code) {
	var sz int64
	r.read(
		0+0+8,
		&c.source,
		&c.name,
		&sz,
	)
	c.code = make([]code.Opcode, sz)
	r.read(
		4*uint64(sz)+8,
		c.code,
		&sz,
	)
	c.lines = make([]int32, sz)
	r.read(
		4*uint64(sz)+8,
		c.lines,
		&sz,
	)
	c.consts = make([]Value, sz)
	for i := range c.consts {
		c.consts[i] = r.readConst()
	}
	r.read(
		2+2+2+8,
		&c.UpvalueCount,
		&c.RegCount,
		&c.CellCount,
		&sz,
	)
	c.UpNames = make([]string, sz)
	for i := range c.UpNames {
		c.UpNames[i] = r.readString()
	}
}

func (r *breader) read(sz uint64, xs ...interface{}) {
	if r.err != nil {
		return
	}
	r.consumeBudget(sz)
	for _, x := range xs {
		switch xx := x.(type) {
		case *string:
			*xx = r.readString()
		default:
			r.err = binary.Read(r.r, binary.LittleEndian, x)
		}
		if r.err != nil {
			return
		}
	}
}

func (r *breader) readString() (s string) {
	if r.err != nil {
		return
	}
	var sl int64
	r.read(8, &sl)
	if r.err != nil {
		return
	}
	r.consumeBudget(uint64(sl))
	b := make([]byte, sl)
	_, r.err = r.r.Read(b)
	if r.err == nil {
		s = string(b)
	}
	return
}

func (r *breader) consumeBudget(amount uint64) {
	if r.budget == 0 {
		return
	}
	if r.budget < amount {
		panic(budgetConsumed)
	}
	r.budget -= amount
}

var errInvalidValueType = errors.New("Invalid value type")
