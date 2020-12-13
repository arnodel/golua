package ir

// Constant is the interface implemented by constant values (e.g. numbers,
// strings, but also code chunks).
type Constant interface {
	ProcessConstant(p ConstantProcessor)
}

// A ConstantProcessor is able to process all the different types of constant
// using the various ProcessXXX methods.
type ConstantProcessor interface {
	ProcessFloat(Float)
	ProcessInt(Int)
	ProcessBool(Bool)
	ProcessString(String)
	ProcessNil(NilType)
	ProcessCode(Code)
}

// A Float is a constant representing a literal floating point number.
type Float float64

// ProcessConstant uses the given ConstantProcessor to process the receiver.
func (f Float) ProcessConstant(p ConstantProcessor) {
	p.ProcessFloat(f)
}

// An Int is a constant representing an integer literal.
type Int int64

// ProcessConstant uses the given ConstantProcessor to process the receiver.
func (n Int) ProcessConstant(p ConstantProcessor) {
	p.ProcessInt(n)
}

// A Bool is a constant representing a boolean literal.
type Bool bool

// ProcessConstant uses the given ConstantProcessor to process the receiver.
func (b Bool) ProcessConstant(p ConstantProcessor) {
	p.ProcessBool(b)
}

// A String is a constant representing a string literal.
type String string

// ProcessConstant uses the given ConstantProcessor to process the receiver.
func (s String) ProcessConstant(p ConstantProcessor) {
	p.ProcessString(s)
}

// NilType is the type of the nil literal.
type NilType struct{}

// ProcessConstant uses the given ConstantProcessor to process the receiver.
func (n NilType) ProcessConstant(p ConstantProcessor) {
	p.ProcessNil(n)
}

// Code is the type of code literals (i.e. function definitions).
type Code struct {
	Instructions []Instruction
	Lines        []int
	Constants    []Constant
	UpvalueDests []Register
	Registers    []RegData
	UpNames      []string
	Name         string
}

// ProcessConstant uses the given ConstantProcessor to process the receiver.
func (c Code) ProcessConstant(p ConstantProcessor) {
	p.ProcessCode(c)
}

// A ConstantPool accumulates constants and associates a stable integer with
// each constant.
type ConstantPool struct {
	constants []Constant
}

// GetConstantIndex returns the index associated with a given constant.  If
// there is none, the constant is registered in the pool and the new index is
// returned.
func (c *ConstantPool) GetConstantIndex(k Constant) uint {
	for i, kk := range c.constants {
		if k == kk {
			return uint(i)
		}
	}
	c.constants = append(c.constants, k)
	return uint(len(c.constants) - 1)
}

// Constants returns the list of all constants registered with this pool.
func (c *ConstantPool) Constants() []Constant {
	return c.constants
}
