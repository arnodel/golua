package ast

// CondStat is a conditional statement, used in e.g. if statements and while /
// repeat until loops.
type CondStat struct {
	Cond ExpNode
	Body BlockStat
}

func (s CondStat) HWrite(w HWriter) {
	s.Cond.HWrite(w)
	w.Next()
	w.Writef("body: ")
	s.Body.HWrite(w)
}
