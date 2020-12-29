package runtime

import "testing"

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
