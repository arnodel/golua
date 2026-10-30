package runtime

import (
	"fmt"
	"strings"
	"testing"
)

// A string of up to 7 bytes has its bytes and length as its scalar, a longer
// one a hash of its bytes with the top byte set to hashedStringTag.
func TestStringValueScalar(t *testing.T) {
	if sc := StringValue("").scalar; sc != 0 {
		t.Errorf("scalar of the empty string is %#x, want 0", sc)
	}
	for _, s := range []string{"a", "abc", "abcdefg"} {
		sc := StringValue(s).scalar
		if sc>>56 != uint64(len(s)) {
			t.Errorf("top byte of the scalar of %q is %#x, want its length %d", s, sc>>56, len(s))
		}
		if sc >= hashedStringTag {
			t.Errorf("scalar of %q is %#x, tagged as hashed", s, sc)
		}
	}
	for _, s := range []string{"abcdefgh", "a long string constant used as a table key", strings.Repeat("x", 1000)} {
		if sc := StringValue(s).scalar; sc>>56 != 0xFF {
			t.Errorf("top byte of the scalar of %.8q... is %#x, want 0xFF", s, sc>>56)
		}
	}
}

func TestStringValueEqualsAndHash(t *testing.T) {
	// Equal strings with different backing arrays are equal and hash the same,
	// whichever layout their scalar has.
	for _, s := range []string{"", "abc", "abcdefg", "abcdefgh", "a long string constant used as a table key"} {
		a, b := StringValue(s), StringValue(strings.Clone(s))
		if !a.Equals(b) {
			t.Errorf("%q made twice is not equal to itself", s)
		}
		if a.Hash() != b.Hash() {
			t.Errorf("%q made twice hashes differently: %#x != %#x", s, a.Hash(), b.Hash())
		}
	}
	// Strings that differ only after their first 7 bytes are told apart.
	for _, pair := range [][2]string{
		{"abcdefg1", "abcdefg2"},
		{"abcdefg", "abcdefgh"},
		{"abcdefgh", "abcdefghi"},
	} {
		a, b := StringValue(pair[0]), StringValue(pair[1])
		if a.Equals(b) {
			t.Errorf("%q equals %q", pair[0], pair[1])
		}
	}
}

// Long string keys built separately from the ones a table was filled with
// find their values.
func TestTableLongStringKeys(t *testing.T) {
	tbl := NewTable()
	key := func(i int) string { return fmt.Sprintf("a long key number %d", i) }
	for i := 0; i < 100; i++ {
		tbl.Set(StringValue(key(i)), IntValue(int64(i)))
	}
	for i := 0; i < 100; i++ {
		if v := tbl.Get(StringValue(strings.Clone(key(i)))); v != IntValue(int64(i)) {
			t.Errorf("tbl[%q] = %v, want %d", key(i), v, i)
		}
	}
	if v := tbl.Get(StringValue("a long key number 100")); !v.IsNil() {
		t.Errorf("missing key found: %v", v)
	}
}
