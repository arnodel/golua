package ast

import (
	"github.com/arnodel/golua/ops"
)

// A BinOp is a expression node that represents any binary operator.  The right
// hand side can be a list of operations of the same type (OpType).  All
// operations of the same type have the same precedence and can be merged into a
// list.
type BinOp struct {
	Location
	Left   ExpNode
	OpType ops.Op
	Right  []Operation
}

var _ ExpNode = BinOp{}

// An Operation is a pair (operator, operand) which can be applied to an
// expression node (e.g. "+ 2" or "/ 5").
type Operation struct {
	Op      ops.Op
	Operand ExpNode
}

// NewBinOp creates a new BinOp from the given arguments.  If the left node is
// already a BinOp node with the same operator type, a new node based on that
// will be created.
func NewBinOp(left ExpNode, op ops.Op, right ExpNode) *BinOp {
	loc := MergeLocations(left, right)
	leftOp, ok := left.(*BinOp)
	opType := op.Type()
	if ok && leftOp.OpType == opType {
		return &BinOp{
			Location: loc,
			Left:     leftOp.Left,
			OpType:   opType,
			Right:    append(leftOp.Right, Operation{op, right}),
		}
	}
	return &BinOp{
		Location: loc,
		Left:     left.(ExpNode),
		OpType:   opType,
		Right:    []Operation{{op, right}},
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
