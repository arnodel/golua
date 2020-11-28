package runtime

import (
	"fmt"
	"strconv"

	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/astcomp"
	"github.com/arnodel/golua/code"
	"github.com/arnodel/golua/ircomp"
	"github.com/arnodel/golua/parsing"
	"github.com/arnodel/golua/scanner"
)

const maxIndexChainLength = 100

// IsNil returns true if v is a nil value.
func IsNil(v Value) bool {
	return v == nil
}

// RawGet returns the item in a table for the given key, or nil if t is nil.  It
// doesn't check the metatable of t.
func RawGet(t *Table, k Value) Value {
	if t == nil {
		return nil
	}
	return t.Get(k)
}

// Index returns the item in a collection for the given key k, using the
// '__index' metamethod if appropriate.
func Index(t *Thread, coll Value, k Value) (Value, *Error) {
	for i := 0; i < maxIndexChainLength; i++ {
		if tbl, ok := coll.(*Table); ok {
			if val := RawGet(tbl, k); val != nil {
				return val, nil
			}
		}
		metaIdx := t.metaGetS(coll, "__index")
		if metaIdx == nil {
			return nil, nil
		}
		switch metaIdx.(type) {
		case *Table:
			coll = metaIdx
		default:
			res := NewTerminationWith(1, false)
			if err := Call(t, metaIdx, []Value{coll, k}, res); err != nil {
				return nil, err
			}
			return res.Get(0), nil
		}
	}
	return nil, NewErrorF("'__index' chain too long; possible loop")
}

// SetIndex sets the item in a collection for the given key, using the
// '__newindex' metamethod if appropriate.
func SetIndex(t *Thread, coll Value, idx Value, val Value) *Error {
	for i := 0; i < maxIndexChainLength; i++ {
		tbl, ok := coll.(*Table)
		if ok {
			if tbl.Get(idx) != nil {
				if err := tbl.SetCheck(idx, val); err != nil {
					return NewErrorE(err)
				}
				return nil
			}
		}
		metaNewIndex := t.metaGetS(coll, "__newindex")
		if metaNewIndex == nil {
			if ok {
				if err := tbl.SetCheck(idx, val); err != nil {
					return NewErrorE(err)
				}
			}
			return nil
		}
		switch metaNewIndex.(type) {
		case *Table:
			coll = metaNewIndex
		default:
			return Call(t, metaNewIndex, []Value{coll, idx, val}, NewTermination(nil, nil))
		}
	}
	return NewErrorF("'__newindex' chain too long; possible loop")
}

// Truth returns true if v is neither nil nor a false boolean.
func Truth(v Value) bool {
	if v == nil {
		return false
	}
	b, ok := v.(Bool)
	return !ok || bool(b)
}

// Metacall calls the metamethod called method on obj with the given arguments
// args, pushing the result to the continuation next.
func Metacall(t *Thread, obj Value, method string, args []Value, next Cont) (*Error, bool) {
	if f := t.metaGetS(obj, method); f != nil {
		return Call(t, f, args, next), true
	}
	return nil, false
}

// Continue tries tried to continue the value f or else use its '__call'
// metamethod and returns the continuations that needs to be run to get the
// results.
func Continue(t *Thread, f Value, next Cont) (Cont, *Error) {
	callable, ok := f.(Callable)
	if ok {
		return callable.Continuation(next), nil
	}
	cont, err, ok := metacont(t, f, "__call", next)
	if !ok {
		return nil, NewErrorF("cannot call %v", f)
	}
	if cont != nil {
		cont.Push(f)
	}
	return cont, err
}

// Call calls f with arguments args, pushing the results on next.  It may use
// the metamethod '__call' if f is not callable.
func Call(t *Thread, f Value, args []Value, next Cont) *Error {
	if f == nil {
		return NewErrorS("attempt to call a nil value")
	}
	callable, ok := f.(Callable)
	if ok {
		return t.call(callable, args, next)
	}
	err, ok := Metacall(t, f, "__call", append([]Value{f}, args...), next)
	if ok {
		return err
	}
	return NewErrorS("call expects a callable")
}

// Call1 is a convenience method that calls f with arguments args and returns
// exactly one value.
func Call1(t *Thread, f Value, args ...Value) (Value, *Error) {
	term := NewTerminationWith(1, false)
	if err := Call(t, f, args, term); err != nil {
		return nil, err
	}
	return term.Get(0), nil
}

// AsString returns x as a String and a boolean which is true if this is a
// 'good' conversion. TODO: refactor or explain the meaning of the boolean
// better.
func AsString(x Value) (String, bool) {
	if x == nil {
		return String("nil"), true
	}
	switch xx := x.(type) {
	case String:
		return xx, true
	case Int:
		return String(strconv.Itoa(int(xx))), true
	case Float:
		return String(strconv.FormatFloat(float64(xx), 'g', -1, 64)), true
	case Bool:
		if xx {
			return String("true"), false
		}
		return String("false"), false
	}
	return String(""), false
}

