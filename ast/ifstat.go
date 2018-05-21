package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

type IfStat struct {
	Location
	ifstat      CondStat
	elseifstats []CondStat
	elsestat    *BlockStat
}

func NewIfStat(endTok *token.Token) IfStat {
	return IfStat{Location: LocFromToken(endTok)}
}

func (s IfStat) AddIf(ifTok *token.Token, cond ExpNode, body BlockStat) (IfStat, error) {
	s.Location = MergeLocations(LocFromToken(ifTok), s)
	s.ifstat = CondStat{cond, body}
	return s, nil
}

func (s IfStat) AddElse(endTok *token.Token, body BlockStat) (IfStat, error) {
	s.Location = MergeLocations(s, LocFromToken(endTok))
	s.elsestat = &body
	return s, nil
}

func (s IfStat) AddElseIf(cond ExpNode, body BlockStat) (IfStat, error) {
	s.elseifstats = append(s.elseifstats, CondStat{cond, body})
	return s, nil
}

func (s IfStat) HWrite(w HWriter) {
	w.Writef("if: ")
	w.Indent()
	s.ifstat.HWrite(w)
	for _, elseifstat := range s.elseifstats {
		w.Next()
		w.Writef("elseif: ")
		elseifstat.HWrite(w)
	}
	if s.elsestat != nil {
		w.Next()
		w.Writef("else:")
		s.elsestat.HWrite(w)
	}
	w.Dedent()
}

func (s IfStat) CompileStat(c *ir.Compiler) {
	endLbl := c.GetNewLabel()
	lbl := c.GetNewLabel()
	s.ifstat.CompileCond(c, lbl)
	for _, s := range s.elseifstats {
		EmitInstr(c, s.cond, ir.Jump{Label: endLbl}) // TODO: better location
		c.EmitLabel(lbl)
		lbl = c.GetNewLabel()
		s.CompileCond(c, lbl)
	}
	if s.elsestat != nil {
		EmitInstr(c, s, ir.Jump{Label: endLbl}) // TODO: better location
		c.EmitLabel(lbl)
		s.elsestat.CompileStat(c)
	} else {
		c.EmitLabel(lbl)
	}
	c.EmitLabel(endLbl)

}
