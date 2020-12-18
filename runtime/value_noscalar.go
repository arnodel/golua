// +build noscalar
//
// This implementation of Value disables special casing of ints, floats and
// bools.  It causes more memory allocations.

package runtime

type Value struct {
	iface interface{}
}

func AsValue(i interface{}) Value {
	return Value{iface: i}
}

func (v Value) Interface() interface{} {
	return v.iface
}

func IntValue(n int64) Value {
	return Value{iface: n}
}

func FloatValue(f float64) Value {
	return Value{iface: f}
}

func BoolValue(b bool) Value {
	return Value{iface: b}
}

func StringValue(s string) Value {
	return Value{iface: s}
}

func TableValue(t *Table) Value {
	return Value{iface: t}
}

func FunctionValue(c Callable) Value {
	return Value{iface: c}
}

func ContValue(c Cont) Value {
	return Value{iface: c}
}

func ArrayValue(a []Value) Value {
	return Value{iface: a}
}

func CodeValue(c *Code) Value {
	return Value{iface: c}
}

func ThreadValue(t *Thread) Value {
	return Value{iface: t}
}

func LightUserDataValue(d LightUserData) Value {
	return Value{iface: d}
}

func UserDataValue(u *UserData) Value {
	return Value{iface: u}
}

var NilValue = Value{}

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

func (v Value) NumberType() ValueType {
	switch v.iface.(type) {
	case int64:
		return IntType
	case float64:
		return FloatType
	}
	return UnknownType
}

func (v Value) AsInt() int64 {
	return v.iface.(int64)
}

func (v Value) AsFloat() float64 {
	return v.iface.(float64)
}

func (v Value) AsBool() bool {
	return v.iface.(bool)
}

func (v Value) AsString() string {
	return v.iface.(string)
}

func (v Value) AsTable() *Table {
	return v.iface.(*Table)
}

func (v Value) AsCont() Cont {
	return v.iface.(Cont)
}

func (v Value) AsArray() []Value {
	return v.iface.([]Value)
}

func (v Value) AsClosure() *Closure {
	return v.iface.(*Closure)
}

func (v Value) AsCode() *Code {
	return v.iface.(*Code)
}

func (v Value) AsUserData() *UserData {
	return v.iface.(*UserData)
}

func (v Value) AsFunction() Callable {
	return v.iface.(Callable)
}

func (v Value) TryInt() (n int64, ok bool) {
	n, ok = v.iface.(int64)
	return
}

func (v Value) TryFloat() (n float64, ok bool) {
	n, ok = v.iface.(float64)
	return
}

func (v Value) TryString() (s string, ok bool) {
	s, ok = v.iface.(string)
	return
}

func (v Value) TryCallable() (c Callable, ok bool) {
	c, ok = v.iface.(Callable)
	return
}

func (v Value) TryClosure() (c *Closure, ok bool) {
	c, ok = v.iface.(*Closure)
	return
}

func (v Value) TryThread() (t *Thread, ok bool) {
	t, ok = v.iface.(*Thread)
	return
}

func (v Value) TryTable() (t *Table, ok bool) {
	t, ok = v.iface.(*Table)
	return
}

func (v Value) TryUserData() (u *UserData, ok bool) {
	u, ok = v.iface.(*UserData)
	return
}

func (v Value) TryBool() (b bool, ok bool) {
	b, ok = v.iface.(bool)
	return
}

func (v Value) TryCont() (c Cont, ok bool) {
	c, ok = v.iface.(Cont)
	return
}

func (v Value) TryCode() (c *Code, ok bool) {
	c, ok = v.iface.(*Code)
	return
}

func (v Value) IsNil() bool {
	return v.iface == nil
}
