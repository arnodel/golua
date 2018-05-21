package runtime

import (
	"fmt"
	"strconv"

	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/parser"
	"github.com/arnodel/golua/scanner"
)

func IsNil(v Value) bool {
	return v == nil || v == NilType{}
}

func RawGet(t *Table, k Value) Value {
	if t == nil {
		return nil
	}
	return t.Get(k)
}

func Index(t *Thread, coll Value, idx Value) (Value, *Error) {
	if tbl, ok := coll.(*Table); ok {
		if val := RawGet(tbl, idx); val != nil {
			return val, nil
		}
	}
	metaIdx := t.MetaGetS(coll, "__index")
	if metaIdx == nil {
		return nil, nil
	}
	switch metaIdx.(type) {
	case *Table:
		return Index(t, metaIdx, idx)
	default:
		res := NewTerminationWith(1, false)
		if err := Call(t, metaIdx, []Value{coll, idx}, res); err != nil {
			return nil, err
		}
		return res.Get(0), nil
	}
}

func setindex(t *Thread, coll Value, idx Value, val Value) *Error {
	tbl, ok := coll.(*Table)
	if ok {
		if tbl.Get(idx) != nil {
			tbl.Set(idx, val)
			return nil
		}
	}
	metaNewIndex := t.MetaGetS(coll, "__newindex")
	if metaNewIndex == nil {
		if ok {
			tbl.Set(idx, val)
		}
		return nil
	}
	switch metaNewIndex.(type) {
	case *Table:
		return setindex(t, metaNewIndex, idx, val)
	default:
		return Call(t, metaNewIndex, []Value{coll, idx, val}, NewTermination(nil, nil))
	}
}

func Truth(v Value) bool {
	if v == nil {
		return false
	}
	switch x := v.(type) {
	case NilType:
		return false
	case Bool:
		return bool(x)
	default:
		return true
	}
}

func Metacall(t *Thread, obj Value, method string, args []Value, next Cont) (*Error, bool) {
	if f := t.MetaGetS(obj, method); f != nil {
		return Call(t, f, args, next), true
	}
	return nil, false
}

func Metacont(t *Thread, obj Value, method string, next Cont) (Cont, *Error, bool) {
	f := t.MetaGetS(obj, method)
	if IsNil(f) {
		return nil, nil, false
	}
	cont, err := Continue(t, f, next)
	if err != nil {
		return nil, err, true
	}
	return cont, nil, true
}

func Continue(t *Thread, f Value, next Cont) (Cont, *Error) {
	callable, ok := f.(Callable)
	if ok {
		return callable.Continuation(next), nil
	}
	cont, err, ok := Metacont(t, f, "__call", next)
	if !ok {
		return nil, NewErrorF("cannot call %v", f)
	}
	if cont != nil {
		cont.Push(f)
	}
	return cont, err
}

func Call(t *Thread, f Value, args []Value, next Cont) *Error {
	callable, ok := f.(Callable)
	if ok {
		return t.Call(callable, args, next)
	}
	err, ok := Metacall(t, f, "__call", append([]Value{f}, args...), next)
	if ok {
		return err
	}
	return NewErrorS("call expects a callable")
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
			return String("true"), true
		}
		return String("false"), true
	case NilType:
		return String("nil"), true
	}
	return String(""), false
}

func concat(t *Thread, x, y Value) (Value, *Error) {
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

func Len(t *Thread, v Value) (Value, *Error) {
	if s, ok := v.(String); ok {
		return Int(len(s)), nil
	}
	res := NewTerminationWith(1, false)
	err, ok := Metacall(t, v, "__len", []Value{v}, res)
	if ok {
		return res.Get(0), err
	}
	if tbl, ok := v.(*Table); ok {
		return tbl.Len(), nil
	}
	return nil, NewErrorS("Cannot compute len")
}

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
	case NilType:
		return String("nil")
	case Bool:
		return String("bool")
	case *Closure:
		return String("function")
	case *Thread:
		return String("thread")
	}
	return String(fmt.Sprintf("unknown(%+v)", v))
}

func SetEnvFunc(t *Table, name string, f func(*Thread, []Value, Cont) (Cont, *Error)) {
	t.Set(String(name), &GoFunction{
		nArgs:  0,
		hasEtc: true,
		f: func(t *Thread, c *GoCont) (Cont, *Error) {
			return f(t, *c.etc, c.next)
		},
	})
}

func SetEnv(t *Table, name string, v Value) {
	t.Set(String(name), v)
}

func SetEnvGoFunc(t *Table, name string, f func(*Thread, *GoCont) (Cont, *Error), nArgs int, hasEtc bool) {
	t.Set(String(name), &GoFunction{
		f:      f,
		nArgs:  nArgs,
		hasEtc: hasEtc,
	})
}

func CompileLuaChunk(name string, source []byte, env *Table) (*Closure, error) {
	p := parser.NewParser()
	s := scanner.New(name, source)
	tree, err := p.Parse(s)
	if err != nil {
		// It would be better if the parser just forwarded the
		// tokenising error but...
		if s.Error() != nil {
			return nil, s.Error()
		}
		return nil, err
	}
	c := tree.(ast.BlockStat).CompileChunk(name)
	kc := c.NewConstantCompiler()
	unit := kc.CompileQueue()
	return LoadLuaUnit(unit, env), nil
}
