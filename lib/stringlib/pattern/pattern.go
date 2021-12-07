package pattern

import "errors"

// A Pattern is a data structure able to interpret a Lua "pattern" (see e.g.
// https://www.lua.org/manual/5.3/manual.html#6.4.1).
type Pattern struct {
	items                  []patternItem
	captureCount           int
	startAnchor, endAnchor bool
}

// New returns a new Pattern built from the given string (or an error if it is
// not a valid pattern string).
func New(ptn string) (*Pattern, error) {
	pb := &patternBuilder{ptn: ptn}
	return pb.getPattern()
}

// MatchFromStart returns a slice of Capture instances that match the given
// string, starting from the `init` index.
func (p *Pattern) MatchFromStart(s string, init int, budget uint64) (captures []Capture, used uint64) {
	defer func() {
		if r := recover(); r == budgetConsumed {
			captures = nil
			used = budget + 1
		}
	}()
	matcher := patternMatcher{
		Pattern: *p,
		s:       s,
		si:      init,
		budget:  budget,
	}
	captures = matcher.findFromStart()
	return captures, budget - matcher.budget
}

// Match returns a slice of Capture instances that match the given
// string, starting from the `init` index.
func (p *Pattern) Match(s string, init int, budget uint64) (captures []Capture, used uint64) {
	defer func() {
		if r := recover(); r == budgetConsumed {
			captures = nil
			used = budget + 1
		}
	}()
	matcher := patternMatcher{
		Pattern: *p,
		s:       s,
		si:      init,
		budget:  budget,
	}
	captures = matcher.find()
	return captures, budget - matcher.budget
}

// A Capture represents a matching substring.
type Capture struct {
	start, end int
}

// Start index of the capture.
func (c Capture) Start() int {
	return c.start
}

// End index of the capture.
func (c Capture) End() int {
	return c.end
}

// IsEmpty is trye if the Capture is empty.
func (c Capture) IsEmpty() bool {
	return c.end == -1
}

type patternItemType byte

const (
	ptnOnce patternItemType = iota
	ptnGreedyRepeat
	ptnGreedyRepeatOnce
	ptnRepeat
	ptnOptional
	ptnCapture
	ptnBalanced
	ptnFrontier
	ptnStartCapture
	ptnEndCapture
)

type patternItem struct {
	bytes   byteSet
	ptnType patternItemType
}

var errInvalidPattern = errors.New("malformed pattern")
var errUnfinishedCapture = errors.New("unfinished capture")
var errInvalidPatternCapture = errors.New("invalid pattern capture")
var errPatternTooComplex = errors.New("pattern too complex")
