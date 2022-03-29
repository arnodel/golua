package runtime

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/arnodel/golua/runtime/internal/weakref"
)

// A Runtime is a Lua runtime.  It contains all the global state of the runtime
// (in particular a reference to the global environment and the main thread).
type Runtime struct {
	globalEnv *Table // The global table

	stringMeta *Table // Metatable for all string values
	numberMeta *Table // Metatable for all numeric values
	boolMeta   *Table // Metatable for all boolan values
	nilMeta    *Table // Metatable for nil

	Stdout     io.Writer // This is useful for testing / repls
	mainThread *Thread   // An initialised Runtimes comes with this thread
	gcThread   *Thread   // Thread for running Lua finalizers
	registry   *Table    // The registry table can store data global to the runtime

	warner Warner // Lua 5.4 introduces a warning system, implemented by this

	// This has an almost empty implementation when the noquotas build tag is
	// set.  It should allow the compiler to compile away almost all runtime
	// context manager methods.
	runtimeContextManager

	// Object pools used to minimise the overhead of Go memory management.

	// Register pools, disabled with the noregpool build tag.
	regPool  valuePool
	argsPool valuePool
	cellPool cellPool

	// Continuation pools, disable with the nocontpool build tag.
	luaContPool luaContPool
	goContPool  goContPool
}

type runtimeOptions struct {
	regPoolSize       uint
	regSetMaxAge      uint
	runtimeContextDef *RuntimeContextDef
}

var defaultRuntimeOptions = runtimeOptions{
	regPoolSize:  10,
	regSetMaxAge: 10,
}

// A RuntimeOption configures the Runtime.
type RuntimeOption func(*runtimeOptions)

// WithRegPoolSize set the size of register pool when creating a new Runtime.
// The default register pool size is 10.
func WithRegPoolSize(sz uint) RuntimeOption {
	return func(rtOpts *runtimeOptions) {
		rtOpts.regPoolSize = sz
	}
}

// WithRegSetMaxAge sets the max age of a register set when creating a new
// Runtime.  The default max age is 10.
func WithRegSetMaxAge(age uint) RuntimeOption {
	return func(rtOpts *runtimeOptions) {
		rtOpts.regSetMaxAge = age
	}
}

func WithRuntimeContext(def RuntimeContextDef) RuntimeOption {
	return func(rtOpts *runtimeOptions) {
		rtOpts.runtimeContextDef = &def
	}
}

// New returns a new pointer to a Runtime with the given stdout.
func New(stdout io.Writer, opts ...RuntimeOption) *Runtime {
	rtOpts := defaultRuntimeOptions
	for _, opt := range opts {
		opt(&rtOpts)
	}
	r := &Runtime{
		globalEnv: NewTable(),
		Stdout:    stdout,
		registry:  NewTable(),
		warner:    NewLogWarner(os.Stderr, "Lua warning: "),
		regPool:   mkValuePool(rtOpts.regPoolSize, rtOpts.regSetMaxAge),
		argsPool:  mkValuePool(rtOpts.regPoolSize, rtOpts.regSetMaxAge),
		cellPool:  mkCellPool(rtOpts.regPoolSize, rtOpts.regSetMaxAge),
	}

	mainThread := NewThread(r)
	mainThread.status = ThreadOK
	r.mainThread = mainThread

	gcThread := NewThread(r)
	gcThread.status = ThreadOK
	r.gcThread = gcThread

	r.runtimeContextManager.initRoot()

	if rtOpts.runtimeContextDef != nil {
		r.PushContext(*rtOpts.runtimeContextDef)
	}

	runtime.SetFinalizer(r, func(r *Runtime) { r.Close(nil) })
	return r
}

// GlobalEnv returns the global environment of the runtime.
func (r *Runtime) GlobalEnv() *Table {
	return r.globalEnv
}

// Registry returns the Value associated with key in the runtime's registry.
func (r *Runtime) Registry(key Value) Value {
	return r.registry.Get(key)
}

// SetRegistry sets the value associated with the key k to v in the registry.
func (r *Runtime) SetRegistry(k, v Value) {
	r.SetTable(r.registry, k, v)
}

// MainThread returns the runtime's main thread.
func (r *Runtime) MainThread() *Thread {
	return r.mainThread
}

// SetStringMeta sets the runtime's string metatable (all strings in a runtime
// have the same metatable).
func (r *Runtime) SetStringMeta(meta *Table) {
	r.stringMeta = meta
}

// SetWarner replaces the current warner (Lua 5.4)
func (r *Runtime) SetWarner(warner Warner) {
	r.warner = warner
}

// Warn emits a warning with the given message (Lua 5.4).  The default warner is
// off to start with.  It can be switch on / off by sending it a message "@on" /
// "@off".
func (r *Runtime) Warn(msgs ...string) {
	if r.warner != nil {
		r.warner.Warn(msgs...)
	}
}

