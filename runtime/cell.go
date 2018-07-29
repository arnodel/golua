package runtime

type Cell struct {
	ref *Value
}

func NewCell(v Value) Cell {
	return Cell{&v}
}

func (c Cell) Get() Value {
	return *c.ref
}

func (c Cell) Set(v Value) {
	*c.ref = v
}

func asCell(v Value) Cell {
	c, ok := v.(Cell)
	if ok {
		return c
	}
	return Cell{&v}
}

func asValue(v Value) Value {
	c, ok := v.(Cell)
	if ok {
		return *c.ref
	}
	return v
}
