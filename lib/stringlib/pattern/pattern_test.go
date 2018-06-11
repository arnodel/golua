package pattern

import (
	"fmt"
	"testing"
)

func sameCaptures(exp, act []Capture) bool {
	if len(exp) != len(act) {
		return false
	}
	for i := range exp {
		if exp[i] != act[i] {
			return false
		}
	}
	return true
}

func TestPattern(t *testing.T) {
	tests := []struct {
		ptn, s   string
		captures []Capture
		invalid  bool
	}{
		{
			ptn:      "a*",
			s:        "aaabbb",
			captures: []Capture{{0, 3}},
		},
		{
			ptn:      "a*aaab",
			s:        "aaaaaaaaabcd",
			captures: []Capture{{0, 10}},
		},
		{
			ptn:      "%l+",
			s:        "xyzABC",
			captures: []Capture{{0, 3}},
		},
		{
			ptn:      "(a+)bb",
			s:        "aaabbb",
			captures: []Capture{{0, 5}, {0, 3}},
		},
		{
			ptn:      "x(%d+(%l+))(zzz)",
			s:        "x123abczzz",
			captures: []Capture{{0, 10}, {1, 7}, {4, 7}, {7, 10}},
		},
		{
			ptn:      "..z",
			s:        "xyxyz",
			captures: []Capture{{2, 5}},
		},
		{
			ptn:      "(..)-%1",
			s:        "ab-ba-ba",
			captures: []Capture{{3, 8}, {3, 5}},
		},
		{
			ptn:      "x%b()y",
			s:        "x(y(y)(y)y)y()y",
			captures: []Capture{{0, 12}},
		},
		{
			ptn:      "[abc]",
			s:        "uvwbzz",
			captures: []Capture{{3, 4}},
		},
		{
			ptn:      "%f[abc].*%f[^abc]",
			s:        "1234baac4321",
			captures: []Capture{{4, 8}},
		},
		{
			ptn:      "%d-123",
			s:        "456123123",
			captures: []Capture{{0, 6}},
		},
		{
			ptn:      "%d*123",
			s:        "456123123",
			captures: []Capture{{0, 9}},
		},
		{
			ptn:      "^abc",
			s:        "123abc",
			captures: nil,
		},
		{
			ptn:      "^a-$",
			s:        "aaaaa",
			captures: []Capture{{0, 5}},
		},
		{
			ptn:     "(xx%1)",
			invalid: true,
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("ptn_%d", i), func(t *testing.T) {
			ptn, err := New(test.ptn)
			if err != nil {
				if test.invalid {
					return
				}
				t.Fatal(err)
			}
			if test.invalid {
				t.Fatal("Expected to be invalid")
			}
			captures := ptn.MatchFromStart(test.s, 0)
			if !sameCaptures(test.captures, captures) {
				t.Error("exp:", test.captures, "act:", captures)
				t.Fail()
			}
		})
	}
}
