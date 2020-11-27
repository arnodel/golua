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

func (e EtcType) ProcessExp(p ExpProcessor) {
	p.ProcessEtcExp(e)
}

func (e EtcType) ProcessTailExp(p TailExpProcessor) {
	p.ProcessEtcTailExp(e)
}

func (e EtcType) HWrite(w HWriter) {
	w.Writef("...")
}
