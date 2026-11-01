package code

import "testing"

// The lookahead predicates agree with the constructors.
func TestLookaheadPredicates(t *testing.T) {
	r1, r2, u1 := ValueReg(1), ValueReg(2), CellReg(1)
	for _, c := range []struct {
		name string
		got  bool
		want bool
	}{
		{"push r1, r2", Push(r1, r2).IsPushInto(r1), true},
		{"push r1, u1", Push(r1, u1).IsPushInto(r1), true},
		{"push r2, r1 into r1", Push(r2, r1).IsPushInto(r1), false},
		{"pushetc r1, r2", PushEtc(r1, r2).IsPushInto(r1), false},
		{"move r1, r2", Transform(OpId, r1, r2).IsPushInto(r1), false},
		{"call r1", Call(r1).IsCallOf(r1), true},
		{"tailcall r1", TailCall(r1).IsCallOf(r1), true},
		{"call r2 of r1", Call(r2).IsCallOf(r1), false},
		{"call u1 of r1", Call(u1).IsCallOf(r1), false},
		{"jump", Jump(0).IsCallOf(Reg{}), false},
		{"clr r2", Clear(r2).IsClearValue(), true},
		{"clr u1", Clear(u1).IsClearValue(), false},
		{"nil r2", LoadNil(r2).IsClearValue(), false},
	} {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}
}
