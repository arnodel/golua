package runtime

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/arnodel/golua/code"
)

// compileMain compiles src and returns the opcodes of its main chunk.
func compileMain(t *testing.T, r *Runtime, src string) []code.Opcode {
	t.Helper()
	clos, err := r.CompileAndLoadLuaChunk("test", []byte(src), TableValue(r.GlobalEnv()))
	if err != nil {
		t.Fatal(err)
	}
	return clos.Code.code
}

// runChunk compiles and runs src, returning its first result.
func runChunk(t *testing.T, r *Runtime, src string) Value {
	t.Helper()
	clos, err := r.CompileAndLoadLuaChunk("test", []byte(src), TableValue(r.GlobalEnv()))
	if err != nil {
		t.Fatal(err)
	}
	term := NewTerminationWith(nil, 1, false)
	if err := Call(r.MainThread(), FunctionValue(clos), nil, term); err != nil {
		t.Fatal(err)
	}
	return term.Get(0)
}

func countOps(ops []code.Opcode, pfx code.Opcode) int {
	n := 0
	for _, op := range ops {
		if !op.HasType1() && op.TypePfx() == pfx {
			n++
		}
	}
	return n
}

// Field access, method calls and globals compile to the constant key opcode,
// with no opcode to load the key.
func TestConstantKeyOpcodes(t *testing.T) {
	r := New(io.Discard)
	ops := compileMain(t, r, `
		local o = {x = 1}
		o.y = o.x
		o:m()
		print(o)
	`)
	// {x = 1}, o.x, o.y =, :m, print, _ENV lookups: all constant keys.
	if n := countOps(ops, code.Type8Pfx); n != 5 {
		t.Errorf("got %d constant key opcodes, want 5", n)
	}
	if n := countOps(ops, code.Type2Pfx); n != 0 {
		t.Errorf("got %d register key opcodes, want 0", n)
	}
	// Only the closure of the chunk is loaded from the constant table.
	for _, op := range ops {
		if !op.HasType1() && op.TypePfx() == code.Type3Pfx && op.GetY() == code.OpK {
			t.Errorf("unexpected constant load %s", op.Disassemble(nil, 0))
		}
	}
}

// A chunk with more than 256 constants falls back to loading the key into a
// register, including when the table and the result share a register, and
// when a scratch register is needed for a store.
func TestConstantKeyFallback(t *testing.T) {
	var b strings.Builder
	b.WriteString("local t = {}\n")
	for i := 0; i < 300; i++ {
		fmt.Fprintf(&b, "t.k%d = 'v%d'\n", i, i)
	}
	b.WriteString(`
		local o = {}
		o.y = 2
		local s = {rep = function(a, n) return a .. a end}
		return o.y .. s.rep("a", 2) .. t.k299
	`)
	r := New(io.Discard)
	ops := compileMain(t, r, b.String())
	if n := countOps(ops, code.Type2Pfx); n == 0 {
		t.Error("expected some register key opcodes from the fallback")
	}
	if got := runChunk(t, r, b.String()); got != StringValue("2aav299") {
		t.Errorf("got %v, want 2aav299", got)
	}
}

// A function with constant keys survives what string.dump does to it: its
// constants are slimmed down and renumbered (keys first, so they stay within
// 8 bits), it is marshalled, unmarshalled and verified, and runs again.
func TestConstantKeyDump(t *testing.T) {
	r := New(io.Discard)
	f := runChunk(t, r, `return function(o) o.n = o.n + 1; return o:get() end`)
	slim := r.RefactorCodeConsts(f.AsClosure().Code)
	if n := countOps(slim.code, code.Type8Pfx); n != 3 {
		t.Errorf("got %d constant key opcodes after refactoring, want 3", n)
	}
	var buf bytes.Buffer
	if _, err := MarshalConst(&buf, CodeValue(slim), 1<<20); err != nil {
		t.Fatal(err)
	}
	v, _, err := UnmarshalConst(&buf, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	g := NewClosure(r, v.AsCode())

	o := NewTable()
	o.Set(StringValue("n"), IntValue(1))
	o.Set(StringValue("get"), runChunk(t, r, `return function(self) return self.n end`))
	term := NewTerminationWith(nil, 1, false)
	if err := Call(r.MainThread(), FunctionValue(g), []Value{TableValue(o)}, term); err != nil {
		t.Fatal(err)
	}
	if got := term.Get(0); got != IntValue(2) {
		t.Errorf("got %v, want 2", got)
	}
}