// RawMetatable returns the raw metatable for a value (that is, not looking at
// the metatable's '__metatable' key).
func (r *Runtime) RawMetatable(v Value) *Table {
	if v.IsNil() {
		return r.nilMeta
	}
	switch v.Type() {
	case StringType:
		return r.stringMeta
	case FloatType, IntType:
		return r.numberMeta
	case BoolType:
		return r.boolMeta
	case TableType:
		return v.AsTable().Metatable()
	case UserDataType:
		return v.AsUserData().Metatable()
	default:
		return nil
	}
}

// SetRawMetatable sets the metatable for value v to meta.
func (r *Runtime) SetRawMetatable(v Value, meta *Table) {
	if v.IsNil() {
		r.nilMeta = meta
	}
	switch v.Type() {
	case StringType:
		r.stringMeta = meta
	case FloatType, IntType:
		r.numberMeta = meta
	case BoolType:
		r.boolMeta = meta
	case TableType:
		tbl := v.AsTable()
		tbl.SetMetatable(meta)
		if !RawGet(meta, MetaFieldGcValue).IsNil() {
			r.addFinalizer(tbl, weakref.Finalize)
		}
	case UserDataType:
		udata := v.AsUserData()
		udata.SetMetatable(meta)
		r.addFinalizer(udata, udata.MarkFlags())
	default:
		// Should there be an error here?
	}
}

func (r *Runtime) addFinalizer(ref weakref.Value, flags weakref.MarkFlags) {
	if flags != 0 {
		r.weakRefPool.Mark(ref, flags)
	}
}

func (r *Runtime) runPendingFinalizers() {

	// Running finalizers may panic if we run out of resources
	pendingFinalize := r.weakRefPool.ExtractPendingFinalize()
	if len(pendingFinalize) > 0 {
		r.runFinalizers(pendingFinalize)
	}

	// If we get there, releasing resources should not panic
	pendingRelease := r.weakRefPool.ExtractPendingRelease()
	if len(pendingRelease) > 0 {
		releaseResources(pendingRelease)
	}
}

func (r *Runtime) runFinalizers(refs []weakref.Value) {
	for _, ref := range refs {
		term := NewTerminationWith(nil, 0, false)
		v := AsValue(ref)
		err, _ := Metacall(r.gcThread, v, MetaFieldGcString, []Value{v}, term)
		if err != nil {
			r.Warn(fmt.Sprintf("error in finalizer: %s", err))
		}
	}
}

func (t *Thread) CollectGarbage() {
	if t != t.gcThread {
		runtime.GC()
		t.runPendingFinalizers()
	}
}

func (r *Runtime) Close(err *error) {
	runtime.SetFinalizer(r, nil)
	if r := recover(); r != nil {
		ctErr, ok := r.(ContextTerminationError)
		if !ok {
			panic(r)
		}
		if err != nil && *err == nil {
			*err = ctErr
		}
	}
	defer func() {
		if r.PopContext() != nil {
			r.Close(err)
		} else {
			releaseResources(r.weakRefPool.ExtractAllMarkedRelease())
		}
		if r := recover(); r != nil {
			ctErr, ok := r.(ContextTerminationError)
			if !ok {
				panic(r)
			}
			if err != nil && *err == nil {
				*err = ctErr
			}
		}
	}()
	r.runFinalizers(r.weakRefPool.ExtractAllMarkedFinalize())
	return
}

// Metatable returns the metatalbe of v (looking for '__metatable' in the raw
// metatable).
func (r *Runtime) Metatable(v Value) Value {
	meta := r.RawMetatable(v)
	if meta == nil {
		return NilValue
	}
	metam := RawGet(meta, StringValue("__metatable"))
	if metam != NilValue {
		return metam
	}
	return TableValue(meta)
}

// Set a value in a table, requiring memory if needed, and always consuming >0
// CPU.
func (r *Runtime) SetTable(t *Table, k, v Value) {
	r.RequireCPU(1)
	r.RequireMem(t.Set(k, v))
}

var errTableIndexIsNil = errors.New("table index is nil")
var errTableIndexIsNaN = errors.New("table index is NaN")

// SetTableCheck sets k => v in table T if possible, returning an error if k is
// nil or NaN.
func (r *Runtime) SetTableCheck(t *Table, k, v Value) error {
	if k.IsNil() {
		return errTableIndexIsNil
	}
	if k.IsNaN() {
		return errTableIndexIsNaN
	}
	r.SetTable(t, k, v)
	return nil
}

func (r *Runtime) metaGetS(v Value, k string) Value {
	meta := r.RawMetatable(v)
	return RawGet(meta, StringValue(k))
}
