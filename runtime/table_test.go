package runtime

import "testing"

func TestTable_Remove(t *testing.T) {
	tbl := NewTable()
	for i := 1; i <= 5; i++ {
		tbl.Set(v(i), v(i))
	}
	tbl.Set(v(5), NilValue)
	tbl.Set(v(4), NilValue)
	tlen := tbl.Len()
	if tlen != 3 {
		t.Errorf("Expected table length to be 3, got %d", tlen)
	}
}

func TestTable_FloatKey(t *testing.T) {
	tbl := NewTable()
	for i := 1; i <= 5; i++ {
		tbl.Set(v(i), v(i))
	}
	tbl.Set(v(float64(500)), v("hi"))
	val := tbl.Get(v(500))
	if val != v("hi") {
		t.Errorf(`Expected "hi", got %v`, val)
	}
}

func TestTable_Next(t *testing.T) {
	tbl := NewTable()
	tbl.Set(v(1), v("x"))
	tbl.Set(v(2), v("y"))
	k1, v1, ok1 := tbl.Next(NilValue)
	if !ok1 {
		t.Fatal("Next failed at first step")
	}
	k2, v2, ok2 := tbl.Next(k1)
	if !ok2 {
		t.Fatal("Next failed at second step")
	}
	k3, _, ok3 := tbl.Next(k2)
	if !ok3 {
		t.Error("Expected ok3 to be true")
	}
	if k3 != NilValue {
		t.Errorf("Expected k3 to be nil, got %v", k3)
	}
	if k1 != v(1) {
		k1, v1, k2, v2 = k2, v2, k1, v1
	}
	if !(k1 == v(1) && v1 == v("x") && k2 == v(2) && v2 == v("y")) {
		t.Errorf("Expected (1, x) and (2, y) to be the items, got (%v, %v) and (%v, %v)", k1, v1, k2, v2)
	}
}
