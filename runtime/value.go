package runtime

import (
	"fmt"
	"hash/maphash"
	"math"
	"strconv"
	"strings"
	"unsafe"
)

// A Value is a runtime value.  It is two words: a scalar and a pointer.
//
//   - nil is the zero Value.
//   - An integer, a float or a boolean has its bits in the scalar, and its
//     pointer is one of three fixed addresses that says which it is (an
//     immediate value).
//   - Any other value is a reference: the pointer is to its object, and the
//     top byte of the scalar is its kind.  The rest of the scalar is the
//     bytes or a hash of a string, or the address of the object, so that
//     Equals and Hash mostly need the scalar alone.
//
// Values must be built with the constructor functions (IntValue, StringValue,
// TableValue, ...).  Values cannot be compared with ==; use Equals.
type Value struct {
	_      [0]func()
	scalar uint64
	ptr    unsafe.Pointer
}

// The pointers of immediate values point into this array: an integer's to
// its first byte, a float's to its second, a boolean's to its third.
var immediate [3]byte

const (
	tagInt uintptr = iota
	tagFloat
	tagBool
	immediateCount
)

// tag returns tagInt, tagFloat or tagBool for an immediate value and a larger
// number for any other.
func (v Value) tag() uintptr {
	return uintptr(v.ptr) - uintptr(unsafe.Pointer(&immediate[0]))
}

func (v Value) isImmediate() bool {
	return v.tag() < immediateCount
}

func (v Value) isInt() bool {
	return v.ptr == unsafe.Pointer(&immediate[tagInt])
}

func (v Value) isFloat() bool {
	return v.ptr == unsafe.Pointer(&immediate[tagFloat])
}

// A valueKind is the top byte of the scalar of a reference.  0 is not a
// kind, so that the kind of nil is none.
type valueKind uint8

const (
	kindTable valueKind = iota + 1
	kindClosure
	kindGoFunction
	kindLuaCont
	kindGoCont
	kindTermination
	kindMessageHandlerCont
	kindProtectedCall
	kindThread
	kindUserData
	kindCode
	kindBox
	kindArray
	kindOther // the pointer is to an interface{} holding the Go value

	// A string of n bytes, n <= 7, has kind kindShortString+n and its bytes
	// in the scalar.  A longer one has kind kindLongString and a hash of its
	// bytes in the scalar.  The string kinds are those with the top bit set.
	kindShortString valueKind = 0x80
	kindLongString  valueKind = 0xFF
)

const (
	kindShift   = 56
	addressMask = 1<<kindShift - 1

	// Closures are equal when they have the same code and upvalues, and Go
	// values when Go says so, whatever their addresses: the scalars of
	// those two kinds hold the kind alone.
	closureScalar = uint64(kindClosure) << kindShift
	otherScalar   = uint64(kindOther) << kindShift
)

// reference returns the Value of kind k for the object at p.
func reference(k valueKind, p unsafe.Pointer) Value {
	return Value{scalar: uint64(k)<<kindShift | uint64(uintptr(p))&addressMask, ptr: p}
}

// kind returns the kind of v if it is a reference, 0 otherwise.
func (v Value) kind() valueKind {
	if v.isImmediate() {
		return 0
	}
	return valueKind(v.scalar >> kindShift)
}

// is reports whether v is a reference of kind k (not a string kind).
func (v Value) is(k valueKind) bool {
	return valueKind(v.scalar>>kindShift) == k && !v.isImmediate()
}

func (v Value) isString() bool {
	return int64(v.scalar) < 0 && !v.isImmediate()
}

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
	case *Table:
		return TableValue(x)
	case *Closure:
		return closureValue(x)
	case *GoFunction:
		return reference(kindGoFunction, unsafe.Pointer(x))
	case *LuaCont:
		return luaContValue(x)
	case *GoCont:
		return goContValue(x)
	case *Termination:
		return reference(kindTermination, unsafe.Pointer(x))
	case *messageHandlerCont:
		return reference(kindMessageHandlerCont, unsafe.Pointer(x))
	case *protectedCall:
		return reference(kindProtectedCall, unsafe.Pointer(x))
	case *Thread:
		return ThreadValue(x)
	case *UserData:
		return UserDataValue(x)
	case *Code:
		return CodeValue(x)
	case Cell:
		return BoxValue(x)
	case []Value:
		return ArrayValue(x)
	default:
		p := new(interface{})
		*p = i
		return Value{scalar: otherScalar, ptr: unsafe.Pointer(p)}
	}
}

