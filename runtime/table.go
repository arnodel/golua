package runtime

const (
	borderOK int64 = iota
	borderCheckUp
	borderCheckDown
)

type tableValue struct {
	value, next Value
}

type Table struct {
	content     map[Value]tableValue
	meta        *Table
	border      Int
	borderState int64
	first       Value
}

func NewTable() *Table {
	return &Table{content: make(map[Value]tableValue)}
}

func (t *Table) Metatable() *Table {
	return t.meta
}

func (t *Table) SetMetatable(m *Table) {
	t.meta = m
}

func (t *Table) Get(k Value) Value {
	if x, ok := k.(Float); ok {
		if n, tp := x.ToInt(); tp == IsInt {
			k = n
		}
	}
	return t.content[k].value
}

func (t *Table) setInt(n Int, v Value) {
	if n > t.border {
		t.border = n
		t.borderState = borderCheckUp
	} else if IsNil(v) && t.border > 0 && n == t.border {
		t.border--
		t.borderState = borderCheckDown
	}
	t.set(n, v)
}

func (t *Table) Set(k Value, v Value) {
	switch x := k.(type) {
	case Int:
		t.setInt(x, v)
		return
	case Float:
		if n, tp := x.ToInt(); tp == IsInt {
			t.setInt(n, v)
			return
		}
	}
	t.set(k, v)
}

func (t *Table) Len() Int {
	switch t.borderState {
	case borderCheckDown:
		for t.border > 0 && t.content[t.border].value == nil {
			t.border--
		}
		t.borderState = borderOK
	case borderCheckUp:
		for t.content[t.border+1].value != nil {
			t.border++
		}
		t.borderState = borderOK
	}
	return t.border
}

func (t *Table) set(k Value, v Value) {
	tv, ok := t.content[k]
	if IsNil(v) {
		tv.value = nil
	} else {
		tv.value = v
	}
	if !ok {
		tv.next = t.first
		t.first = k
	}
	t.content[k] = tv
}

func (t *Table) Next(k Value) (next Value, val Value) {
	if IsNil(k) {
		next = t.first
	} else {
		next = t.content[k].next
	}
	for next != nil {
		ntv := t.content[next]
		if val = ntv.value; val != nil {
			return
		}
		next = ntv.next
	}
	return
}
