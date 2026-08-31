//go:build go1.24

package runtime

import "testing"

var hashSink uintptr

func BenchmarkValueHashInt(b *testing.B) {
	v := IntValue(0x0123456789abcdef)
	for b.Loop() {
		hashSink ^= v.Hash()
	}
}

func BenchmarkValueHashShortString(b *testing.B) {
	v := StringValue("abcdef")
	for b.Loop() {
		hashSink ^= v.Hash()
	}
}

func BenchmarkValueHashLongString(b *testing.B) {
	v := StringValue("a fairly long string used as a table key")
	for b.Loop() {
		hashSink ^= v.Hash()
	}
}

func BenchmarkValueHashTable(b *testing.B) {
	v := TableValue(NewTable())
	for b.Loop() {
		hashSink ^= v.Hash()
	}
}
