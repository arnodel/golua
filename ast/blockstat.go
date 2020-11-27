package ast

type BlockStat struct {
	Location
	Stats  []Stat
	Return []ExpNode
}

var _ Stat = BlockStat{}

func NewBlockStat(stats []Stat, rtn []ExpNode) BlockStat {
	return BlockStat{
		// TODO: set Location
		Stats:  stats,
		Return: rtn,
	}
}

func (s BlockStat) HWrite(w HWriter) {
	w.Writef("block")
	w.Indent()
	for _, stat := range s.Stats {
		w.Next()
		stat.HWrite(w)
	}
	if s.Return != nil {
		w.Next()
		w.Writef("return")
		w.Indent()
		for _, val := range s.Return {
			w.Next()
			val.HWrite(w)
		}
		w.Dedent()
	}
	w.Dedent()
}

func (s BlockStat) ProcessStat(p StatProcessor) {
	p.ProcessBlockStat(s)
}
