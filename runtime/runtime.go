package runtime

import (
	"errors"
	"io"
)

// A Runtime is a Lua runtime.  It contains all the global state of the runtime
// (in particular a reference to the global environment and the main thread).
type Runtime struct {
	globalEnv  *Table
	stringMeta *Table
	numberMeta *Table
	boolMeta   *Table
	nilMeta    *Table
	Stdout     io.Writer
	mainThread *Thread
	registry   *Table

	// This has an empty implementation when the noquotas build tag is set.  It
	// should allow the compiler to compile away all quota manager methods.
	quotaManager

	regPool  valuePool
	argsPool valuePool
	cellPool cellPool

	luaContPool luaContPool
	goContPool  goContPool
}

type runtimeOptions struct {
	regPoolSize  uint
	regSetMaxAge uint
}

var defaultRuntimeOptions = runtimeOptions{
	regPoolSize:  10,
	regSetMaxAge: 10,
}

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
		regPool:   mkValuePool(rtOpts.regPoolSize, rtOpts.regSetMaxAge),
		argsPool:  mkValuePool(rtOpts.regPoolSize, rtOpts.regSetMaxAge),
		cellPool:  mkCellPool(rtOpts.regPoolSize, rtOpts.regSetMaxAge),
	}
	mainThread := NewThread(r)
	mainThread.status = ThreadOK
	r.mainThread = mainThread
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
		v.AsTable().SetMetatable(meta)
	case UserDataType:
		v.AsUserData().SetMetatable(meta)
	default:
		// Shoul there be an error here?
	}
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

// Set a value in a table, updating the memory quota.
func (r *Runtime) SetTable(t *Table, k, v Value) {
	r.RequireMem(t.Set(k, v))
}

func (r *Runtime) SetTableCheck(t *Table, k, v Value) error {
	if k.IsNil() {
		return errors.New("table index is nil")
	}
	r.SetTable(t, k, v)
	return nil
}

func (r *Runtime) metaGetS(v Value, k string) Value {
	meta := r.RawMetatable(v)
	return RawGet(meta, StringValue(k))
}

type QuotaExceededError struct {
	message string
}

func (p QuotaExceededError) Error() string {
	return p.message
}
