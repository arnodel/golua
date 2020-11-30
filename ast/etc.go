package ast

import (
	"github.com/arnodel/golua/token"
)

// Etc is the "..." expression node (ellipsis).
type Etc struct {
	Location
}

var _ ExpNode = Etc{}
var _ TailExpNode = Etc{}

// NewEtc returns an Etc instance at the location given by the passed token.
func NewEtc(tok *token.Token) Etc {
	return Etc{Location: LocFromToken(tok)}
}

// ProcessExp uses the given ExpProcessor to process the receiver.
func (e Etc) ProcessExp(p ExpProcessor) {
	p.ProcessEtcExp(e)
}

// ProcessTailExp uses the given TailExpProcessor to process the reveiver.
func (e Etc) ProcessTailExp(p TailExpProcessor) {
	p.ProcessEtcTailExp(e)
}

// HWrite prints a tree representation of the node.
func (e Etc) HWrite(w HWriter) {
	w.Writef("...")
}
