//go:build !noquotas

package runtime

import (
	"io"
	"testing"
)

func TestLuaTableHeaderChargedOnce(t *testing.T) {
	r := New(io.Discard)
	defer r.Close(nil)
	th := r.MainThread()
	chunk, err := r.CompileAndLoadLuaChunk("table-memory", []byte("return {}"), TableValue(r.GlobalEnv()))
	if err != nil {
		t.Fatal(err)
	}
	term := NewTerminationWith(nil, 1, false)
	// Allocate and release the continuation in the same quota context.
	ctx, err := th.CallContext(RuntimeContextDef{HardLimits: RuntimeResources{Memory: 1 << 20}}, func() error {
		return Call(th, FunctionValue(chunk), nil, term)
	})
	if err != nil {
		t.Fatalf("table opcode failed: %v", err)
	}
	if got := ctx.UsedResources().Memory; got != TableHeaderSize {
		t.Fatalf("charged %d bytes after the call, want one header %d", got, TableHeaderSize)
	}
	if _, ok := term.Get(0).TryTable(); !ok {
		t.Fatal("table opcode did not return a table")
	}
}
