package runtime

import (
	"errors"
	"strconv"

	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/lexer"
	"github.com/arnodel/golua/parser"
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

func getindex(t *Thread, coll Value, idx Value) (Value, error) {
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
		return getindex(t, metaIdx, idx)
	default:
		res := NewTerminationWith(1, false)
		if err := Call(t, metaIdx, []Value{idx}, res); err != nil {
			return nil, err
		}
		return res.Get(0), nil
	}
}

func setindex(t *Thread, coll Value, idx Value, val Value) error {
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
		return Call(t, metaNewIndex, []Value{coll, idx, val}, nil)
	}
}

func truth(v Value) bool {
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

func Metacall(t *Thread, obj Value, method string, args []Value, next Continuation) (error, bool) {
	if f := t.MetaGetS(obj, method); f != nil {
		return Call(t, f, args, next), true
	}
	return nil, false
}

func Call(t *Thread, f Value, args []Value, next Continuation) error {
	callable, ok := f.(Callable)
	if ok {
		return t.RunContinuation(ContWithArgs(callable, args, next))
	}
	err, ok := Metacall(t, f, "__call", append([]Value{f}, args...), next)
	if ok {
		return err
	}
	return errors.New("call expects a callable")
}

func metabin(t *Thread, f string, x Value, y Value) (Value, error, bool) {
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

func metaun(t *Thread, f string, x Value) (Value, error, bool) {
	res := NewTerminationWith(1, false)
	err, ok := Metacall(t, x, f, []Value{x}, res)
	if ok {
		return res.Get(0), err, true
	}
	return nil, nil, false
}

func AsString(x Value) (String, bool) {
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
	}
	return String(""), false
}

func concat(t *Thread, x, y Value) (Value, error) {
	if sx, ok := AsString(x); ok {
		if sy, ok := AsString(y); ok {
			return sx + sy, nil
		}
	}
	res, err, ok := metabin(t, "__concat", x, y)
	if ok {
		return res, err
	}
	return nil, errors.New("concat expects concatable values")
}

func length(t *Thread, v Value) (Value, error) {
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
	return nil, errors.New("Cannot compute len")
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
	}
	return String("unknown")
}

func SetEnvFunc(t *Table, name string, f func(*Thread, []Value, Continuation) (Continuation, error)) {
	t.Set(String(name), GoFunction(f))
}

func SetEnv(t *Table, name string, v Value) {
	t.Set(String(name), v)
}

func CompileLuaChunk(source []byte, env *Table) (*Closure, error) {
	p := parser.NewParser()
	s := lexer.NewLexer(source)
	tree, err := p.Parse(s)
	if err != nil {
		return nil, err
	}
	c := tree.(ast.BlockStat).CompileChunk()
	kc := c.NewConstantCompiler()
	unit := kc.CompileQueue()
	return LoadLuaUnit(unit, env), nil
}
