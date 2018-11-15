package runtime

// A UserData is a Go value of any type wrapped to be used as a Lua value.  It
// has a metatable which may allow Lua code to interact with it.
type UserData struct {
	value interface{}
	meta  *Table
}

// NewUserData returns a new UserData pointer for the value v, giving it meta as
// a metatable.
func NewUserData(v interface{}, meta *Table) *UserData {
	return &UserData{value: v, meta: meta}
}

// Value returns the userdata's value.
func (d *UserData) Value() interface{} {
	return d.value
}

// Metatable returns the userdata's metatable.
func (d *UserData) Metatable() *Table {
	return d.meta
}

// SetMetatable sets d's metatable to m.
func (d *UserData) SetMetatable(m *Table) {
	d.meta = m
}