// Interface turns the Value into an interface.  As AsValue, this can trigger
// allocations so use with caution.
func (v Value) Interface() interface{} {
	switch v.tag() {
	case tagInt:
		return v.AsInt()
	case tagFloat:
		return v.AsFloat()
	case tagBool:
		return v.AsBool()
	}
	if v.ptr == nil {
		return nil
	}
	switch k := valueKind(v.scalar >> kindShift); k {
	case kindTable:
		return (*Table)(v.ptr)
	case kindClosure:
		return (*Closure)(v.ptr)
	case kindGoFunction:
		return (*GoFunction)(v.ptr)
	case kindLuaCont:
		return (*LuaCont)(v.ptr)
	case kindGoCont:
		return (*GoCont)(v.ptr)
	case kindTermination:
		return (*Termination)(v.ptr)
	case kindMessageHandlerCont:
		return (*messageHandlerCont)(v.ptr)
	case kindProtectedCall:
		return (*protectedCall)(v.ptr)
	case kindThread:
		return (*Thread)(v.ptr)
	case kindUserData:
		return (*UserData)(v.ptr)
	case kindCode:
		return (*Code)(v.ptr)
	case kindBox:
		return Cell{(*Value)(v.ptr)}
	case kindArray:
		return *(*[]Value)(v.ptr)
	case kindOther:
		return *(*interface{})(v.ptr)
	default:
		return *(*string)(v.ptr)
	}
}

// Equals returns true if v is equal to v2.
func (v Value) Equals(v2 Value) bool {
	// Different scalars mean different values, except for the floats 0.0 and
	// -0.0, whose scalars differ only by the sign bit.  This test is kept
	// small enough to inline; the rest of the comparison is in equalsSlow.
	if v.scalar != v2.scalar && (v.scalar|v2.scalar)&^signBit != 0 {
		return false
	}
	return v.equalsSlow(v2)
}

// signBit is the sign bit of a float64 held in a Value's scalar.
const signBit = 1 << 63

// equalsString reports whether v equals k, where k is known to be a string.
// It is what Equals computes, without a call in the common case: unequal
// scalars mean unequal values, and a reference with the scalar of a string
// is a string, equal to k when the scalars hold the bytes (strings of up to
// 7 bytes).  Only equal hashes of longer strings need the bytes compared.
func (v Value) equalsString(k Value) bool {
	if v.scalar != k.scalar || v.isImmediate() {
		return false
	}
	return k.scalar < hashedStringTag || v.equalsLongString(k)
}

func (v Value) equalsLongString(k Value) bool {
	return *(*string)(v.ptr) == *(*string)(k.ptr)
}

func (v Value) equalsSlow(v2 Value) bool {
	if v.ptr == v2.ptr {
		// The same object, or immediate values of the same type.
		if v.isFloat() {
			// Some non bitwise equal floats are equal and vice-versa:
			// NaN != NaN, -0 == +0
			return v.AsFloat() == v2.AsFloat()
		}
		return v.scalar == v2.scalar
	}
	if v.scalar != v2.scalar || v.isImmediate() || v2.isImmediate() || v.ptr == nil || v2.ptr == nil {
		return false
	}
	// Two objects with the same scalar, so of the same kind.
	switch k := valueKind(v.scalar >> kindShift); {
	case k == kindLongString:
		return *(*string)(v.ptr) == *(*string)(v2.ptr)
	case k >= kindShortString:
		return true
	case k == kindClosure:
		return (*Closure)(v.ptr).Equals((*Closure)(v2.ptr))
	case k == kindOther:
		return *(*interface{})(v.ptr) == *(*interface{})(v2.ptr)
	}
	return false
}

// Hash returns a hash for the value.
func (v Value) Hash() uintptr {
	if v.scalar != 0 && v.scalar != closureScalar && v.scalar != otherScalar {
		return hashScalar(v.scalar, 0)
	}
	return v.hashSlow()
}

func (v Value) hashSlow() uintptr {
	if !v.isImmediate() && v.scalar == otherScalar {
		return hashInterface(*(*interface{})(v.ptr), 0)
	}
	if v.scalar != 0 && v.isImmediate() {
		return hashScalar(v.scalar, 0)
	}
	// 0, 0.0 and false by their tags, a closure by its address.
	return hashScalar(uint64(uintptr(v.ptr)), 0)
}

