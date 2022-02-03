package runtime

// A UserData is a Go value of any type wrapped to be used as a Lua value.  It
// has a metatable which may allow Lua code to interact with it.
type UserData struct {
	value interface{}
	meta  *Table
}

// NewUserData returns a new UserData pointer for the value v, giving it meta as
// a metatable.  This does not register a GC finalizer because access to a
// runtime is needed for that.
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

// NewUserDataValue creates a Value containing the user data with the given Go
// value and metatable.  It also registers a GC finalizer if the metadata has a
// __gc field.
func (r *Runtime) NewUserDataValue(iface interface{}, meta *Table) Value {
	v := UserDataValue(NewUserData(iface, meta))
	if !RawGet(meta, MetaFieldGcValue).IsNil() {
		r.addFinalizer(v)
	}
	return v
}
