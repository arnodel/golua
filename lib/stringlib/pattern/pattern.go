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

func (p *Pattern) Match(s string) []Capture {
	matcher := patternMatcher{
		Pattern: *p,
		s:       s,
	}
	return matcher.find()
}

type Capture struct {
	start, end int
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
