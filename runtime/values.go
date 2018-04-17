package runtime

import "github.com/arnodel/golua/code"

type Bool bool
type Int int64
type Float float64
type NilType struct{}
type String string

type Callable interface {
	Call(*Thread, []Value, []Value) error
}

type ToStringable interface {
	ToString() string
}

type Table struct {
	content map[Value]Value
	meta    *Table
}

func NewTable() *Table {
	return &Table{content: make(map[Value]Value)}
}

func (t *Table) Metatable() *Table {
	return t.meta
}

type Metatabler interface {
	Metatable() *Table
}

type Closure struct {
	*code.Code
	upvalues     []Value
	upvalueIndex int
}

func NewClosure(c *code.Code) *Closure {
	return &Closure{
		Code:     c,
		upvalues: make([]Value, c.UpvalueCount),
	}
}

func (c *Closure) AddUpvalue(v Value) {
	c.upvalues[c.upvalueIndex] = v
	c.upvalueIndex++
}
