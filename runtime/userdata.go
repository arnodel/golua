package runtime

type UserData struct {
	value interface{}
	meta  *Table
}

func NewUserData(v interface{}, meta *Table) *UserData {
	return &UserData{value: v, meta: meta}
}

func (d *UserData) Value() interface{} {
	return d.value
}

func (d *UserData) Metatable() *Table {
	return d.meta
}

func (d *UserData) SetMetatable(m *Table) {
	d.meta = m
}
