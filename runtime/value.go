package runtime

import (
	"unsafe"
)

type ValueType uint8

const (
	NilType ValueType = iota
	IntType
	FloatType
	BoolType
	StringType
	TableType
	FunctionType
	ThreadType
	UserDataType
	UnknownType
)

type Value struct {
	scalar uint64
	iface  interface{}
}

var (
	dummyInt64   interface{} = int64(0)
	dummyFloat64 interface{} = float64(0)
	dummyBool    interface{} = false
)

func IntValue(n int64) Value {
	return Value{uint64(n), dummyInt64}
}

func FloatValue(f float64) Value {
	return Value{*(*uint64)(unsafe.Pointer(&f)), dummyFloat64}
}

func BoolValue(b bool) Value {
	var s uint64
	if b {
		s = 1
	}
	return Value{s, dummyBool}
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

func (v Value) AsInt() int64 {
	return int64(v.scalar)
}

func (v Value) AsFloat() float64 {
	return *(*float64)(unsafe.Pointer(&v.scalar))
}

func (v Value) AsBool() bool {
	return v.scalar != 0
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

func (v Value) TryInt() (n int64, ok bool) {
	_, ok = v.iface.(int64)
	if ok {
		n = v.AsInt()
	}
	return
}

func (v Value) TryFloat() (n float64, ok bool) {
	_, ok = v.iface.(float64)
	if ok {
		n = v.AsFloat()
	}
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
	_, ok = v.iface.(bool)
	if ok {
		b = v.scalar != 0
	}
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
