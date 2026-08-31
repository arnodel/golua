package runtime

import (
	"math"
	"math/bits"
	"testing"
)

// The hash table relies on: if a.Equals(b) then a.Hash() == b.Hash().
func TestValueHashConsistentWithEquals(t *testing.T) {
	tbl := NewTable()
	values := []Value{
		NilValue,
		IntValue(0), IntValue(1), IntValue(-1), IntValue(1 << 40),
		FloatValue(0), FloatValue(1.5), FloatValue(-1.5), FloatValue(math.Inf(1)),
		BoolValue(true), BoolValue(false),
		StringValue(""), StringValue("a"), StringValue("abcdefg"),
		StringValue("a slightly longer string used as a key"),
		TableValue(tbl),
	}
	for _, a := range values {
		for _, b := range values {
			if a.Equals(b) && a.Hash() != b.Hash() {
				t.Errorf("Equals(%v, %v) but hashes differ: %#x != %#x", a, b, a.Hash(), b.Hash())
			}
		}
	}
	// Re-constructing an equal value must give the same hash.
	for _, mk := range []func() Value{
		func() Value { return IntValue(1234567) },
		func() Value { return FloatValue(3.14159) },
		func() Value { return StringValue("short") },
		func() Value { return StringValue("a long string constant used as a table key") },
	} {
		if h1, h2 := mk().Hash(), mk().Hash(); h1 != h2 {
			t.Errorf("hash not stable for %v: %#x != %#x", mk(), h1, h2)
		}
	}
}

// The seed must actually influence the result, so that seeding it later (for
// hash-flooding protection) has an effect.
func TestHashScalarSeeded(t *testing.T) {
	for _, x := range []uint64{0, 1, 0xdeadbeef, 1 << 40} {
		if hashScalar(x, 0) == hashScalar(x, 1) {
			t.Errorf("hashScalar(%#x, 0) == hashScalar(%#x, 1)", x, x)
		}
	}
}

// hashScalar should have decent avalanche behaviour: flipping a single input
// bit should change roughly half the output bits on average.
func TestHashScalarAvalanche(t *testing.T) {
	const samples = 4096
	var total, count int
	for i := 0; i < samples; i++ {
		x := uint64(i) * 0x9e3779b97f4a7c15
		h := uint64(hashScalar(x, 0))
		for b := 0; b < 64; b++ {
			h2 := uint64(hashScalar(x^(1<<b), 0))
			total += bits.OnesCount64(h ^ h2)
			count++
		}
	}
	avg := float64(total) / float64(count)
	if avg < 28 || avg > 36 {
		t.Errorf("poor avalanche: average %.2f changed bits (want ~32)", avg)
	}
}

// Low bits (the ones actually used for bucketing) should be well distributed
// for the kinds of keys Lua programs use most: small consecutive integers.
func TestHashScalarBucketing(t *testing.T) {
	const n = 1 << 16
	const buckets = 1 << 8
	var counts [buckets]int
	for i := 0; i < n; i++ {
		counts[hashScalar(uint64(i), 0)&(buckets-1)]++
	}
	const expected = n / buckets
	for b, c := range counts {
		if c < expected/2 || c > expected*3/2 {
			t.Errorf("bucket %d got %d keys, expected ~%d", b, c, expected)
		}
	}
}
