package ast

import (
	"github.com/arnodel/golua/ops"
)

type BinOp struct {
	Location
	Left   ExpNode
	OpType ops.Op
	Right  []Operand
}

var _ ExpNode = BinOp{}

type Operand struct {
	Op      ops.Op
	Operand ExpNode
}

func NewBinOp(left ExpNode, op ops.Op, right ExpNode) *BinOp {
	loc := MergeLocations(left, right)
	leftOp, ok := left.(*BinOp)
	opType := op & 0xFF
	if ok && leftOp.OpType == opType {
		return &BinOp{
			Location: loc,
			Left:     leftOp.Left,
			OpType:   opType,
			Right:    append(leftOp.Right, Operand{op, right}),
		}
	}
	return &BinOp{
		Location: loc,
		Left:     left.(ExpNode),
		OpType:   opType,
		Right:    []Operand{{op, right}},
	}
}

// ProcessExp uses the given ExpProcessor to process the receiver.
func (b BinOp) ProcessExp(p ExpProcessor) {
	p.ProcessBinOpExp(b)
}

// HWrite prints a tree representation of the node.
func (b BinOp) HWrite(w HWriter) {
	w.Writef("binop: %d", b.OpType)
	w.Indent()
	w.Next()
	b.Left.HWrite(w)
	for _, r := range b.Right {
		w.Next()
		w.Writef("op: %s", r.Op)
		w.Next()
		w.Writef("right: ")
		r.Operand.HWrite(w)
	}
	w.Dedent()
}