// Concat returns x .. y, possibly calling the '__concat' metamethod.
func Concat(t *Thread, x, y Value) (Value, *Error) {
	if sx, ok := AsString(x); ok {
		if sy, ok := AsString(y); ok {
			return sx + sy, nil
		}
	}
	res, err, ok := metabin(t, "__concat", x, y)
	if ok {
		return res, err
	}
	return nil, NewErrorS("concat expects concatable values")
}

// IntLen returns the length of v as an Int, possibly calling the '__len'
// metamethod.  This is an optimization of Len for an integer output.
func IntLen(t *Thread, v Value) (Int, *Error) {
	if s, ok := v.(String); ok {
		return Int(len(s)), nil
	}
	res := NewTerminationWith(1, false)
	err, ok := Metacall(t, v, "__len", []Value{v}, res)
	if ok {
		if err != nil {
			return Int(0), err
		}
		l, ok := ToInt(res.Get(0))
		if !ok {
			err = NewErrorS("len should return an integer")
		}
		return l, err
	}
	if tbl, ok := v.(*Table); ok {
		return tbl.Len(), nil
	}
	return Int(0), NewErrorS("Cannot compute len")
}

// Len returns the length of v, possibly calling the '__len' metamethod.
func Len(t *Thread, v Value) (Value, *Error) {
	if s, ok := v.(String); ok {
		return Int(len(s)), nil
	}
	res := NewTerminationWith(1, false)
	err, ok := Metacall(t, v, "__len", []Value{v}, res)
	if ok {
		if err != nil {
			return nil, err
		}
		return res.Get(0), nil
	}
	if tbl, ok := v.(*Table); ok {
		return tbl.Len(), nil
	}
	return nil, NewErrorS("Cannot compute len")
}

// Type returns a String describing the Lua type of v.
func Type(v Value) String {
	if v == nil {
		return String("nil")
	}
	switch v.(type) {
	case String:
		return String("string")
	case Int, Float:
		return String("number")
	case *Table:
		return String("table")
	case Bool:
		return String("boolean")
	case *Closure, *GoFunction:
		return String("function")
	case *Thread:
		return String("thread")
	case *UserData:
		return String("userdata")
	}
	return String(fmt.Sprintf("unknown(%+v)", v))
}

// SetEnv sets the item in the table t for a string key.  Useful when writing
// libraries
func SetEnv(t *Table, name string, v Value) {
	t.Set(String(name), v)
}

// SetEnvGoFunc sets the item in the table t for a string key to be a GoFunction
// defined by f.  Useful when writing libraries
func SetEnvGoFunc(t *Table, name string, f func(*Thread, *GoCont) (Cont, *Error), nArgs int, hasEtc bool) {
	t.Set(String(name), &GoFunction{
		f:      f,
		name:   name,
		nArgs:  nArgs,
		hasEtc: hasEtc,
	})
}

// ParseLuaChunk parses a string as a Lua statement and returns the AST.
func ParseLuaChunk(name string, source []byte) (*ast.BlockStat, error) {
	s := scanner.New(name, source)
	stat, err := parsing.ParseChunk(s.Scan)
	if err != nil {
		parseErr, ok := err.(parsing.Error)
		if !ok {
			return nil, err
		}
		return nil, NewSyntaxError(name, parseErr)
	}
	return &stat, nil
}

// CompileLuaChunk parses and compiles the source as a Lua Chunk and returns the
// compile code Unit.
func CompileLuaChunk(name string, source []byte) (*code.Unit, error) {
	stat, err := ParseLuaChunk(name, source)
	if err != nil {
		return nil, err
	}
	irCode := astcomp.CompileLuaChunk(name, *stat)
	constants := append(irCode.Constants, irCode)
	kc := ircomp.NewConstantCompiler(constants, code.NewBuilder(name))
	kc.QueueConstant(uint(len(irCode.Constants)))
	return kc.CompileQueue(), nil
}

// CompileAndLoadLuaChunk parses, compiles and loads a Lua chunk from source and
// returns the closure that runs the chunk in the given global environment.
func CompileAndLoadLuaChunk(name string, source []byte, env Value) (*Closure, error) {
	unit, err := CompileLuaChunk(name, source)
	if err != nil {
		return nil, err
	}
	return LoadLuaUnit(unit, env), nil
}

func metacont(t *Thread, obj Value, method string, next Cont) (Cont, *Error, bool) {
	f := t.metaGetS(obj, method)
	if IsNil(f) {
		return nil, nil, false
	}
	cont, err := Continue(t, f, next)
	if err != nil {
		return nil, err, true
	}
	return cont, nil, true
}

func metabin(t *Thread, f string, x Value, y Value) (Value, *Error, bool) {
	xy := []Value{x, y}
	res := NewTerminationWith(1, false)
	err, ok := Metacall(t, x, f, xy, res)
	if !ok {
		err, ok = Metacall(t, y, f, xy, res)
	}
	if ok {
		return res.Get(0), err, true
	}
	return nil, nil, false
}

func metaun(t *Thread, f string, x Value) (Value, *Error, bool) {
	res := NewTerminationWith(1, false)
	err, ok := Metacall(t, x, f, []Value{x}, res)
	if ok {
		return res.Get(0), err, true
	}
	return nil, nil, false
}
