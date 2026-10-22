//go:build !noquotas

package runtime

import (
	"io"
	"testing"
	"unsafe"
)

func TestRuntimeNewTableChargesHeader(t *testing.T) {
	for _, soft := range []bool{false, true} {
		t.Run(map[bool]string{false: "hard", true: "soft"}[soft], func(t *testing.T) {
			r := New(io.Discard)
			defer r.Close(nil)
			def := RuntimeContextDef{HardLimits: RuntimeResources{Memory: 1 << 20}}
			if soft {
				def = RuntimeContextDef{SoftLimits: RuntimeResources{Memory: 1 << 20}}
			}
			r.PushContext(def)
			defer r.PopContext()
			r.NewTable()
			if got := r.UsedResources().Memory; got != TableHeaderSize {
				t.Fatalf("charged %d, want one header %d", got, TableHeaderSize)
			}
		})
	}
}

func TestRuntimeNewTableChecksLimitBeforeAllocation(t *testing.T) {
	r := New(io.Discard)
	defer r.Close(nil)
	var table *Table
	ctx, err := r.MainThread().CallContext(RuntimeContextDef{HardLimits: RuntimeResources{Memory: TableHeaderSize - 1}}, func() error { table = r.NewTable(); return nil })
	if err == nil || table != nil || ctx.UsedResources().Memory != 0 {
		t.Fatalf("limit not enforced before table creation: table=%v, err=%v, used=%v", table, err, ctx.UsedResources())
	}
}

func TestVarargTableChargesOnlyNewStorage(t *testing.T) {
	for _, size := range []int{0, 3} {
		t.Run(map[int]string{0: "empty", 3: "populated"}[size], func(t *testing.T) {
			values := make([]Value, size)
			for i := range values {
				values[i] = IntValue(int64(i + 1))
			}
			raw := NewTableFromSlice(values)
			want := TableHeaderSize + uint64(unsafe.Sizeof(array{})) + raw.Set(varargN, IntValue(int64(size)))
			r := New(io.Discard)
			defer r.Close(nil)
			r.PushContext(RuntimeContextDef{HardLimits: RuntimeResources{Memory: 1 << 20}})
			defer r.PopContext()
			table := r.newVarargTable(values)
			if got := r.UsedResources().Memory; got != want {
				t.Fatalf("charged %d, want new wrappers and n storage %d (not the borrowed slice)", got, want)
			}
			if table.Get(varargN) != IntValue(int64(size)) {
				t.Fatal("vararg count changed")
			}
			if size > 0 {
				table.Set(IntValue(1), IntValue(99))
				if values[0] != IntValue(99) {
					t.Fatal("vararg slice is no longer shared")
				}
			}
		})
	}
}
