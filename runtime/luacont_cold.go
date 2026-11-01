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

// rareBinOp runs a binary operation other than the four commonest arithmetic
// operations and the comparisons.
func rareBinOp(t *Thread, op code.BinOp, x, y Value) (res Value, err error) {
	var ok bool
	switch op {
	case code.OpFloorDiv:
		res, ok, err = Idiv(x, y)
		if !ok {
			res, err = binaryArithFallback(t, metaIdiv, x, y)
		}
	case code.OpMod:
		res, ok, err = Mod(x, y)
		if !ok {
			res, err = binaryArithFallback(t, metaMod, x, y)
		}
	case code.OpPow:
		res, ok = Pow(x, y)
		if !ok {
			res, err = binaryArithFallback(t, metaPow, x, y)
		}
	case code.OpBitAnd:
		res, err = band(t, x, y)
	case code.OpBitOr:
		res, err = bor(t, x, y)
	case code.OpBitXor:
		res, err = bxor(t, x, y)
	case code.OpShiftL:
		res, err = shl(t, x, y)
	case code.OpShiftR:
		res, err = shr(t, x, y)
	case code.OpConcat:
		res, err = Concat(t, x, y)
	default:
		panic("unsupported")
	}
	return res, err
}

// prepFor prepares a numeric for loop (prepfor rA, rB, rC): it checks that
// the start, limit and step are numbers, gives the start and step the same
// numeric type, and sets the start to nil if the loop must not run.
func (c *LuaCont) prepFor(opcode code.Opcode) error {
	startReg, stopReg, stepReg := opcode.GetA(), opcode.GetB(), opcode.GetC()
	start := getReg(c.registers, c.cells, startReg)
	stop := getReg(c.registers, c.cells, stopReg)
	step := getReg(c.registers, c.cells, stepReg)
	start, tstart := ToNumberValue(start)
	stop, tstop := ToNumberValue(stop)
	step, tstep := ToNumberValue(step)
	if tstart == NaN || tstop == NaN || tstep == NaN {
		var (
			role string
			val  Value
		)
		switch {
		case tstart == NaN:
			role, val = "initial value", start
		case tstop == NaN:
			role, val = "limit", stop
		default:
			role, val = "step", step
		}
		return fmt.Errorf("'for' %s: expected number, got %s", role, val.CustomTypeName())
	}
	// The loop is done with integers if start and step are
	// both integers, whatever the limit.
	intLoop := tstart == IsInt && tstep == IsInt
	// Make sure start and step have the same numeric type
	if tstart != tstep {
		// One is a float, one is an int, turn them both to floats
		if tstart == IsInt {
			start = FloatValue(float64(start.AsInt()))
		} else {
			step = FloatValue(float64(step.AsInt()))
		}
	}
	// A 0 step is an error
	if isZero(step) {
		return errors.New("'for' step is zero")
	}
	// If the loop is already finished, startReg is set to nil.
	var done bool
	if intLoop {
		// Make the limit an integer too, so that advancing the
		// loop only deals with integers.
		var ok bool
		stop, ok = forLimit(stop, tstop, step.AsInt())
		done = !ok || forPassedInt(start.AsInt(), stop.AsInt(), step.AsInt())
	} else {
		done = forPassed(start, stop, step)
	}
	if done {
		start = NilValue
	}
	setReg(c.registers, c.cells, startReg, start)
	setReg(c.registers, c.cells, stopReg, stop)
	setReg(c.registers, c.cells, stepReg, step)
	return nil
}

// advForFloat advances a for loop that is not all integers: it returns the
// next value of the loop variable and whether the loop is done.
func advForFloat(start, stop, step Value) (Value, bool) {
	next, _ := Add(start, step)
	return next, forPassed(next, stop, step) || forPassed(start, next, step)
}

// callHooks calls the debug hooks for a call, tail call or return.
func (c *LuaCont) callHooks(t *Thread, contReg code.Reg, isTail bool, next Cont) {
	switch {
	case contReg == code.ValueReg(0):
		_ = t.triggerReturn(t, c)
	case isTail:
		_ = t.triggerTailCall(t, next)
	default:
		_ = t.triggerCall(t, next)
	}
}
