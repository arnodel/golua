package runtime

import "github.com/arnodel/golua/runtime/internal/luagc"

type ResourceReleaser interface {
	ReleaseResources()
}

func releaseResources(refs []luagc.Value) {
	for _, r := range refs {
		if rr, ok := r.(ResourceReleaser); ok {
			rr.ReleaseResources()
		}
	}
}

type UserDataResourceReleaser interface {
	ReleaseResources(d *UserData)
}

// A UserData is a Go value of any type wrapped to be used as a Lua value.  It
// has a metatable which may allow Lua code to interact with it.
type UserData struct {
	value interface{}
	meta  *Table
}

var _ ResourceReleaser = (*UserData)(nil)

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

var _ luagc.Value = (*UserData)(nil)

func (d *UserData) Key() luagc.Key {
	return d.value
}

func (d *UserData) Clone() luagc.Value {
	clone := new(UserData)
	*clone = *d
	return clone
}

// HasFinalizer returns true if the user data has finalizing code (either via
// __gc metamethod or the value needs prefinalization).
func (d *UserData) MarkFlags() (flags luagc.MarkFlags) {
	_, ok := d.value.(UserDataResourceReleaser)
	if ok {
		flags |= luagc.Release
	}
	if !RawGet(d.meta, MetaFieldGcValue).IsNil() {
		flags |= luagc.Finalize
	}
	return flags
}

// Prefinalizer runs the value's prefinalize
func (d *UserData) ReleaseResources() {
	if pf, ok := d.value.(UserDataResourceReleaser); ok {
		pf.ReleaseResources(d)
	}
}

// NewUserDataValue creates a Value containing the user data with the given Go
// value and metatable.  It also registers a GC finalizer if the metadata has a
// __gc field.
func (r *Runtime) NewUserDataValue(iface interface{}, meta *Table) Value {
	udata := NewUserData(iface, meta)
	r.addFinalizer(udata, udata.MarkFlags())
	return UserDataValue(udata)
}

//
// LightUserData
//

// A LightUserData is some Go value of unspecified type wrapped to be used as a
// lua Value.
type LightUserData struct {
	Data interface{}
}
