package ast

import (
	"github.com/arnodel/golua/token"
)

type IfStat struct {
	Location
	If      CondStat
	ElseIfs []CondStat
	Else    *BlockStat
}

var _ Stat = IfStat{}

func NewIfStat(endTok *token.Token) IfStat {
	return IfStat{Location: LocFromToken(endTok)}
}

func (s IfStat) AddIf(ifTok *token.Token, cond ExpNode, body BlockStat) IfStat {
	s.Location = MergeLocations(LocFromToken(ifTok), s)
	s.If = CondStat{cond, body}
	return s
}

func (s IfStat) AddElse(endTok *token.Token, body BlockStat) IfStat {
	s.Location = MergeLocations(s, LocFromToken(endTok))
	s.Else = &body
	return s
}

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
