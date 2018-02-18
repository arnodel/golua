package ir

type Constant interface{}

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

type Code struct {
	Instructions []Instruction
	Constants    []Constant
	RegCount     int
	UpvalueCount int
	Labels       map[Label]int
}

type Float float64

type Int int64

type Bool bool

type String string

type NilType struct{}
