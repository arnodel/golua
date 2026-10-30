package runtime

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/arnodel/golua/code"
)

// A frame with 2 value registers and 1 cell, and no upvalues.
func testCode(ops ...code.Opcode) *Code {
	return &Code{code: ops, RegCount: 2, CellCount: 1}
}

var (
	r0, r1, r2 = code.ValueReg(0), code.ValueReg(1), code.ValueReg(2)
	u0, u1     = code.CellReg(0), code.CellReg(1)
)

func TestVerifyCounts(t *testing.T) {
	for _, c := range []struct {
		name string
		code *Code
		want string
	}{
		{"no register for the continuation", &Code{RegCount: 0}, "register count"},
		{"too many registers", &Code{RegCount: 257}, "register count"},
		{"negative cells", &Code{RegCount: 1, CellCount: -1}, "cell count"},
		{"too many cells", &Code{RegCount: 1, CellCount: 257}, "cell count"},
		{"more upvalues than cells", &Code{RegCount: 1, CellCount: 1, UpvalueCount: 2, UpNames: []string{"a", "b"}}, "upvalue count"},
		{"negative upvalues", &Code{RegCount: 1, UpvalueCount: -1}, "upvalue count"},
		{"names and upvalues disagree", &Code{RegCount: 1, CellCount: 1, UpvalueCount: 1}, "upvalue names"},
	} {
		err := c.code.verify()
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: got %v, want an error about the %s", c.name, err, c.want)
		}
	}
	ok := &Code{RegCount: 256, CellCount: 256, UpvalueCount: 256, UpNames: make([]string, 256)}
	if err := ok.verify(); err != nil {
		t.Errorf("largest frame: %v", err)
	}
}

func TestVerifyOpcodes(t *testing.T) {
	type tc struct {
		name string
		op   code.Opcode
		ok   bool
	}
	cases := []tc{
		// Type1: A, B, C are registers.
		{"type1 in range", code.Combine(code.OpAdd, r1, r0, u0), true},
		{"type1 bad A", code.Combine(code.OpAdd, r2, r0, r0), false},
		{"type1 bad B", code.Combine(code.OpAdd, r0, r2, r0), false},
		{"type1 bad C", code.Combine(code.OpAdd, r0, r0, r2), false},
		{"type1 bad cell", code.Combine(code.OpAdd, r0, u1, r0), false},
		// Type2: A, B, C are registers.
		{"type2 in range", code.LoadLookup(r1, r0, u0), true},
		{"type2 bad C", code.SetIndex(r0, r0, r2), false},
		// Type3: only A is a register; N is a literal.
		{"type3 in range", code.LoadInt16(r1, -1), true},
		{"type3 large literal is not a register", code.LoadConst(r0, code.KIndex(0xffff)), true},
		{"type3 bad A", code.LoadInt16(r2, 0), false},
		// Type4a: A and B are registers; upval needs a cell in B.
		{"type4a in range", code.Transform(code.OpNeg, r1, r0), true},
		{"type4a bad B", code.Transform(code.OpNeg, r0, r2), false},
		{"type4a push", code.Push(r0, r1), true},
		{"upval with a cell", code.Upval(r0, u0), true},
		{"upval with a value register", code.Upval(r0, r1), false},
		{"upval with a cell out of range", code.Upval(r0, u1), false},
		// Type4b: only A is a register; L and Z are literals.
		{"type4b in range", code.LoadBool(r1, true), true},
		{"type4b literal is not a register", code.LoadStr1(r0, []byte{0xff}), true},
		{"type4b bad A", code.LoadNil(r2), false},
		// Type5: A is a register; D is an offset or a height.
		{"call in range", code.Call(r1), true},
		{"call bad A", code.TailCall(r2), false},
		{"jumpif in range", code.JumpIf(0, r1), true},
		{"jumpif bad A", code.JumpIf(0, r2), false},
		{"cltrunc height is not a register", code.ClTrunc(0xffff), true},
		{"clpush bad A", code.ClPush(r2), false},
		// Type6: A and B are registers; M is an index.
		{"type6 in range", code.LoadEtcLookup(r1, r0, 255), true},
		{"type6 bad B", code.FillTable(r0, r2, 0), false},
		// Type7: A, B, C are registers.
		{"type7 in range", code.PrepForLoop(r0, r1, u0), true},
		{"type7 bad C", code.AdvForLoop(r0, r0, r2), false},
		// Type0: only A is a register.
		{"type0 in range", code.Receive(r1), true},
		{"type0 bad A", code.ReceiveEtc(r2), false},
		// The unused prefix.
		{"unknown type", code.Opcode(1 << 28), false},
	}
	for _, c := range cases {
		err := testCode(c.op).verify()
		if c.ok && err != nil {
			t.Errorf("%s: unexpected error %v", c.name, err)
		}
		if !c.ok && err == nil {
			t.Errorf("%s: expected an error", c.name)
		}
	}
}

func TestVerifyJumps(t *testing.T) {
	// Three opcodes; jumps are relative to the jump itself.
	for _, c := range []struct {
		name   string
		offset code.Offset
		ok     bool
	}{
		{"to the first opcode", -1, true},
		{"to itself", 0, true},
		{"to the last opcode", 1, true},
		{"past the end", 2, false},
		{"before the start", -2, false},
	} {
		c1 := testCode(code.LoadNil(r1), code.Jump(c.offset), code.Call(r0))
		err := c1.verify()
		if c.ok && err != nil {
			t.Errorf("jump %s: unexpected error %v", c.name, err)
		}
		if !c.ok && err == nil {
			t.Errorf("jump %s: expected an error", c.name)
		}
	}
}

// Compiled code passes, all the way down to nested functions.
func TestVerifyCompiledCode(t *testing.T) {
	r := New(io.Discard)
	clos, err := r.CompileAndLoadLuaChunk("test", []byte(`
		local x, y = 1, 2
		local function f(...)
			local t = {...}
			for i = 1, #t do x = x + t[i] end
			return function() return x, y end
		end
		return f(3, 4)()
	`), TableValue(r.GlobalEnv()))
	if err != nil {
		t.Fatal(err)
	}
	// All the functions of a chunk share its constant table, and are in it.
	n := 0
	for _, k := range clos.consts {
		if c, ok := k.TryCode(); ok {
			n++
			if err := c.verify(); err != nil {
				t.Errorf("%s: %v", c.name, err)
			}
		}
	}
	if n != 3 {
		t.Errorf("found %d functions, want 3", n)
	}
}

// A binary chunk that names a register outside its frame is rejected when it
// is loaded, rather than when it runs.
func TestVerifyBinaryChunk(t *testing.T) {
	bad := &Code{
		source:   "test",
		name:     "bad",
		code:     []code.Opcode{code.Receive(r0), code.LoadNil(r2), code.Call(r0)},
		lines:    []int32{1, 1, 1},
		RegCount: 2,
	}
	var buf bytes.Buffer
	if _, err := MarshalConst(&buf, CodeValue(bad), 1<<20); err != nil {
		t.Fatal(err)
	}
	_, _, err := UnmarshalConst(&buf, 1<<20)
	if err == nil || !strings.Contains(err.Error(), "r2") {
		t.Errorf("got %v, want an error about register r2", err)
	}
}
