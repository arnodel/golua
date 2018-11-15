package runtime

import "io"

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
}

// New returns a new pointer to a Runtime with the given stdout.
func New(stdout io.Writer) *Runtime {
	r := &Runtime{
		globalEnv: NewTable(),
		Stdout:    stdout,
		registry:  NewTable(),
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
	r.registry.Set(k, v)
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
	if v == nil {
		return r.nilMeta
	}
	switch x := v.(type) {
	case String:
		return r.stringMeta
	case Float, Int:
		return r.numberMeta
	case Bool:
		return r.boolMeta
	case *Table:
		return x.Metatable()
	case *UserData:
		return x.Metatable()
	default:
		return nil
	}
}

// SetRawMetatable sets the metatable for value v to meta.
func (r *Runtime) SetRawMetatable(v Value, meta *Table) {
	if v == nil {
		r.nilMeta = meta
	}
	switch x := v.(type) {
	case String:
		r.stringMeta = meta
	case Float, Int:
		r.numberMeta = meta
	case Bool:
		r.boolMeta = meta
	case *Table:
		x.SetMetatable(meta)
	case *UserData:
		x.SetMetatable(meta)
	default:
		// Shoul there be an error here?
	}
}

// Metatable returns the metatalbe of v (looking for '__metatable' in the raw
// metatable).
func (r *Runtime) Metatable(v Value) Value {
	meta := r.RawMetatable(v)
	if meta == nil {
		return nil
	}
	metam := RawGet(meta, String("__metatable"))
	if metam != nil {
		return metam
	}
	return meta
}

func (r *Runtime) metaGetS(v Value, k string) Value {
	meta := r.RawMetatable(v)
	return RawGet(meta, String(k))
}
