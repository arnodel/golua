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

func BenchmarkAsCont(b *testing.B) {
	v1 := ContValue(new(GoCont))
	v2 := ContValue(new(LuaCont))
	v3 := ContValue(new(Termination))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v1.AsCont()
		_ = v2.AsCont()
		_ = v3.AsCont()
	}
}
