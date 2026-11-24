package pattern

import (
	"fmt"
	"math/rand"
	"strings"
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

// A stepMeter records the steps that a match charges, and panics when they
// exceed its limit if it has one.
type stepMeter struct {
	charged, limit uint64
	largestCharge  uint64
}

type stepLimitReached struct{}

func (m *stepMeter) RequireCPU(steps uint64) {
	m.charged += steps
	if steps > m.largestCharge {
		m.largestCharge = steps
	}
	if m.limit > 0 && m.charged > m.limit {
		panic(stepLimitReached{})
	}
}

// A match charges its steps as it goes, not all of them when it ends.
func TestMatchChargesStepsInBatches(t *testing.T) {
	// Each of the 2001 places a match can start at goes to the end of the
	// string looking for a b, one pattern item at a time.  That is three
	// steps for each character (".-" is tried and takes it, then "b" is
	// tried), and two at the end of the string, where both are tried.
	s := strings.Repeat("a", 2000)
	ptn, err := New(".-b")
	if err != nil {
		t.Fatal(err)
	}
	meter := new(stepMeter)
	if captures := ptn.Match(s, 0, meter); captures != nil {
		t.Fatal("expected no match")
	}
	const wantSteps = 3*(2000*2001/2) + 2*2001
	if meter.charged != wantSteps {
		t.Errorf("charged %d steps, want %d", meter.charged, wantSteps)
	}
	// An item takes two steps at most here, so a charge is never more than
	// one step over.
	if meter.largestCharge > stepsPerCharge+1 {
		t.Errorf("largest charge is %d steps, want at most %d", meter.largestCharge, stepsPerCharge+1)
	}
}

// One pattern item can take as many steps as the string is long before they
// are charged.
func TestMatchChargesLongItemAfterIt(t *testing.T) {
	s := strings.Repeat("a", 10*stepsPerCharge)
	ptn, err := New("^a*$")
	if err != nil {
		t.Fatal(err)
	}
	meter := new(stepMeter)
	if captures := ptn.Match(s, 0, meter); captures == nil {
		t.Fatal("expected a match")
	}
	// One step to try the item and one for each character.
	if want := uint64(len(s)) + 1; meter.charged != want {
		t.Errorf("charged %d steps, want %d", meter.charged, want)
	}
	if meter.largestCharge != meter.charged {
		t.Errorf("largest charge is %d steps, want all %d in one", meter.largestCharge, meter.charged)
	}
}

// An item that is tried and matches nothing is a step: there can be many of
// them at every position of the string.
func TestMatchChargesItemsThatMatchNothing(t *testing.T) {
	const items, length = 100, 1000
	s := strings.Repeat("a", length)
	ptn, err := New(strings.Repeat("x?", items) + "y")
	if err != nil {
		t.Fatal(err)
	}
	meter := new(stepMeter)
	if captures := ptn.Match(s, 0, meter); captures != nil {
		t.Fatal("expected no match")
	}
	if least := uint64(items * length); meter.charged < least {
		t.Errorf("charged %d steps, want at least %d", meter.charged, least)
	}
}

// Comparing with a capture is a step for each byte compared.
func TestMatchChargesBackReference(t *testing.T) {
	const length = 1000
	s := strings.Repeat("a", length)
	ptn, err := New("^(a*)%1b")
	if err != nil {
		t.Fatal(err)
	}
	meter := new(stepMeter)
	if captures := ptn.MatchFromStart(s, 0, meter); captures != nil {
		t.Fatal("expected no match")
	}
	// The capture is compared at each length that leaves room for it twice,
	// from half the string down to nothing.
	const half = length / 2
	if least := uint64(half * (half + 1) / 2); meter.charged < least {
		t.Errorf("charged %d steps, want at least %d", meter.charged, least)
	}
}

// A meter that panics stops a match soon after its limit, however long the
// match would take.
func TestMeterStopsMatch(t *testing.T) {
	// This pattern takes a number of steps exponential in the length of s.
	s := strings.Repeat("a", 50)
	ptn, err := New(strings.Repeat("a?", 50) + s + "b")
	if err != nil {
		t.Fatal(err)
	}
	meter := &stepMeter{limit: 100000}
	func() {
		defer func() {
			if r := recover(); r != (stepLimitReached{}) {
				t.Errorf("recovered %v, want the panic of the meter", r)
			}
		}()
		ptn.Match(s, 0, meter)
		t.Error("the match should not end")
	}()
	if most := meter.limit + stepsPerCharge + uint64(len(s)); meter.charged > most {
		t.Errorf("charged %d steps, want at most %d", meter.charged, most)
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
