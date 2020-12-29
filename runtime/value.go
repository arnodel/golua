//+build !noscalar

package runtime

import (
	"unsafe"
)

// A Value is a runtime value.
type Value struct {
	scalar uint64
	iface  interface{}
}

var (
	dummyInt64   interface{} = int64(0)
	dummyFloat64 interface{} = float64(0)
	dummyBool    interface{} = false
)

// AsValue returns a Value for the passed interface.  Use carefully, as it may
// trigger allocations (e.g. AsValue(1) will allocate as the interface holding 1
// will put 1 on the heap).
func AsValue(i interface{}) Value {
	if i == nil {
		return NilValue
	}
	switch x := i.(type) {
	case int64:
		return IntValue(x)
	case int:
		return IntValue(int64(x))
	case float64:
		return FloatValue(x)
	case float32:
		return FloatValue(float64(x))
	case bool:
		return BoolValue(x)
	case Value:
		return x
	default:
		return Value{iface: i}
	}
}

// Interface turns the Value into an interface.  As AsValue, this can trigger
// allocations so use with caution.
func (v Value) Interface() interface{} {
	if v.iface == nil {
		return nil
	}
	switch v.iface.(type) {
	case int64:
		return v.AsInt()
	case float64:
		return v.AsFloat()
	case bool:
		return v.AsBool()
	default:
		return v.iface
	}
}

// IntValue returns a Value holding the given arg.
func IntValue(n int64) Value {
	return Value{uint64(n), dummyInt64}
}

// FloatValue returns a Value holding the given arg.
func FloatValue(f float64) Value {
	return Value{*(*uint64)(unsafe.Pointer(&f)), dummyFloat64}
}

// BoolValue returns a Value holding the given arg.
func BoolValue(b bool) Value {
	var s uint64
	if b {
		s = 1
	}
	return Value{s, dummyBool}
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
	return int64(v.scalar)
}

// AsFloat returns v as a float64 (or panics).
func (v Value) AsFloat() float64 {
	return *(*float64)(unsafe.Pointer(&v.scalar))
}

// AsBool returns v as a bool (or panics).
func (v Value) AsBool() bool {
	return v.scalar != 0
}

// AsString returns v as a string (or panics).
func (v Value) AsString() string {
	return v.iface.(string)
}

// AsTable returns v as a *Table (or panics).
func (v Value) AsTable() *Table {
	return v.iface.(*Table)
}

// AsCont returns v as a Cont, by looking at the concrete type (or panics).  It
// is an optimisation as type assertion in Go seems to have a significant cost.
func (v Value) AsCont() Cont {
	switch cont := v.iface.(type) {
	case *GoCont:
		return cont
	case *LuaCont:
		return cont
	case *Termination:
		return cont
	default:
		panic("value is not a continuation")
	}
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

// AsFunction returns v as a Callable if possible by looking at the
// possible concrete types (ok is false otherwise).  It is an optimisation as
// type assertion in Go seems to have a significant cost.
func (v Value) AsFunction() Callable {
	switch c := v.iface.(type) {
	case *Closure:
		return c
	case *GoFunction:
		return c
	default:
		panic("value is not a Callable")
	}
}

// TryInt converts v to type int64 if possible (ok is false otherwise).
func (v Value) TryInt() (n int64, ok bool) {
	_, ok = v.iface.(int64)
	if ok {
		n = v.AsInt()
	}
	return
}

// TryFloat converts v to type float64 if possible (ok is false otherwise).
func (v Value) TryFloat() (n float64, ok bool) {
	_, ok = v.iface.(float64)
	if ok {
		n = v.AsFloat()
	}
	return
}

// TryString converts v to type string if possible (ok is false otherwise).
func (v Value) TryString() (s string, ok bool) {
	s, ok = v.iface.(string)
	return
}

// TryCallable converts v to type Callable if possible by looking at the
// possible concrete types (ok is false otherwise).  It is an optimisation as
// type assertion in Go seems to have a significant cost.
func (v Value) TryCallable() (c Callable, ok bool) {
	switch c := v.iface.(type) {
	case *Closure:
		return c, true
	case *GoFunction:
		return c, true
	default:
		return nil, false
	}
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
	_, ok = v.iface.(bool)
	if ok {
		b = v.scalar != 0
	}
	return
}

// TryCont returns v as a Cont, by looking at the concrete type (ok is false if
// it doesn't implement the Cont interface).  It is an optimisation as type
// assertion in Go seems to have a significant cost.
func (v Value) TryCont() (c Cont, ok bool) {
	switch cont := v.iface.(type) {
	case *GoCont:
		return cont, true
	case *LuaCont:
		return cont, true
	case *Termination:
		return cont, true
	default:
		return nil, false
	}
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
