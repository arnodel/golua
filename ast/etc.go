package ast

import (
	"github.com/arnodel/golua/token"
)

type EtcType struct {
	Location
}

var _ ExpNode = EtcType{}
var _ TailExpNode = EtcType{}

func Etc(tok *token.Token) EtcType {
	return EtcType{Location: LocFromToken(tok)}
}

// ProcessExp uses the given ExpProcessor to process the receiver.
func (e EtcType) ProcessExp(p ExpProcessor) {
	p.ProcessEtcExp(e)
}

func (e EtcType) ProcessTailExp(p TailExpProcessor) {
	p.ProcessEtcTailExp(e)
}

// HWrite prints a tree representation of the node.
func (e EtcType) HWrite(w HWriter) {
	w.Writef("...")
}
