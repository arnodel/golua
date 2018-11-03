package runtime

import "io"

type Runtime struct {
	globalEnv  *Table
	stringMeta *Table
	numberMeta *Table
	boolMeta   *Table
	Stdout     io.Writer
	mainThread *Thread
	registry   *Table
}

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

func (r *Runtime) GlobalEnv() *Table {
	return r.globalEnv
}

func (r *Runtime) Registry(key Value) Value {
	return r.registry.Get(key)
}

func (r *Runtime) SetRegistry(k, v Value) {
	r.registry.Set(k, v)
}

func (r *Runtime) MainThread() *Thread {
	return r.mainThread
}

func (r *Runtime) SetStringMeta(meta *Table) {
	r.stringMeta = meta
}

func (r *Runtime) RawMetatable(v Value) *Table {
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

func (r *Runtime) MetaGet(v Value, k Value) Value {
	meta := r.RawMetatable(v)
	return RawGet(meta, k)
}

func (r *Runtime) MetaGetS(v Value, k string) Value {
	return r.MetaGet(v, String(k))
}
