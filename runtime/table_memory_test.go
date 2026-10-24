//go:build !noquotas

package runtime

import (
	"io"
	"testing"
	"unsafe"
)

func TestNewTableChargesHeader(t *testing.T) {
	r := New(io.Discard)
	defer r.Close(nil)
	r.PushContext(RuntimeContextDef{HardLimits: RuntimeResources{Memory: 1 << 20}})
	defer r.PopContext()
	r.NewTable()
	if got := r.UsedResources().Memory; got != TableHeaderSize {
		t.Fatalf("charged %d bytes, want %d", got, TableHeaderSize)
	}
}

func TestNewTableChecksLimitFirst(t *testing.T) {
	r := New(io.Discard)
	defer r.Close(nil)
	var tbl *Table
	def := RuntimeContextDef{HardLimits: RuntimeResources{Memory: TableHeaderSize - 1}}
	_, err := r.MainThread().CallContext(def, func() error {
		tbl = r.NewTable()
		return nil
	})
	if err == nil {
		t.Fatal("context was not killed")
	}
	if tbl != nil {
		t.Fatal("table created although the limit was exceeded")
	}
}

func TestNewVarargTableChargesHeaderAndN(t *testing.T) {
	values := []Value{IntValue(1), IntValue(2), IntValue(3)}
	// The slice is borrowed from the varargs, so the new allocations are the
	// table, its array part and the storage for n.
	want := TableHeaderSize + uint64(unsafe.Sizeof(array{})) + NewTableFromSlice(values).Set(varargN, IntValue(3))
	r := New(io.Discard)
	defer r.Close(nil)
	r.PushContext(RuntimeContextDef{HardLimits: RuntimeResources{Memory: 1 << 20}})
	defer r.PopContext()
	r.newVarargTable(values)
	if got := r.UsedResources().Memory; got != want {
		t.Fatalf("charged %d bytes, want %d", got, want)
	}
}
