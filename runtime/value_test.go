package runtime

import "testing"

func BenchmarkValue(b *testing.B) {
	for n := 0; n < b.N; n++ {
		sv := IntValue(0)
		for i := 0; i < 1000; i++ {
			iv := IntValue(int64(i))
			sv, _ = add(nil, sv, iv)
		}
	}
}
