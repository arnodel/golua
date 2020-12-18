// +build noscalar
//
// This implementation of Value disables special casing of ints, floats and
// bools.  It causes more memory allocations.

package runtime

// A Value is a runtime value.
type Value struct {
	iface interface{}
}

// AsValue returns a Value for the passed interface.
func AsValue(i interface{}) Value {
	return Value{iface: i}
}

// Interface turns the Value into an interface.
func (v Value) Interface() interface{} {
	return v.iface
}

// IntValue returns a Value holding the given arg.
func IntValue(n int64) Value {
	return Value{iface: n}
}

// FloatValue returns a Value holding the given arg.
func FloatValue(f float64) Value {
	return Value{iface: f}
}

// BoolValue returns a Value holding the given arg.
func BoolValue(b bool) Value {
	return Value{iface: b}
}

// StringValue returns a Value holding the given arg.
func StringValue(s string) Value {
	return Value{iface: s}
}

// TableValue returns a Value holding the given arg.
func TableValue(t *Table) Value {
	return Value{iface: t}
}

// FunctionValue returns a Value holding the given arg.
func FunctionValue(c Callable) Value {
	return Value{iface: c}
}

// ContValue returns a Value holding the given arg.
func ContValue(c Cont) Value {
	return Value{iface: c}
}

// ArrayValue returns a Value holding the given arg.
func ArrayValue(a []Value) Value {
	return Value{iface: a}
}

// CodeValue returns a Value holding the given arg.
func CodeValue(c *Code) Value {
	return Value{iface: c}
}

// ThreadValue returns a Value holding the given arg.
func ThreadValue(t *Thread) Value {
	return Value{iface: t}
}

// LightUserDataValue returns a Value holding the given arg.
func LightUserDataValue(d LightUserData) Value {
	return Value{iface: d}
}

// UserDataValue returns a Value holding the given arg.
func UserDataValue(u *UserData) Value {
	return Value{iface: u}
}

// NilValue is a value holding Nil.
var NilValue = Value{}

// Type returns the ValueType of v.
func (v Value) Type() ValueType {
	if v.iface == nil {
		return NilType
	}
	switch v.iface.(type) {
	case int64:
		return IntType
	case float64:
		return FloatType
	case bool:
		return BoolType
	case string:
		return StringType
	case *Table:
		return TableType
	case *Code:
		return CodeType
	case Callable:
		return FunctionType
	case *Thread:
		return ThreadType
	case *UserData:
		return UserDataType
	default:
		return UnknownType
	}
}

// NumberType return the ValueType of v if it is a number, otherwise
// UnknownType.
func (v Value) NumberType() ValueType {
	switch v.iface.(type) {
	case int64:
		return IntType
	case float64:
		return FloatType
	}
	return UnknownType
}

// AsInt returns v as a int64 (or panics).
func (v Value) AsInt() int64 {
	return v.iface.(int64)
}

// AsFloat returns v as a float64 (or panics).
func (v Value) AsFloat() float64 {
	return v.iface.(float64)
}

// AsBool returns v as a bool (or panics).
func (v Value) AsBool() bool {
	return v.iface.(bool)
}

// AsString returns v as a string (or panics).
func (v Value) AsString() string {
	return v.iface.(string)
}

// AsTable returns v as a *Table (or panics).
func (v Value) AsTable() *Table {
	return v.iface.(*Table)
}

// AsCont returns v as a Cont (or panics).
func (v Value) AsCont() Cont {
	return v.iface.(Cont)
}

// AsArray returns v as a [] (or panics).
func (v Value) AsArray() []Value {
	return v.iface.([]Value)
}

// AsClosure returns v as a *Closure (or panics).
func (v Value) AsClosure() *Closure {
	return v.iface.(*Closure)
}

// AsCode returns v as a *Code (or panics).
func (v Value) AsCode() *Code {
	return v.iface.(*Code)
}

// AsUserData returns v as a *UserData (or panics).
func (v Value) AsUserData() *UserData {
	return v.iface.(*UserData)
}

// AsFunction returns v as a Callable (or panics).
func (v Value) AsFunction() Callable {
	return v.iface.(Callable)
}

// TryInt converts v to type int64 if possible (ok is false otherwise).
func (v Value) TryInt() (n int64, ok bool) {
	n, ok = v.iface.(int64)
	return
}

// TryFloat converts v to type float64 if possible (ok is false otherwise).
func (v Value) TryFloat() (n float64, ok bool) {
	n, ok = v.iface.(float64)
	return
}

// TryString converts v to type string if possible (ok is false otherwise).
func (v Value) TryString() (s string, ok bool) {
	s, ok = v.iface.(string)
	return
}

// TryCallable converts v to type Callable if possible (ok is false otherwise).
func (v Value) TryCallable() (c Callable, ok bool) {
	c, ok = v.iface.(Callable)
	return
}

// TryClosure converts v to type *Closure if possible (ok is false otherwise).
func (v Value) TryClosure() (c *Closure, ok bool) {
	c, ok = v.iface.(*Closure)
	return
}

// TryThread converts v to type *Thread if possible (ok is false otherwise).
func (v Value) TryThread() (t *Thread, ok bool) {
	t, ok = v.iface.(*Thread)
	return
}

// TryTable converts v to type *Table if possible (ok is false otherwise).
func (v Value) TryTable() (t *Table, ok bool) {
	t, ok = v.iface.(*Table)
	return
}

// TryUserData converts v to type *UserData if possible (ok is false otherwise).
func (v Value) TryUserData() (u *UserData, ok bool) {
	u, ok = v.iface.(*UserData)
	return
}

// TryBool converts v to type bool if possible (ok is false otherwise).
func (v Value) TryBool() (b bool, ok bool) {
	b, ok = v.iface.(bool)
	return
}

// TryCont converts v to type Cont if possible (ok is false otherwise).
func (v Value) TryCont() (c Cont, ok bool) {
	c, ok = v.iface.(Cont)
	return
}

// TryCode converts v to type *Code if possible (ok is false otherwise).
func (v Value) TryCode() (c *Code, ok bool) {
	c, ok = v.iface.(*Code)
	return
}

// IsNil returns true if v is nil.
func (v Value) IsNil() bool {
	return v.iface == nil
}