// IntValue returns a Value holding the given arg.
func IntValue(n int64) Value {
	return Value{scalar: uint64(n), ptr: unsafe.Pointer(&immediate[tagInt])}
}

// FloatValue returns a Value holding the given arg.
func FloatValue(f float64) Value {
	return Value{scalar: math.Float64bits(f), ptr: unsafe.Pointer(&immediate[tagFloat])}
}

// BoolValue returns a Value holding the given arg.
func BoolValue(b bool) Value {
	var s uint64
	if b {
		s = 1
	}
	return Value{scalar: s, ptr: unsafe.Pointer(&immediate[tagBool])}
}

// The scalar of a string Value holds the string's bytes if it has at most 7,
// with the kind (which gives the length) in the top byte, and otherwise a
// hash of the string in the low 56 bits with the top byte set to
// kindLongString.  Either way Hash can use the scalar, and Equals can tell
// unequal strings apart by their scalars without comparing their bytes.
const hashedStringTag = uint64(kindLongString) << kindShift

var stringSeed = maphash.MakeSeed()

// StringValue returns a Value holding the given arg.
func StringValue(s string) (v Value) {
	p := new(string)
	*p = s
	v.ptr = unsafe.Pointer(p)
	ls := len(s)
	if ls <= 7 {
		var bs [8]byte
		copy(bs[:], s)
		bs[7] = byte(kindShortString) | byte(ls)
		v.scalar = *(*uint64)(unsafe.Pointer(&bs[0]))
	} else {
		v.scalar = hashedStringTag | maphash.String(stringSeed, s)
	}
	return
}

// TableValue returns a Value holding the given arg.
func TableValue(t *Table) Value {
	return reference(kindTable, unsafe.Pointer(t))
}

// FunctionValue returns a Value holding the given arg.
func FunctionValue(c Callable) Value {
	switch f := c.(type) {
	case *Closure:
		return closureValue(f)
	case *GoFunction:
		return reference(kindGoFunction, unsafe.Pointer(f))
	}
	return AsValue(c)
}

func closureValue(c *Closure) Value {
	return Value{scalar: closureScalar, ptr: unsafe.Pointer(c)}
}

// ContValue returns a Value holding the given arg.
func ContValue(c Cont) Value {
	switch c := c.(type) {
	case *LuaCont:
		return luaContValue(c)
	case *GoCont:
		return goContValue(c)
	}
	return AsValue(c)
}

func luaContValue(c *LuaCont) Value {
	return reference(kindLuaCont, unsafe.Pointer(c))
}

func goContValue(c *GoCont) Value {
	return reference(kindGoCont, unsafe.Pointer(c))
}

// ArrayValue returns a Value holding the given arg.
func ArrayValue(a []Value) Value {
	p := new([]Value)
	*p = a
	return reference(kindArray, unsafe.Pointer(p))
}

// CodeValue returns a Value holding the given arg.
func CodeValue(c *Code) Value {
	return reference(kindCode, unsafe.Pointer(c))
}

// ThreadValue returns a Value holding the given arg.
func ThreadValue(t *Thread) Value {
	return reference(kindThread, unsafe.Pointer(t))
}

// LightUserDataValue returns a Value holding the given arg.
func LightUserDataValue(d LightUserData) Value {
	return AsValue(d)
}

// UserDataValue returns a Value holding the given arg.
func UserDataValue(u *UserData) Value {
	return reference(kindUserData, unsafe.Pointer(u))
}

// NilValue is a value holding Nil.
var NilValue = Value{}

// Type returns the ValueType of v.
func (v Value) Type() ValueType {
	switch v.tag() {
	case tagInt:
		return IntType
	case tagFloat:
		return FloatType
	case tagBool:
		return BoolType
	}
	if v.ptr == nil {
		return NilType
	}
	switch k := valueKind(v.scalar >> kindShift); {
	case k >= kindShortString:
		return StringType
	case k == kindTable:
		return TableType
	case k == kindCode:
		return CodeType
	case k == kindClosure, k == kindGoFunction:
		return FunctionType
	case k == kindThread:
		return ThreadType
	case k == kindUserData:
		return UserDataType
	default:
		return UnknownType
	}
}

