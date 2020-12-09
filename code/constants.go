package code

import (
	"fmt"
	"strconv"
)

// A Constant is a literal that can be loaded into a register (that include code
// chunks).
type Constant interface {
	ShortString() string
}

// Code is a constant representing a chunk of code.  It doesn't contain any
// actual opcodes, but refers to a range in the code unit it belongs to.
type Code struct {
	Name                   string   // Name of the function (if it has one)
	StartOffset, EndOffset uint     // Where to find the opcode in the code Unit this belongs to
	UpvalueCount           int16    // Number of upvalues
	CellCount              int16    // Number of cell registers needed to run the code
	RegCount               int16    // Number of registers needed to run the coee
	UpNames                []string // Names of the upvalues
}

var _ Constant = Code{}
var _ spanGetter = Code{}

// ShortString returns a short string describing this constant (e.g. for disassembly)
func (c Code) ShortString() string {
	return fmt.Sprintf("function %s [%d - %d]", c.nameString(), c.StartOffset, c.EndOffset-1)
}

// GetSpan returns the name of the function this code is compiled from, and the
// range of instructions in the code Unit this code belongs to.
func (c Code) GetSpan() (name string, start, end int) {
	name = c.nameString()
	start = int(c.StartOffset)
	end = int(c.EndOffset - 1)
	return
}

func (c Code) nameString() string {
	if c.Name == "" {
		return "<anon>"
	}
	return c.Name
}

// A Float is a floating point literal.
type Float float64

var _ Constant = Float(0)

// ShortString returns a short string describing this constant (e.g. for disassembly)
func (f Float) ShortString() string {
	return strconv.FormatFloat(float64(f), 'g', -1, 64)
}

// An Int is an integer literal.
type Int int64

var _ Constant = Int(0)

// ShortString returns a short string describing this constant (e.g. for disassembly)
func (i Int) ShortString() string {
	return strconv.FormatInt(int64(i), 10)
}

// A Bool is a boolean literal.
type Bool bool

var _ Constant = Bool(false)

// ShortString returns a short string describing this constant (e.g. for disassembly)
func (b Bool) ShortString() string {
	return strconv.FormatBool(bool(b))
}

// A String is a string literal.
type String string

var _ Constant = String("")

// ShortString returns a short string describing this constant (e.g. for disassembly)
func (s String) ShortString() string {
	return strconv.Quote(string(s))
}

// NilType is the type of the nil literal.
type NilType struct{}

var _ Constant = NilType(struct{}{})

// ShortString returns a short string describing this constant (e.g. for disassembly)
func (n NilType) ShortString() string {
	return "nil"
}
