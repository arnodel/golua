package ast

import "github.com/arnodel/golua/token"

// A Location is a span between two token in the source code.
type Location struct {
	start *token.Pos
	end   *token.Pos
}

var _ Locator = Location{}

// StartPos returns the start position of the location.
func (l Location) StartPos() *token.Pos {
	return l.start
}

// EndPos returns the end position of the location.
func (l Location) EndPos() *token.Pos {
	return l.end
}

// Locate returns the receiver.
func (l Location) Locate() Location {
	return l
}

// LocFromToken returns a location that starts and ends at the token's position.
func LocFromToken(tok *token.Token) Location {
	if tok == nil || tok.Pos.Offset < 0 {
		return Location{}
	}
	pos := tok.Pos
	return Location{&pos, &pos}
}

// LocFromTokens returns a location that starts at t1 and ends at t2 (t1 and t2
// may be nil).
func LocFromTokens(t1, t2 *token.Token) Location {
	var p1, p2 *token.Pos
	if t1 != nil && t1.Pos.Offset >= 0 {
		p1 = new(token.Pos)
		*p1 = t1.Pos
	}
	if t2 != nil && t2.Pos.Offset >= 0 {
		p2 = new(token.Pos)
		*p2 = t2.Pos
	}
	return Location{p1, p2}
}

// MergeLocations takes two locators and merges them into one Location, starting
// at the earliest start and ending at the latest end.
func MergeLocations(l1, l2 Locator) Location {
	l := l1.Locate()
	ll := l2.Locate()
	if ll.start != nil && (l.start == nil || l.start.Offset > ll.start.Offset) {
		l.start = ll.start
	}
	if ll.end != nil && (l.end == nil || l.end.Offset < ll.end.Offset) {
		l.end = ll.end
	}
	return l
}
