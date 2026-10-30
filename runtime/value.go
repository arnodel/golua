//go:build !noscalar
// +build !noscalar

package runtime

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unsafe"
)

// A Value is a runtime value.
//
// Values must be built with the constructor functions (IntValue, StringValue,
// TableValue, ...), as Equals and Hash depend on the scalar field they set: a
// string Value in particular carries its bytes or a hash of them there, so
// two Values made from equal strings are only equal if both were made by
// StringValue.
type Value struct {
	scalar uint64
	iface  interface{}
}

// In order to minimize memory allocations, the values of scalars Valuess are
// not stored in the iface field, but their type is, using a single interface{}
// value for each scalar type.
var (
	dummyInt64   interface{} = int64(0)   // All IntValue instances share this iface
	dummyFloat64 interface{} = float64(0) // All FloatValue instances share this iface
	dummyBool    interface{} = false      // All BoolVale instances share this iface
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
	case string:
		return StringValue(x)
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

// Some unsafe function to determine quickly if two interfaces have the same
// type.
func ifaceType(iface interface{}) uintptr {
	return *(*uintptr)(unsafe.Pointer(&iface))
}

func ifacePtr(iface interface{}) uintptr {
	return (*[2]uintptr)(unsafe.Pointer(&iface))[1]
}

var float64IfaceType = ifaceType(float64(1))
var int64IfaceType = ifaceType(int64(1))

// Equals returns true if v is equal to v2.  Provided that v and v2 have been
// built with the official constructor functions, it is equivalent to but
// slightly faster than '=='.
func (v Value) Equals(v2 Value) bool {
	// Different scalars mean different values, except for the floats 0.0 and
	// -0.0, whose scalars differ only by the sign bit.  This test is kept
	// small enough to inline; the rest of the comparison is in equalsSlow.
	if v.scalar != v2.scalar && (v.scalar|v2.scalar)<<1 != 0 {
		return false
	}
	return v.equalsSlow(v2)
}

func (v Value) equalsSlow(v2 Value) bool {
	ift := ifaceType(v.iface)
	if ift != ifaceType(v2.iface) {
		return false
	}
	if ift == float64IfaceType {
		// Special case for floats, as some non bitwise equal floats can be equal and vice-versa
		// NaN != NaN
		// -0 == +0
		return v.AsFloat() == v2.AsFloat()
	}
	if v.scalar != v2.scalar {
		return false
	}
	switch x := v.iface.(type) {
	case int64, float64:
		return true
	case string:
		// Equal scalars holding the strings' bytes mean equal strings.  Equal
		// scalars holding their hashes mean the bytes must be compared.
		if v.scalar < hashedStringTag {
			return true
		}
	case *Closure:
		return x.Equals(v2.iface.(*Closure))
	}
	return v.iface == v2.iface
}

// Hash returns a hash for the value.
func (v Value) Hash() uintptr {
	if v.scalar != 0 {
		return hashScalar(v.scalar, 0)
	}
	return hashInterface(v.iface, 0)
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

// The scalar of a string Value holds the string's bytes if it has at most 7,
// with the length in the top byte, and otherwise a hash of the string in the
// low 56 bits with the top byte set to hashedStringTag.  Either way Hash can
// use the scalar, and Equals can tell unequal strings apart by their scalars
// without comparing their bytes.  The empty string's scalar is 0.
const hashedStringTag = 0xFF << 56

// StringValue returns a Value holding the given arg.
func StringValue(s string) (v Value) {
	v.iface = s
	ls := len(s)
	if ls <= 7 {
		var bs [8]byte
		copy(bs[:], s)
		bs[7] = byte(ls)
		v.scalar = *(*uint64)(unsafe.Pointer(&bs[0]))
	} else {
		v.scalar = hashedStringTag | uint64(hashInterface(v.iface, 0))
	}
	return
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
	case *GoFunction, *Closure:
		return FunctionType
	case *Thread:
		return ThreadType
	case *UserData:
		return UserDataType
	default:
		return UnknownType
	}
}

// TypeName returns a string representing the type of the value (as in the Lua
// function type(v))
func (v Value) TypeName() string {
	if v.iface == nil {
		return "nil"
	}
	switch v.iface.(type) {
	case int64, float64:
		return "number"
	case bool:
		return "boolean"
	case string:
		return "string"
	case *Table:
		return "table"
		// return getName("table", v.iface.(*Table).Metatable())
	case *Code:
		return "code"
	case *GoFunction, *Closure:
		return "function"
	case *Thread:
		return "thread"
	case *UserData:
		return "userdata"
		// return getName("userdata", v.iface.(*UserData).Metatable())
	default:
		return "<unknown type>"
	}
}

// CustomTypeName is like TypeName but can be changed if the metatable has a
// __name field.
func (v Value) CustomTypeName() string {
	if v.iface == nil {
		return "nil"
	}
	switch v.iface.(type) {
	case int64, float64:
		return "number"
	case bool:
		return "boolean"
	case string:
		return "string"
	case *Table:
		return getName("table", v.iface.(*Table).Metatable())
	case *Code:
		return "code"
	case *GoFunction, *Closure:
		return "function"
	case *Thread:
		return "thread"
	case *UserData:
		return getName("userdata", v.iface.(*UserData).Metatable())
	default:
		return "<unknown type>"
	}
}

// ToString returns a simple string representation of a Value.  The boolean
// returned specifies whether this is a good string value or not ("good" should
// be defined).
func (v Value) ToString() (string, bool) {
	if v.iface == nil {
		return "nil", false
	}
	switch x := v.iface.(type) {
	case int64:
		return strconv.FormatInt(v.AsInt(), 10), true
	case float64:
		return FloatToString(v.AsFloat()), true
	case bool:
		return strconv.FormatBool(v.AsBool()), false
	case string:
		return v.AsString(), true
	case *Table:
		return fmt.Sprintf("%s: %p", getName("table", x.Metatable()), x), false
	case *Code:
		return fmt.Sprintf("code: %p", x), false
	case *GoFunction:
		return fmt.Sprintf("gofunction: %s", x.name), false
	case *Closure:
		return fmt.Sprintf("function: %p", x), false
	case *Thread:
		return fmt.Sprintf("thread: %p", x), false
	case *UserData:
		return fmt.Sprintf("%s: %p", getName("userdata", x.Metatable()), x), false
	default:
		return "<unknown>", false
	}
}

func getName(defaultName string, meta *Table) string {
	if v := RawGet(meta, metaName); !v.IsNil() {
		s, ok := v.ToString()
		if ok {
			return s
		}
	}
	return defaultName
}

// FloatToString formats a float64 for Lua as Lua 5.5 does
// (tostringbuffFloat in lobject.c): "%.15g", or "%.17g" when that does not
// read back as x, with ".0" added when the result looks like an integer so
// that floats are distinguished from integers.  Infinities and NaNs are
// spelled as C's printf spells them: "inf", "-inf", "nan" and "-nan".
func FloatToString(x float64) string {
	switch {
	case math.IsInf(x, 1):
		return "inf"
	case math.IsInf(x, -1):
		return "-inf"
	case math.IsNaN(x):
		if math.Signbit(x) {
			return "-nan"
		}
		return "nan"
	}
	s := strconv.FormatFloat(x, 'g', 15, 64)
	if y, _ := strconv.ParseFloat(s, 64); y != x {
		s = strconv.FormatFloat(x, 'g', 17, 64)
	}
	if strings.Trim(s, "-0123456789") == "" {
		s += ".0"
	}
	return s
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
	case *messageHandlerCont:
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

// AsCallable returns v as a Callable if possible (or panics)).  It is an
// optimisation as type assertion in Go seems to have a significant cost.
func (v Value) AsCallable() Callable {
	switch c := v.iface.(type) {
	case *Closure:
		return c
	case *GoFunction:
		return c
	default:
		panic("value is not a Callable")
	}
}

// AsThread returns v as a *Thread (or panics).
func (v Value) AsThread() *Thread {
	return v.iface.(*Thread)
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
	case *LuaCont:
		return cont, true
	case *Termination:
		return cont, true
	// These cases come after because this function is only used in
	// LuaCont.Next() and in that context it is unlikely that the next
	// continuation will be a *GoCont, and probably impossible that it is a
	// *MessageHandlerCont.
	case *GoCont:
		return cont, true
	case *messageHandlerCont:
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

// IsNaN returns true if v is a FloatValue which has a NaN value.  It is trying
// to do this as efficiently as possible.
func (v Value) IsNaN() bool {
	if ifaceType(v.iface) != float64IfaceType {
		return false
	}
	x := v.AsFloat()
	return math.IsNaN(x)
}
