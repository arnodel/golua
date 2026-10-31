package runtime

import (
	"io"
	"testing"
)

// The string-key lookup, which Get and Reset take for a string key, must not
// confuse a string with another value whose scalar has the same bits: a short
// string's scalar is its bytes, an integer can have those bits too.
func TestStringKeyLookup(t *testing.T) {
	short := StringValue("ab")
	twin := IntValue(int64(short.scalar))
	long1 := StringValue("a longer string 1")
	long2 := StringValue("a longer string 2")
	empty := StringValue("")

	tbl := NewTable()
	tbl.Set(short, StringValue("short"))
	tbl.Set(twin, StringValue("twin"))
	tbl.Set(long1, StringValue("long1"))
	tbl.Set(long2, StringValue("long2"))
	tbl.Set(empty, StringValue("empty"))
	tbl.Set(IntValue(1), StringValue("one")) // array part

	for _, c := range []struct {
		name string
		k    Value
		want Value
	}{
		{"short string", short, StringValue("short")},
		{"long string 1", long1, StringValue("long1")},
		{"long string 2", long2, StringValue("long2")},
		{"empty string", empty, StringValue("empty")},
		{"missing short string", StringValue("ac"), NilValue},
		{"missing long string", StringValue("a longer string 3"), NilValue},
	} {
		if got := tbl.Get(c.k); got != c.want {
			t.Errorf("%s: Get = %v, want %v", c.name, got, c.want)
		}
	}
	// The integer twin is untouched by the string's lookups and resets.
	if !tbl.Reset(short, StringValue("short2")) {
		t.Error("Reset of an existing string key returned false")
	}
	if got := tbl.Get(twin); got != StringValue("twin") {
		t.Errorf("twin after Reset = %v, want twin", got)
	}
	if got := tbl.Get(short); got != StringValue("short2") {
		t.Errorf("short after Reset = %v, want short2", got)
	}
	if tbl.Reset(StringValue("zz"), StringValue("x")) {
		t.Error("Reset of a missing key returned true")
	}
	// Resetting to nil removes the key.
	if !tbl.Reset(long1, NilValue) || tbl.Get(long1) != NilValue {
		t.Error("Reset to nil did not remove the key")
	}
}

// Large tables take the hashed probe rather than the small-table scan.
func TestStringKeyLookupLargeTable(t *testing.T) {
	tbl := NewTable()
	keys := make([]Value, 100)
	for i := range keys {
		keys[i] = StringValue("key number " + string(rune('A'+i%26)) + string(rune('a'+i/26)))
		tbl.Set(keys[i], IntValue(int64(i)))
	}
	for i, k := range keys {
		if got := tbl.Get(k); got != IntValue(int64(i)) {
			t.Errorf("%v: Get = %v, want %d", k, got, i)
		}
	}
	if got := tbl.Get(StringValue("key number zz")); got != NilValue {
		t.Errorf("missing key: got %v", got)
	}
}

// Field access, methods through __index, assignment to a field and to nil,
// and __index functions all go through the string-key path.  The bare
// runtime has no setmetatable, so the metatables are set from Go.
func TestStringKeyIndexOpcodes(t *testing.T) {
	r := New(io.Discard)
	class := NewTable()
	class.Set(StringValue("__index"), TableValue(class))
	class.Set(StringValue("get"), runChunk(t, r, `return function(self) return self.n end`))
	o := NewTable()
	o.Set(StringValue("n"), IntValue(1))
	o.SetMetatable(class)
	proxyMeta := NewTable()
	proxyMeta.Set(StringValue("__index"), runChunk(t, r, `return function(_, k) return k .. "!" end`))
	proxy := NewTable()
	proxy.SetMetatable(proxyMeta)

	f := runChunk(t, r, `
		return function(o, proxy)
			o.n = o.n + 1
			o.gone = true
			o.gone = nil
			return o:get() * 10 + (o.gone == nil and 1 or 0), proxy.field
		end
	`)
	term := NewTerminationWith(nil, 2, false)
	if err := Call(r.MainThread(), f, []Value{TableValue(o), TableValue(proxy)}, term); err != nil {
		t.Fatal(err)
	}
	if got := term.Get(0); got != IntValue(21) {
		t.Errorf("got %v, want 21", got)
	}
	if got := term.Get(1); got != StringValue("field!") {
		t.Errorf("proxy.field = %v, want field!", got)
	}
}
