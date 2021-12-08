package runtime

import (
	"fmt"

	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/astcomp"
	"github.com/arnodel/golua/code"
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ircomp"
	"github.com/arnodel/golua/parsing"
	"github.com/arnodel/golua/scanner"
)

const maxIndexChainLength = 100

// IsNil returns true if v is a nil value.
// TODO: remove
func IsNil(v Value) bool {
	return v.IsNil()
}

// RawGet returns the item in a table for the given key, or nil if t is nil.  It
// doesn't check the metatable of t.
func RawGet(t *Table, k Value) Value {
	if t == nil {
		return NilValue
	}
	return t.Get(k)
}

// Index returns the item in a collection for the given key k, using the
// '__index' metamethod if appropriate.
// Index always consumes CPU.
func Index(t *Thread, coll Value, k Value) (Value, *Error) {
	for i := 0; i < maxIndexChainLength; i++ {
		t.RequireCPU(1)
		if tbl, ok := coll.TryTable(); ok {
			if val := RawGet(tbl, k); !val.IsNil() {
				return val, nil
			}
		}
		metaIdx := t.metaGetS(coll, "__index")
		if metaIdx.IsNil() {
			return NilValue, nil
		}
		if _, ok := metaIdx.TryTable(); ok {
			coll = metaIdx
		} else {
			res := NewTerminationWith(t.CurrentCont(), 1, false)
			if err := Call(t, metaIdx, []Value{coll, k}, res); err != nil {
				return NilValue, err
			}
			return res.Get(0), nil
		}
	}
	return NilValue, NewErrorF("'__index' chain too long; possible loop")
}

// SetIndex sets the item in a collection for the given key, using the
// '__newindex' metamethod if appropriate.  SetIndex always consumes CPU if it
// doesn't return an error.
func SetIndex(t *Thread, coll Value, idx Value, val Value) *Error {
	if idx.IsNil() {
		return NewErrorS("index is nil")
	}
	for i := 0; i < maxIndexChainLength; i++ {
		t.RequireCPU(1)
		tbl, isTable := coll.TryTable()
		if isTable && tbl.Reset(idx, val) {
			return nil
		}
		metaNewIndex := t.metaGetS(coll, "__newindex")
		if metaNewIndex.IsNil() {
			if isTable {
				if err := t.SetTableCheck(tbl, idx, val); err != nil {
					return NewErrorE(err)
				}
			}
			return nil
		}
		if _, ok := metaNewIndex.TryTable(); ok {
			coll = metaNewIndex
		} else {
			return Call(t, metaNewIndex, []Value{coll, idx, val}, NewTermination(t.CurrentCont(), nil, nil))
		}
	}
	return NewErrorF("'__newindex' chain too long; possible loop")
}

// Truth returns true if v is neither nil nor a false boolean.
func Truth(v Value) bool {
	if v.IsNil() {
		return false
	}
	b, ok := v.TryBool()
	return !ok || b
}

// Metacall calls the metamethod called method on obj with the given arguments
// args, pushing the result to the continuation next.
func Metacall(t *Thread, obj Value, method string, args []Value, next Cont) (*Error, bool) {
	if f := t.metaGetS(obj, method); !f.IsNil() {
		return Call(t, f, args, next), true
	}
	return nil, false
}

// Continue tries to continue the value f or else use its '__call'
// metamethod and returns the continuations that needs to be run to get the
// results.
func Continue(t *Thread, f Value, next Cont) (Cont, *Error) {
	callable, ok := f.TryCallable()
	if ok {
		return callable.Continuation(t.Runtime, next), nil
	}
	cont, err, ok := metacont(t, f, "__call", next)
	if !ok {
		return nil, NewErrorF("attempt to call a %s value", f.TypeName())
	}
	if cont != nil {
		t.Push1(cont, f)
	}
	return cont, err
}

