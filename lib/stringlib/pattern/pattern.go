package pattern

import "errors"

type Pattern struct {
	items                  []patternItem
	captureCount           int
	startAnchor, endAnchor bool
}

func New(ptn string) (*Pattern, error) {
	pb := &patternBuilder{ptn: ptn}
	return pb.getPattern()
}

func (p *Pattern) MatchFromStart(s string, init int) []Capture {
	matcher := patternMatcher{
		Pattern: *p,
		s:       s,
		si:      init,
	}
	return matcher.findFromStart()
}

func (p *Pattern) Match(s string, init int) []Capture {
	matcher := patternMatcher{
		Pattern: *p,
		s:       s,
		si:      init,
	}
	return matcher.find()
}

type Capture struct {
	start, end int
}

func (c Capture) Start() int {
	return c.start
}

func (c Capture) End() int {
	return c.end
}

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

var errInvalidPattern = errors.New("Invalid pattern")
