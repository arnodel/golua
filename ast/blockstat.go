package ast

// A BlockStat is a statement node that represents a block of statements,
// optionally ending in a return statement (if Return is not a nil slice - note
// that a bare return is encoded as a non-nil slice of length 0).
type BlockStat struct {
	Location
	Stats  []Stat
	Return []ExpNode
}

var _ Stat = BlockStat{}

// NewBlockStat returns a BlockStat instance conatining the given stats and
// return statement.
func NewBlockStat(stats []Stat, rtn []ExpNode) BlockStat {
	return BlockStat{
		// TODO: set Location
		Stats:  stats,
		Return: rtn,
	}
}

// HWrite prints a tree representation of the node.
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

// ProcessStat uses the given StatProcessor to process the receiver.
func (s BlockStat) ProcessStat(p StatProcessor) {
	p.ProcessBlockStat(s)
}
