package runtime

import (
	"errors"
	"fmt"

	"github.com/arnodel/golua/code"
)

// The handlers in this file run opcodes that are rare at run time.  They are
// kept out of RunInThread so that the run loop holds only what runs often:
// the loop's speed depends on where its code lands, and every handler inside
// it moves the others (benchmarks/layout_ab.sh measures that).  Each takes
// the opcode and works on the frame through c; an error it returns is
// reported by the run loop at the opcode's pc.

// rareConst returns the value of a Type 3 load other than an integer or a
// constant: a two-byte string or a new closure.
func (c *LuaCont) rareConst(t *Thread, opcode code.Opcode) Value {
	n := opcode.GetN()
	switch opcode.GetY() {
	case code.OpStr2:
		return StringValue(string(code.Lit16(n).ToStr2()))
	case code.OpClosureK:
		return FunctionValue(NewClosure(t.Runtime, c.consts[n].AsCode()))
	default:
		panic("Unsupported opcode")
	}
}

// rareUnOp runs a rare Type 4a operation on val, the value of rB.  It returns
// the result for the run loop to store or push, or done if the operation has
// done all its work itself.
func (c *LuaCont) rareUnOp(t *Thread, opcode code.Opcode, val Value) (res Value, done bool, err error) {
	regs, cells := c.registers, c.cells
	dst := opcode.GetA()
	switch opcode.GetUnOp() {
	case code.OpNeg:
		var ok bool
		res, ok = Unm(val)
		if !ok {
			res, err = unaryArithFallback(t, metaUnm, val)
		}
		return res, false, err
	case code.OpBitNot:
		res, err = bnot(t, val)
		return res, false, err
	case code.OpEtcId:
		cont := getReg(regs, cells, dst).AsCont()
		etc, err := expandVarargs(val)
		if err != nil {
			return NilValue, true, err
		}
		cont.PushEtc(t.Runtime, etc)
		return NilValue, true, nil
	case code.OpUpvalue:
		cell := c.getRegCell(opcode.GetB())
		getReg(regs, cells, dst).AsClosure().AddUpvalue(cell)
		return NilValue, true, nil
	case code.OpMkVarargTable:
		tbl := TableValue(t.newVarargTable(val.AsArray()))
		// Overwrite the etc register so ... expansion reads from the table
		// (respecting t.n modifications).
		setReg(regs, cells, opcode.GetB(), tbl)
		return tbl, false, nil
	case code.OpCheckNotDefined:
		// Lua 5.5: check that table[index] is not already defined.  Used by
		// "global x = value" declarations.  rA is the table (_ENV), val the
		// index (the name).
		tbl := getReg(regs, cells, dst).AsTable()
		if tbl != nil && !tbl.Get(val).IsNil() {
			return NilValue, true, fmt.Errorf("global '%s' already defined", val.AsString())
		}
		return NilValue, true, nil
	default:
		panic("unsupported")
	}
}

// rareConstK returns the value of a rare Type 4b load.
func (c *LuaCont) rareConstK(opcode code.Opcode) Value {
	switch code.UnOpK(opcode.GetUnOp()) {
	case code.OpCC:
		return ContValue(c)
	case code.OpStr0:
		return StringValue("")
	case code.OpStr1:
		return StringValue(string(opcode.GetL().ToStr1()))
	default:
		panic("unsupported")
	}
}

// clStack runs a close stack opcode: push rA onto the close stack (F On), or
// truncate the close stack to height D, closing the values removed (F Off).
func (c *LuaCont) clStack(t *Thread, opcode code.Opcode) error {
	if opcode.GetF() {
		v := getReg(c.registers, c.cells, opcode.GetA())
		if Truth(v) && t.metaGet(v, metaClose).IsNil() {
			return errors.New("to be closed value missing a __close metamethod")
		}
		t.closeStack.push(v)
		return nil
	}
	h := c.closeStackBase + int(opcode.GetClStackOffset())
	return t.cleanupCloseStack(c, h, nil)
}

// etcOp runs a Type 6 opcode: rA <- the M-th value of the etc rB (F Off), or
// rA[M+i] <- the i-th value of the etc rB for every i (F On).
func (c *LuaCont) etcOp(t *Thread, opcode code.Opcode) error {
	regs, cells := c.registers, c.cells
	dst := opcode.GetA()
	etc, err := expandVarargs(getReg(regs, cells, opcode.GetB()))
	if err != nil {
		return err
	}
	idx := int(opcode.GetM())
	if opcode.GetF() {
		tbl := getReg(regs, cells, dst).AsTable()
		for i, v := range etc {
			t.SetTable(tbl, IntValue(int64(i+idx)), v)
		}
		return nil
	}
	var val Value
	if idx < len(etc) {
		val = etc[idx]
	}
	setReg(regs, cells, dst, val)
	return nil
}
