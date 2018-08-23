package luatesting

import "testing"

func TestExtractLineCheckers(t *testing.T) {
	src := `
Hello
--> =abc def

--> ~ab
sdfa
asdf
--> ~^ab.*yz$
`
	checkers := ExtractLineCheckers([]byte(src))
	if len(checkers) != 3 {
		t.Fatal("expected 3 checkers")
	}

	checks := []struct {
		i       int
		line    string
		success bool
	}{
		{0, "abc", false},
		{0, "abc def", true},
		{0, "abc def\n", false},
		{1, "abcd", true},
		{1, "xyzab", true},
		{1, "axyb", false},
		{2, "abyz", true},
		{2, "xabyz", false},
		{2, "ab....yz", true},
		{2, "abyz\n", false},
	}
	for i, check := range checks {
		err := checkers[check.i].CheckLine([]byte(check.line))
		if (err != nil && check.success) || (err == nil && !check.success) {
			t.Fatalf("Check #%d failed", i+1)
		}
	}
}