// TypeName returns a string representing the type of the value (as in the Lua
// function type(v))
func (v Value) TypeName() string {
	switch v.Type() {
	case NilType:
		return "nil"
	case IntType, FloatType:
		return "number"
	case BoolType:
		return "boolean"
	case StringType:
		return "string"
	case TableType:
		return "table"
	case CodeType:
		return "code"
	case FunctionType:
		return "function"
	case ThreadType:
		return "thread"
	case UserDataType:
		return "userdata"
	default:
		return "<unknown type>"
	}
}

// CustomTypeName is like TypeName but can be changed if the metatable has a
// __name field.
func (v Value) CustomTypeName() string {
	switch v.Type() {
	case TableType:
		return getName("table", v.AsTable().Metatable())
	case UserDataType:
		return getName("userdata", v.AsUserData().Metatable())
	}
	return v.TypeName()
}

// ToString returns a simple string representation of a Value.  The boolean
// returned specifies whether this is a good string value or not ("good" should
// be defined).
func (v Value) ToString() (string, bool) {
	switch v.tag() {
	case tagInt:
		return strconv.FormatInt(v.AsInt(), 10), true
	case tagFloat:
		return FloatToString(v.AsFloat()), true
	case tagBool:
		return strconv.FormatBool(v.AsBool()), false
	}
	if v.ptr == nil {
		return "nil", false
	}
	switch k := valueKind(v.scalar >> kindShift); {
	case k >= kindShortString:
		return v.AsString(), true
	case k == kindTable:
		x := (*Table)(v.ptr)
		return fmt.Sprintf("%s: %p", getName("table", x.Metatable()), x), false
	case k == kindCode:
		return fmt.Sprintf("code: %p", (*Code)(v.ptr)), false
	case k == kindGoFunction:
		return fmt.Sprintf("gofunction: %s", (*GoFunction)(v.ptr).name), false
	case k == kindClosure:
		return fmt.Sprintf("function: %p", (*Closure)(v.ptr)), false
	case k == kindThread:
		return fmt.Sprintf("thread: %p", (*Thread)(v.ptr)), false
	case k == kindUserData:
		x := (*UserData)(v.ptr)
		return fmt.Sprintf("%s: %p", getName("userdata", x.Metatable()), x), false
	default:
		return "<unknown>", false
	}
}

