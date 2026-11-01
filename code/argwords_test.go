package code

import "testing"

func TestArgWords(t *testing.T) {
	regs := []Reg{ValueReg(1), CellReg(2), ValueReg(255), CellReg(0), ValueReg(7)}
	for n := 0; n <= len(regs); n++ {
		clear := make([]bool, n)
		for i := range clear {
			clear[i] = i%2 == 0
		}
		words := ArgWords(regs[:n], clear)
		if len(words) != ArgWordCount(n) {
			t.Errorf("%d registers: %d words, want %d", n, len(words), ArgWordCount(n))
		}
		got := ArgRegs(words, n)
		if len(got) != n {
			t.Fatalf("%d registers: unpacked %d", n, len(got))
		}
		for i := range got {
			if got[i] != regs[i] {
				t.Errorf("%d registers: argument %d is %v, want %v", n, i, got[i], regs[i])
			}
		}
		for j, w := range words {
			if w>>30 != 0 {
				t.Errorf("operand word %08x has type bits set", uint32(w))
			}
			for k, c := range w.ArgClears() {
				if i := 3*j + k; i < n && c != clear[i] {
					t.Errorf("%d registers: clear flag %d is %v, want %v", n, i, c, clear[i])
				}
			}
		}
	}
	if CallArgs(ValueReg(3), 5).GetArgCount() != 5 || TailCallArgs(ValueReg(3), 200).GetArgCount() != 200 {
		t.Error("argument count not encoded")
	}
	if Call(ValueReg(3)).GetArgCount() != 0 {
		t.Error("plain call has arguments")
	}
}

func TestIsCallOf(t *testing.T) {
	r1, r2 := ValueReg(1), ValueReg(2)
	for _, c := range []struct {
		name string
		got  bool
		want bool
	}{
		{"call r1", Call(r1).IsCallOf(r1), true},
		{"tailcall r1", TailCall(r1).IsCallOf(r1), true},
		{"call r1 with arguments", CallArgs(r1, 3).IsCallOf(r1), true},
		{"tailcall r1 with arguments", TailCallArgs(r1, 200).IsCallOf(r1), true},
		{"call r2", CallArgs(r2, 1).IsCallOf(r1), false},
		{"call u1", Call(CellReg(1)).IsCallOf(r1), false},
		{"jump", Jump(1).IsCallOf(Reg{}), false},
		{"push", Push(r1, r2).IsCallOf(r1), false},
	} {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}
}
