package ast

import (
	"github.com/arnodel/golua/token"
)

// IfStat is a statement node representing an "if ... [elseif] ... [else] ...
// end" statement, where the else clause is optional and there can be 0 or more
// elseif clauses.
type IfStat struct {
	Location
	If      CondStat
	ElseIfs []CondStat
	Else    *BlockStat
}

var _ Stat = IfStat{}

// NewIfStat returns an IfStat with the given condition and then clause.
func NewIfStat(ifTok *token.Token, cond ExpNode, body BlockStat) IfStat {
	return IfStat{
		Location: LocFromToken(ifTok),
		If:       CondStat{cond, body},
	}
}

// WithElse returns an IfStat based on the receiver but with the given else
// clause.
func (s IfStat) WithElse(endTok *token.Token, body BlockStat) IfStat {
	s.Location = MergeLocations(s, LocFromToken(endTok))
	s.Else = &body
	return s
}

// AddElseIf returns an IfStat based on the receiver but adding an extra elseif
// clause.
func (s IfStat) AddElseIf(cond ExpNode, body BlockStat) IfStat {
	s.ElseIfs = append(s.ElseIfs, CondStat{cond, body})
	return s
}

// ProcessStat uses the given StatProcessor to process the receiver.
func (s IfStat) ProcessStat(p StatProcessor) {
	p.ProcessIfStat(s)
}

// HWrite prints a tree representation of the node.
func (s IfStat) HWrite(w HWriter) {
	w.Writef("if: ")
	w.Indent()
	s.If.HWrite(w)
	for _, elseifstat := range s.ElseIfs {
		w.Next()
		w.Writef("elseif: ")
		elseifstat.HWrite(w)
	}
	if s.Else != nil {
		w.Next()
		w.Writef("else:")
		s.Else.HWrite(w)
	}
	w.Dedent()
}
