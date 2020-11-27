package ast

import "github.com/arnodel/golua/token"

type Location struct {
	start *token.Pos
	end   *token.Pos
}

func (l Location) StartPos() *token.Pos {
	return l.start
}

func (l Location) EndPos() *token.Pos {
	return l.end
}

func (l Location) Locate() Location {
	return l
}

func LocFromToken(tok *token.Token) Location {
	if tok == nil || tok.Pos.Offset < 0 {
		return Location{}
	}
	pos := tok.Pos
	return Location{&pos, &pos}
}

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
