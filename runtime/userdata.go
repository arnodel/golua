package runtime

import "github.com/arnodel/golua/runtime/internal/weakref"

// A UserData is a Go value of any type wrapped to be used as a Lua value.  It
// has a metatable which may allow Lua code to interact with it.
type UserData struct {
	value interface{}
	meta  *Table
}

var _ weakref.Prefinalizer = (*UserData)(nil)

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

// HasFinalizer returns true if the user data has finalizing code (either via
// __gc metamethod or the value needs prefinalization).
func (d *UserData) HasFinalizer() bool {
	_, ok := d.value.(UserDataPrefinalizer)
	return ok || !RawGet(d.meta, MetaFieldGcValue).IsNil()
}

// Prefinalizer runs the value's prefinalize
func (d *UserData) Prefinalize() {
	if pf, ok := d.value.(UserDataPrefinalizer); ok {
		pf.Prefinalize(d)
	}
}

// NewUserDataValue creates a Value containing the user data with the given Go
// value and metatable.  It also registers a GC finalizer if the metadata has a
// __gc field.
func (r *Runtime) NewUserDataValue(iface interface{}, meta *Table) Value {
	udata := NewUserData(iface, meta)
	if udata.HasFinalizer() {
		r.addFinalizer(udata)
	}
	return UserDataValue(udata)
}

type UserDataPrefinalizer interface {
	Prefinalize(*UserData)
}

//
// LightUserData
//

// A LightUserData is some Go value of unspecified type wrapped to be used as a
// lua Value.
type LightUserData struct {
	Data interface{}
}
