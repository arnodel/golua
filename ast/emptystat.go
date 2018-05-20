package ast

import (
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

type EmptyStat struct {
	Location
}

func NewEmptyStat(tok *token.Token) (EmptyStat, error) {
	return EmptyStat{Location: LocFromToken(tok)}, nil
}

func (s EmptyStat) HWrite(w HWriter) {
	w.Writef("empty stat")
}

func (s EmptyStat) CompileStat(c *ir.Compiler) {
	// Nothing to compile!
}
