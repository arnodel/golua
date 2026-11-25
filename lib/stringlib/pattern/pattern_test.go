package pattern

import (
	"fmt"
	"math/rand"
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
		{
			ptn:      "$",
			s:        "abc",
			captures: []Capture{{3, 3}},
		},
		{
			ptn:      "%C+",
			s:        "abc123 xyz",
			captures: []Capture{{0, 10}},
		},
		{
			ptn:      "[]%d]+x?",
			s:        "xxx]22x456",
			captures: []Capture{{3, 7}},
		},
		{
			ptn:      "[0-][a-f]+$^",
			s:        "123--cafe$^ll",
			captures: []Capture{{4, 11}},
		},
		{
			ptn:     "((((((((((a))))))))))",
			invalid: true,
		},
		{
			ptn:     "(",
			invalid: true,
		},
		{
			ptn:     "aa)",
			invalid: true,
		},
		{
			ptn:     "aaa%",
			invalid: true,
		},
		{
			ptn:     "%b",
			invalid: true,
		},
		{
			ptn:     "%b<",
			invalid: true,
		},
		{
			ptn:     "%0",
			invalid: true,
		},
		{
			ptn:     "%e",
			invalid: true,
		},
		{
			ptn:      "%[",
			s:        "[",
			captures: []Capture{{0, 1}},
		},
		{
			ptn:     "[%",
			invalid: true,
		},
		{
			ptn:     "[%e]",
			invalid: true,
		},
		{
			ptn:     "[x",
			invalid: true,
		},
		{
			ptn:     "[x-",
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
			captures := ptn.MatchFromStart(test.s, 0, nil)
			if !sameCaptures(test.captures, captures) {
				t.Error("exp:", test.captures, "act:", captures)
				t.Fail()
			}
		})
	}
}

type randomMeter struct{ used, limit uint64 }

type randomMeterLimit struct{}

func (m *randomMeter) RequireCPU(steps uint64) {
	m.used += steps
	if m.used > m.limit {
		panic(randomMeterLimit{})
	}
}

// Patterns and subjects put together at random from pieces of pattern syntax:
// building a pattern and matching with it panic for nothing but the limit of
// the meter.
func TestRandomPatternsDoNotPanic(t *testing.T) {
	pieces := []string{
		"a", "b", ".", "%a", "%d", "%s", "%w", "[ab]", "[^a]", "[a-c]", "[%a_]",
		"*", "+", "-", "?", "(", ")", "()", "%1", "%2", "%9", "%0",
		"%b", "%bab", "%f", "%f[a]", "^", "$", "%", "[", "]", "%%", "%z", "\x00", "\xff",
	}
	subjects := []string{"a", "b", "ab", "ba", " ", "", "1", "_", "\x00", "\xff"}
	rng := rand.New(rand.NewSource(1))
	for n := 0; n < 200000 && !t.Failed(); n++ {
		var ptn, s string
		for k := rng.Intn(8); k >= 0; k-- {
			ptn += pieces[rng.Intn(len(pieces))]
		}
		for k := rng.Intn(12); k > 0; k-- {
			s += subjects[rng.Intn(len(subjects))]
		}
		func() {
			defer func() {
				if r := recover(); r != nil && r != (randomMeterLimit{}) {
					t.Errorf("pattern %q, subject %q: panic: %v", ptn, s, r)
				}
			}()
			p, err := New(ptn)
			if err != nil {
				return
			}
			for init := 0; init <= len(s); init++ {
				p.Match(s, init, &randomMeter{limit: 100000})
				p.MatchFromStart(s, init, &randomMeter{limit: 100000})
			}
		}()
	}
}
