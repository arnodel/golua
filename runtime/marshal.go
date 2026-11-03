package runtime

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"

	"github.com/arnodel/golua/code"
)

var marshalPrefix = []byte{6, 0, 4}
var ErrInvalidMarshalPrefix = errors.New("Invalid marshal prefix")

// HasMarshalPrefix returns true if the byte slice passed starts witht the magic
// prefix for Lua marshalled values.
func HasMarshalPrefix(bs []byte) bool {
	return len(bs) >= len(marshalPrefix) && bytes.Equal(marshalPrefix, bs[:len(marshalPrefix)])
}

// MarshalConst serializes a const value to the writer w.
func MarshalConst(w io.Writer, c Value, budget uint64) (used uint64, err error) {
	defer func() {
		if r := recover(); r == budgetConsumed {
			used = budget
		} else if r != nil {
			panic(r)
		}
	}()
	if _, err := w.Write(marshalPrefix); err != nil {
		return 0, err
	}
	bw := bwriter{w: w, budget: budget}
	bw.writeConst(c)
	return budget - bw.budget, bw.err
}

// UnmarshalConst reads from r to deserialize a const value.
//
// The input may come from anywhere, so every length in it is checked before
// it is used, and what it describes is read as it comes: the input can make
// UnmarshalConst allocate no more than the input holds.
func UnmarshalConst(r io.Reader, budget uint64) (v Value, used uint64, err error) {
	defer func() {
		if r := recover(); r == budgetConsumed {
			used = budget
		} else if r != nil {
			panic(r)
		}
	}()
	pfx := make([]byte, len(marshalPrefix))
	_, err = r.Read(pfx)
	if !bytes.Equal(pfx, marshalPrefix) {
		err = ErrInvalidMarshalPrefix
	}
	if err != nil {
		return
	}
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
		if r.err == nil {
			r.err = x.verify()
		}
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
	r.read(0+0, &c.source, &c.name)
	c.code = readSlice[code.Opcode](r, r.readLen(4), 4)
	c.lines = readSlice[int32](r, r.readLen(4), 4)
	for n := r.readLen(1); len(c.consts) < n && r.err == nil; {
		c.consts = append(c.consts, r.readConst())
	}
	r.read(2+2+2, &c.UpvalueCount, &c.RegCount, &c.CellCount)
	for n := r.readLen(8); len(c.UpNames) < n && r.err == nil; {
		c.UpNames = append(c.UpNames, r.readString())
	}
}

// readLen reads the number of items of what comes next in the input, each
// taking at least itemSize bytes of it, and charges the budget for them.
func (r *breader) readLen(itemSize uint64) int {
	var n int64
	r.read(8, &n)
	if r.err != nil {
		return 0
	}
	if n < 0 || uint64(n) > math.MaxInt {
		r.err = errInvalidLength
		return 0
	}
	cost := uint64(n) * itemSize
	if cost/itemSize != uint64(n) {
		cost = math.MaxUint64
	}
	r.consumeBudget(cost)
	return int(n)
}

// The most items readSlice allocates for before reading them.
const readChunk = 4096

// readSlice reads n items of the given size.  n comes from the input, so
// the slice grows as items are read, in chunks: a length larger than the
// input fails at its end, having allocated about what it held.
func readSlice[T any](r *breader, n int, size uint64) []T {
	if r.err != nil {
		return nil
	}
	xs := make([]T, 0, min(n, readChunk))
	for len(xs) < n {
		chunk := make([]T, min(n-len(xs), readChunk))
		if r.err = binary.Read(r.r, binary.LittleEndian, chunk); r.err != nil {
			return nil
		}
		xs = append(xs, chunk...)
	}
	return xs
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
	b := readSlice[byte](r, r.readLen(1), 1)
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
var errInvalidLength = errors.New("Invalid length")