func getName(defaultName string, meta *Table) string {
	if v := RawGet(meta, metaNames[metaName]); !v.IsNil() {
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
	switch v.tag() {
	case tagInt:
		return IntType
	case tagFloat:
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
	return math.Float64frombits(v.scalar)
}

// AsBool returns v as a bool (or panics).
func (v Value) AsBool() bool {
	return v.scalar != 0
}

// wrongKind panics: a Value was taken as a kind of value it is not.
func wrongKind(what string) {
	panic("value is not " + what)
}

// AsString returns v as a string (or panics).
func (v Value) AsString() string {
	if !v.isString() {
		wrongKind("a string")
	}
	return *(*string)(v.ptr)
}

// AsTable returns v as a *Table (or panics).
func (v Value) AsTable() *Table {
	if !v.is(kindTable) {
		wrongKind("a table")
	}
	return (*Table)(v.ptr)
}

// AsCont returns v as a Cont, by looking at the concrete type (or panics).
func (v Value) AsCont() Cont {
	c, ok := v.TryCont()
	if !ok {
		wrongKind("a continuation")
	}
	return c
}

// AsArray returns v as a [] (or panics).
func (v Value) AsArray() []Value {
	if !v.is(kindArray) {
		wrongKind("an array")
	}
	return *(*[]Value)(v.ptr)
}

// AsClosure returns v as a *Closure (or panics).
func (v Value) AsClosure() *Closure {
	if !v.is(kindClosure) {
		wrongKind("a closure")
	}
	return (*Closure)(v.ptr)
}

// AsCode returns v as a *Code (or panics).
func (v Value) AsCode() *Code {
	if !v.is(kindCode) {
		wrongKind("code")
	}
	return (*Code)(v.ptr)
}

// AsUserData returns v as a *UserData (or panics).
func (v Value) AsUserData() *UserData {
	if !v.is(kindUserData) {
		wrongKind("a userdata")
	}
	return (*UserData)(v.ptr)
}

// AsCallable returns v as a Callable if possible (or panics)).
func (v Value) AsCallable() Callable {
	c, ok := v.TryCallable()
	if !ok {
		wrongKind("a Callable")
	}
	return c
}

// AsThread returns v as a *Thread (or panics).
func (v Value) AsThread() *Thread {
	if !v.is(kindThread) {
		wrongKind("a thread")
	}
	return (*Thread)(v.ptr)
}

// TryInt converts v to type int64 if possible (ok is false otherwise).
func (v Value) TryInt() (n int64, ok bool) {
	if v.isInt() {
		return v.AsInt(), true
	}
	return 0, false
}

// TryFloat converts v to type float64 if possible (ok is false otherwise).
func (v Value) TryFloat() (n float64, ok bool) {
	if v.isFloat() {
		return v.AsFloat(), true
	}
	return 0, false
}

// TryString converts v to type string if possible (ok is false otherwise).
func (v Value) TryString() (s string, ok bool) {
	if v.isString() {
		return *(*string)(v.ptr), true
	}
	return "", false
}

// TryCallable converts v to type Callable if possible by looking at the
// possible concrete types (ok is false otherwise).
func (v Value) TryCallable() (c Callable, ok bool) {
	switch v.kind() {
	case kindClosure:
		return (*Closure)(v.ptr), true
	case kindGoFunction:
		return (*GoFunction)(v.ptr), true
	default:
		return nil, false
	}
}

// TryClosure converts v to type *Closure if possible (ok is false otherwise).
func (v Value) TryClosure() (c *Closure, ok bool) {
	if v.is(kindClosure) {
		return (*Closure)(v.ptr), true
	}
	return nil, false
}

// tryGoFunction converts v to type *GoFunction if possible (ok is false
// otherwise).
func (v Value) tryGoFunction() (f *GoFunction, ok bool) {
	if v.is(kindGoFunction) {
		return (*GoFunction)(v.ptr), true
	}
	return nil, false
}

// tryLuaCont converts v to type *LuaCont if possible (ok is false
// otherwise).
func (v Value) tryLuaCont() (c *LuaCont, ok bool) {
	if v.is(kindLuaCont) {
		return (*LuaCont)(v.ptr), true
	}
	return nil, false
}

// tryBox returns the box v holds if it holds one (ok is false otherwise).
func (v Value) tryBox() (c Cell, ok bool) {
	if v.is(kindBox) {
		return Cell{(*Value)(v.ptr)}, true
	}
	return Cell{}, false
}

// TryThread converts v to type *Thread if possible (ok is false otherwise).
func (v Value) TryThread() (t *Thread, ok bool) {
	if v.is(kindThread) {
		return (*Thread)(v.ptr), true
	}
	return nil, false
}

// TryTable converts v to type *Table if possible (ok is false otherwise).
func (v Value) TryTable() (t *Table, ok bool) {
	if v.is(kindTable) {
		return (*Table)(v.ptr), true
	}
	return nil, false
}

// TryUserData converts v to type *UserData if possible (ok is false otherwise).
func (v Value) TryUserData() (u *UserData, ok bool) {
	if v.is(kindUserData) {
		return (*UserData)(v.ptr), true
	}
	return nil, false
}

// TryBool converts v to type bool if possible (ok is false otherwise).
func (v Value) TryBool() (b bool, ok bool) {
	if v.tag() == tagBool {
		return v.scalar != 0, true
	}
	return false, false
}

// TryCont returns v as a Cont, by looking at its kind (ok is false if it
// doesn't implement the Cont interface).
func (v Value) TryCont() (c Cont, ok bool) {
	switch v.kind() {
	case kindLuaCont:
		return (*LuaCont)(v.ptr), true
	case kindTermination:
		return (*Termination)(v.ptr), true
	case kindGoCont:
		return (*GoCont)(v.ptr), true
	case kindMessageHandlerCont:
		return (*messageHandlerCont)(v.ptr), true
	case kindProtectedCall:
		return (*protectedCall)(v.ptr), true
	case kindOther:
		c, ok = (*(*interface{})(v.ptr)).(Cont)
		return c, ok
	default:
		return nil, false
	}
}

// TryCode converts v to type *Code if possible (ok is false otherwise).
func (v Value) TryCode() (c *Code, ok bool) {
	if v.is(kindCode) {
		return (*Code)(v.ptr), true
	}
	return nil, false
}

// IsNil returns true if v is nil.
func (v Value) IsNil() bool {
	return v.ptr == nil
}

// IsNaN returns true if v is a FloatValue which has a NaN value.  It is trying
// to do this as efficiently as possible.
func (v Value) IsNaN() bool {
	if !v.isFloat() {
		return false
	}
	x := v.AsFloat()
	return math.IsNaN(x)
}
