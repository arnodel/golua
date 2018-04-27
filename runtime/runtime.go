package runtime

import "io"

type Runtime struct {
	globalEnv  *Table
	stringMeta *Table
	numberMeta *Table
	boolMeta   *Table
	Stdout     io.Writer
	mainThread *Thread
}

func New(stdout io.Writer) *Runtime {
	r := &Runtime{
		globalEnv: NewTable(),
		Stdout:    stdout,
	}
	mainThread := NewThread(r)
	mainThread.status = ThreadRunning
	r.mainThread = mainThread
	return r
}

func (r *Runtime) GlobalEnv() *Table {
	return r.globalEnv
}

func (r *Runtime) MainThread() *Thread {
	return r.mainThread
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
		// case userdata:
	default:
		return nil
	}
}

func (r *Runtime) Metatable(v Value) *Table {
	meta := r.RawMetatable(v)
	metam := RawGet(meta, "__metatable")
	if metam != nil {
		// We assume a metatable must be a table...
		return metam.(*Table)
	}
	return meta
}

func (r *Runtime) MetaGet(v Value, k Value) Value {
	return RawGet(r.Metatable(v), k)
}

func (r *Runtime) MetaGetS(v Value, k string) Value {
	return RawGet(r.Metatable(v), String(k))
}
