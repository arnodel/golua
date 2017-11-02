package ir

type Constant interface{}

type Code struct {
	Instructions []Instruction
	Constants    []Constant
	RegCount     int
	upValueCount int
}

type Float float64

type Int int64

type Bool bool

type String string

type NilType struct{}
