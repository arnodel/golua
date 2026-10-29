package runtime

import "testing"

func TestSlicePoolClasses(t *testing.T) {
	for _, tc := range []struct{ n, class, size int }{
		{0, 0, 8}, {1, 0, 8}, {8, 0, 8}, {9, 1, 16}, {100, 4, 128}, {1024, 7, 1024}, {1025, -1, 1025},
	} {
		if class, size := poolClass(tc.n); class != tc.class || size != tc.size {
			t.Errorf("poolClass(%d) = %d, %d, want %d, %d", tc.n, class, size, tc.class, tc.size)
		}
	}
}

func TestSlicePoolReuse(t *testing.T) {
	p := mkValuePool(2)
	s := p.get(5)
	if len(s) != 5 || cap(s) != 8 {
		t.Fatalf("get(5) has len %d, cap %d, want 5, 8", len(s), cap(s))
	}
	for i := range s {
		s[i] = IntValue(int64(i + 1))
	}
	p.release(s)
	// A request of the same class gets the same slice back, cleared, and
	// zero beyond its length too.
	s2 := p.get(3)
	if &s2[0] != &s[0] {
		t.Error("get(3) did not reuse the released slice")
	}
	for i, v := range s2[:cap(s2)] {
		if !v.IsNil() {
			t.Errorf("reused slice not cleared at %d: %v", i, v)
		}
	}
	// A request of another class allocates.
	s3 := p.get(9)
	if cap(s3) != 16 {
		t.Errorf("get(9) has cap %d, want 16", cap(s3))
	}
	// The pool keeps at most max slices in the smallest class, and half as
	// many in each class above.
	a, b, c := p.get(1), p.get(1), p.get(1)
	p.release(a)
	p.release(b)
	p.release(c)
	if n := len(p.free[0]); n != 2 {
		t.Errorf("class 0 holds %d slices, want 2", n)
	}
	d, e := p.get(9), p.get(9)
	p.release(d)
	p.release(e)
	if n := len(p.free[1]); n != 1 {
		t.Errorf("class 1 holds %d slices, want 1", n)
	}
	// A slice whose capacity is not a class's, or is too large, is not kept.
	p.release(make([]Value, 3, 5))
	p.release(make([]Value, 0, 2000))
	for class, free := range p.free {
		if want := map[int]int{0: 2, 1: 1}[class]; len(free) != want {
			t.Errorf("class %d holds %d slices, want %d", class, len(free), want)
		}
	}
}

// A Go function's etc arguments come from the pool and go back to it.
func TestGoContEtcPooled(t *testing.T) {
	r := New(nil)
	defer r.Close(nil)
	var seen [][]Value
	f := NewGoFunction(func(th *Thread, c *GoCont) (Cont, error) {
		seen = append(seen, append([]Value(nil), c.Etc()...))
		return c.Next(), nil
	}, "f", 1, true)
	for i := 0; i < 3; i++ {
		if err := Call(r.MainThread(), FunctionValue(f), []Value{IntValue(0), IntValue(int64(i)), StringValue("x")}, NewTerminationWith(nil, 0, false)); err != nil {
			t.Fatal(err)
		}
	}
	for i, etc := range seen {
		if len(etc) != 2 || etc[0] != IntValue(int64(i)) || etc[1] != StringValue("x") {
			t.Errorf("call %d saw etc %v", i, etc)
		}
	}
}