// Call calls f with arguments args, pushing the results on next.  It may use
// the metamethod '__call' if f is not callable.
func Call(t *Thread, f Value, args []Value, next Cont) *Error {
	if f.IsNil() {
		return NewErrorS("attempt to call a nil value")
	}
	callable, ok := f.TryCallable()
	if ok {
		return t.call(callable, args, next)
	}
	err, ok := Metacall(t, f, "__call", append([]Value{f}, args...), next)
	if ok {
		return err
	}
	return NewErrorF("attempt to call a %s value", f.TypeName())
}

// Call1 is a convenience method that calls f with arguments args and returns
// exactly one value.
func Call1(t *Thread, f Value, args ...Value) (Value, *Error) {
	term := NewTerminationWith(t.CurrentCont(), 1, false)
	if err := Call(t, f, args, term); err != nil {
		return NilValue, err
	}
	return term.Get(0), nil
}

// Concat returns x .. y, possibly calling the '__concat' metamethod.
func Concat(t *Thread, x, y Value) (Value, *Error) {
	var sx, sy string
	var okx, oky bool
	if sx, okx = x.ToString(); okx {
		if sy, oky = y.ToString(); oky {
			t.RequireBytes(len(sx) + len(sy))
			return StringValue(sx + sy), nil
		}
	}
	res, err, ok := metabin(t, "__concat", x, y)
	if ok {
		return res, err
	}
	return NilValue, concatError(x, y, okx, oky)
}

func concatError(x, y Value, okx, oky bool) *Error {
	var wrongVal Value
	switch {
	case oky:
		wrongVal = x
	case okx:
		wrongVal = y
	default:
		return NewErrorF("attempt to concatenate a %s value with a %s value", x.TypeName(), y.TypeName())
	}
	return NewErrorF("attempt to concatenate a %s value", wrongVal.TypeName())
}

// IntLen returns the length of v as an int64, possibly calling the '__len'
// metamethod.  This is an optimization of Len for an integer output.
func IntLen(t *Thread, v Value) (int64, *Error) {
	if s, ok := v.TryString(); ok {
		return int64(len(s)), nil
	}
	res := NewTerminationWith(t.CurrentCont(), 1, false)
	err, ok := Metacall(t, v, "__len", []Value{v}, res)
	if ok {
		if err != nil {
			return 0, err
		}
		l, ok := ToInt(res.Get(0))
		if !ok {
			err = NewErrorS("len should return an integer")
		}
		return l, err
	}
	if tbl, ok := v.TryTable(); ok {
		return tbl.Len(), nil
	}
	return 0, lenError(v)
}

// Len returns the length of v, possibly calling the '__len' metamethod.
func Len(t *Thread, v Value) (Value, *Error) {
	if s, ok := v.TryString(); ok {
		return IntValue(int64(len(s))), nil
	}
	res := NewTerminationWith(t.CurrentCont(), 1, false)
	err, ok := Metacall(t, v, "__len", []Value{v}, res)
	if ok {
		if err != nil {
			return NilValue, err
		}
		return res.Get(0), nil
	}
	if tbl, ok := v.TryTable(); ok {
		return IntValue(tbl.Len()), nil
	}
	return NilValue, lenError(v)
}

func lenError(x Value) *Error {
	return NewErrorF("attempt to get length of a %s value", x.TypeName())
}

// SetEnv sets the item in the table t for a string key.  Useful when writing
// libraries
func (r *Runtime) SetEnv(t *Table, name string, v Value) {
	r.SetTable(t, StringValue(name), v)
}

// SetEnvGoFunc sets the item in the table t for a string key to be a GoFunction
// defined by f.  Useful when writing libraries
func (r *Runtime) SetEnvGoFunc(t *Table, name string, f func(*Thread, *GoCont) (Cont, *Error), nArgs int, hasEtc bool) *GoFunction {
	gof := &GoFunction{
		f:      f,
		name:   name,
		nArgs:  nArgs,
		hasEtc: hasEtc,
	}
	r.SetTable(t, StringValue(name), FunctionValue(gof))
	return gof
}

