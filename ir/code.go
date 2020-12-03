package ir

type Constant interface {
	ProcessConstant(p ConstantProcessor)
}

type ConstantProcessor interface {
	ProcessFloat(Float)
	ProcessInt(Int)
	ProcessBool(Bool)
	ProcessString(String)
	ProcessNil(NilType)
	ProcessCode(Code)
}

type ConstantPool struct {
	constants []Constant
}

func (c *ConstantPool) GetConstant(k Constant) uint {
	for i, kk := range c.constants {
		if k == kk {
			return uint(i)
		}
	}
	c.constants = append(c.constants, k)
	return uint(len(c.constants) - 1)
}

func (c *ConstantPool) Constants() []Constant {
	return c.constants
}

type Float float64

func (f Float) ProcessConstant(p ConstantProcessor) {
	p.ProcessFloat(f)
}

type Int int64

func (n Int) ProcessConstant(p ConstantProcessor) {
	p.ProcessInt(n)
}

type Bool bool

func (b Bool) ProcessConstant(p ConstantProcessor) {
	p.ProcessBool(b)
}

type String string

func (s String) ProcessConstant(p ConstantProcessor) {
	p.ProcessString(s)
}

type NilType struct{}

func (n NilType) ProcessConstant(p ConstantProcessor) {
	p.ProcessNil(n)
}

type Code struct {
	Instructions []Instruction
	Lines        []int
	Constants    []Constant
	UpvalueDests []Register
	Registers    []RegData
	UpNames      []string
	LabelPos     map[int][]Label
	Name         string
}

func (c Code) ProcessConstant(p ConstantProcessor) {
	p.ProcessCode(c)
}
