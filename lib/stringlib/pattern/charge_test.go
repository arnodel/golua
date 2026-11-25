//go:build !noquotas
// +build !noquotas

package pattern

import (
	"strings"
	"testing"
)

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