// ParseLuaChunk parses a string as a Lua statement and returns the AST.
func (r *Runtime) ParseLuaChunk(name string, source []byte) (stat *ast.BlockStat, statSize uint64, err error) {
	s := scanner.New(name, source)

	// Account for CPU and memory used to make the AST.  This is an estimate,
	// but statSize is proportional to the size of the source.
	statSize = uint64(len(source))
	r.LinearRequire(4, uint64(len(source))) // 4 is a factor pulled out of thin air

	stat = new(ast.BlockStat)
	*stat, err = parsing.ParseChunk(s)
	if err != nil {
		r.ReleaseMem(statSize)
		parseErr, ok := err.(parsing.Error)
		if !ok {
			return nil, 0, err
		}
		return nil, 0, NewSyntaxError(name, parseErr)
	}
	return
}

// CompileLuaChunk parses and compiles the source as a Lua Chunk and returns the
// compile code Unit.
func (r *Runtime) CompileLuaChunk(name string, source []byte) (*code.Unit, uint64, error) {
	stat, statSize, err := r.ParseLuaChunk(name, source)
	if err != nil {
		return nil, 0, err
	}

	// In any event the AST goes out of scope when leaving this function
	defer func() { r.ReleaseMem(statSize) }()

	// Account for CPU and memory needed to compile the AST to IR.  This is an
	// estimate, but constsSize is proportional to the size of the AST.
	constsSize := statSize
	r.LinearRequire(4, constsSize) // 4 is a factor pulled out of thin air

	// The IR consts go out of scope when we leave the function
	defer r.ReleaseMem(constsSize)

	// Compile ast to ir
	kidx, constants, err := astcomp.CompileLuaChunk(name, *stat)

	// We no longer need the AST (whether that succeeded or not)
	r.ReleaseMem(statSize)

	if err != nil {
		return nil, 0, fmt.Errorf("%s: %s", name, err)
	}

	statSize = 0 // So that the deferred function above doesn't release the memory again.

	// "Optimise" the ir code
	constants = ir.FoldConstants(constants, ir.DefaultFold)

	// Set up the IR to code compiler
	kc := ircomp.NewConstantCompiler(constants, code.NewBuilder(name))
	kc.QueueConstant(kidx)

	// Account for CPU and memory needed to compile IR to a code unit.  This is
	// an estimate, but unitSize is proportional to the size of the IR consts.
	unitSize := constsSize
	r.LinearRequire(4, unitSize) // 4 is a factor pulled out of thin air

	// Compile IR to code
	unit := kc.CompileQueue()

	// We no longer need the constants
	return unit, unitSize, nil
}

// CompileAndLoadLuaChunk parses, compiles and loads a Lua chunk from source and
// returns the closure that runs the chunk in the given global environment.
func (r *Runtime) CompileAndLoadLuaChunk(name string, source []byte, env Value) (*Closure, error) {
	unit, unitSize, err := r.CompileLuaChunk(name, source)
	defer r.ReleaseMem(unitSize)
	if err != nil {
		return nil, err
	}
	return r.LoadLuaUnit(unit, env), nil
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
	res := NewTerminationWith(t.CurrentCont(), 1, false)
	err, ok := Metacall(t, x, f, xy, res)
	if !ok {
		err, ok = Metacall(t, y, f, xy, res)
	}
	if ok {
		return res.Get(0), err, true
	}
	return NilValue, nil, false
}

func metaun(t *Thread, f string, x Value) (Value, *Error, bool) {
	res := NewTerminationWith(t.CurrentCont(), 1, false)
	err, ok := Metacall(t, x, f, []Value{x}, res)
	if ok {
		return res.Get(0), err, true
	}
	return NilValue, nil, false
}
