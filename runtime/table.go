package runtime

const (
	borderOK int64 = iota
	borderCheckUp
	borderCheckDown
)

type Table struct {
	content     map[Value]Value
	meta        *Table
	border      Int
	borderState int64
}

func NewTable() *Table {
	return &Table{content: make(map[Value]Value)}
}

func (t *Table) Metatable() *Table {
	return t.meta
}

func (t *Table) SetMetatable(m *Table) {
	t.meta = m
}

func (t *Table) Get(k Value) Value {
	if x, ok := k.(Float); ok {
		if n, ok := floatToInt(x); ok {
			k = n
		}
	}
	v := t.content[k]
	// fmt.Printf("GET %#v .... %#v ::::: %#v\n", t, k, v)
	return v
}

func (t *Table) setInt(n Int, v Value) {
	if n > t.border {
		t.border = n
		t.borderState = borderCheckUp
	} else if v == nil && t.border > 0 && n == t.border {
		t.border--
		t.borderState = borderCheckDown
	}
	t.content[n] = v
}

func (t *Table) Set(k Value, v Value) {
	switch x := k.(type) {
	case Int:
		t.setInt(x, v)
		return
	case Float:
		if n, ok := floatToInt(x); ok {
			t.setInt(n, v)
			return
		}
	}
	t.content[k] = v
}

func (t *Table) Len() Int {
	switch t.borderState {
	case borderCheckDown:
		for t.border > 0 && t.content[t.border] == nil {
			t.border--
		}
		t.borderState = borderOK
	case borderCheckUp:
		for t.content[t.border+1] != nil {
			t.border++
		}
		t.borderState = borderOK
	}
	return t.border
}
