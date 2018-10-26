package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

type IfStat struct {
	Location
	If      CondStat
	ElseIfs []CondStat
	Else    *BlockStat
}

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

func (s IfStat) CompileStat(c *ir.Compiler) {
	endLbl := c.GetNewLabel()
	lbl := c.GetNewLabel()
	s.If.CompileCond(c, lbl)
	for _, s := range s.ElseIfs {
		EmitInstr(c, s.Cond, ir.Jump{Label: endLbl}) // TODO: better location
		c.EmitLabel(lbl)
		lbl = c.GetNewLabel()
		s.CompileCond(c, lbl)
	}
	if s.Else != nil {
		EmitInstr(c, s, ir.Jump{Label: endLbl}) // TODO: better location
		c.EmitLabel(lbl)
		s.Else.CompileStat(c)
	} else {
		c.EmitLabel(lbl)
	}
	c.EmitLabel(endLbl)

}
