package runtime

import (
	"math"
	"unsafe"

	"github.com/arnodel/golua/code"
)

// LuaCont is a Lua continuation, made from a closure, values for registers and
// some state.
type LuaCont struct {
	*Closure
	registers      []Value
	cells          []Cell
	pc             int32
	acc            []Value
	running        bool
	borrowedCells  bool
	closeStackBase int
}

var _ Cont = (*LuaCont)(nil)

// NewLuaCont returns a new LuaCont from a closure and next, a continuation to
// push results into.
func NewLuaCont(t *Thread, clos *Closure, next Cont) *LuaCont {
	if clos.upvalueIndex < len(clos.Upvalues) {
		panic("Closure not ready")
	}
	var cells []Cell
	borrowCells := clos.UpvalueCount == clos.CellCount
	if borrowCells {
		cells = clos.Upvalues
	} else {
		cells = t.cellPool.get(int(clos.CellCount))
		copy(cells, clos.Upvalues)
		t.RequireArrSize(unsafe.Sizeof(Cell{}), int(clos.CellCount))
		for i := clos.UpvalueCount; i < clos.CellCount; i++ {
			cells[i] = newCell(NilValue)
		}
	}
	t.RequireArrSize(unsafe.Sizeof(Value{}), int(clos.RegCount))
	registers := t.regPool.get(int(clos.RegCount))
	registers[0] = ContValue(next)
	cont := t.luaContPool.get()
	t.RequireSize(unsafe.Sizeof(LuaCont{}))
	*cont = LuaCont{
		Closure:        clos,
		registers:      registers,
		cells:          cells,
		borrowedCells:  borrowCells,
		closeStackBase: t.closeStack.size(),
	}
	return cont
}

func (c *LuaCont) release(r *Runtime) {
	r.regPool.release(c.registers)
	r.ReleaseArrSize(unsafe.Sizeof(Value{}), int(c.RegCount))
	if !c.borrowedCells {
		r.ReleaseArrSize(unsafe.Sizeof(Cell{}), int(c.CellCount))
		r.cellPool.release(c.cells)
	}
	r.luaContPool.release(c)
	r.ReleaseSize(unsafe.Sizeof(LuaCont{}))
}

// Push implements Cont.Push.
func (c *LuaCont) Push(r *Runtime, val Value) {
	opcode := c.code[c.pc]
	if opcode.HasType0() {
		r.RequireCPU(1)
		dst := opcode.GetA()
		if opcode.GetF() {
			// It's an etc
			r.RequireSize(unsafe.Sizeof(Value{}))
			c.acc = append(c.acc, val)
		} else {
			c.pc++
			setReg(c.registers, c.cells, dst, val)
		}
	}
}

// PushEtc implements Cont.PushEtc.  TODO: optimise.
func (c *LuaCont) PushEtc(r *Runtime, vals []Value) {
	for _, val := range vals {
		c.Push(r, val)
	}
}

// Next implements Cont.Next.
func (c *LuaCont) Next() Cont {
	next, ok := c.registers[0].TryCont()
	if !ok {
		return nil
	}
	return next
}

func (c *LuaCont) Parent() Cont {
	return c.Next()
}

// RunInThread implements Cont.RunInThread.
func (c *LuaCont) RunInThread(t *Thread) (Cont, error) {
	pc := c.pc
	consts := c.consts
	lines := c.lines
	var lastLine int32
	c.running = true
	opcodes := c.code
	regs := c.registers
	cells := c.cells
RunLoop:
	for {
		t.RequireCPU(1)

		if t.DebugHooks.areFlagsEnabled(HookFlagLine) {
			line := lines[pc]
			if line > 0 && line != lastLine {
				lastLine = line
				if err := t.triggerLine(t, c, line); err != nil {
					return nil, err
				}
			}
		}
		opcode := opcodes[pc]
		if opcode.HasType1() {
			dst := opcode.GetA()
			x := getReg(regs, cells, opcode.GetB())
			y := getReg(regs, cells, opcode.GetC())
			var res Value
			var err error
			var ok bool
			switch opcode.GetX() {

			// Arithmetic

			case code.OpAdd:
				res, ok = Add(x, y)
				if !ok {
					res, err = binaryArithFallback(t, metaAdd, x, y)
				}
			case code.OpSub:
				res, ok = Sub(x, y)
				if !ok {
					res, err = binaryArithFallback(t, metaSub, x, y)
				}
			case code.OpMul:
				res, ok = Mul(x, y)
				if !ok {
					res, err = binaryArithFallback(t, metaMul, x, y)
				}
			case code.OpDiv:
				res, ok = Div(x, y)
				if !ok {
					res, err = binaryArithFallback(t, metaDiv, x, y)
				}
			// Comparison

			case code.OpEq:
				var r bool
				r, err = eq(t, x, y)
				res = BoolValue(r)
			case code.OpLt:
				var r bool
				r, err = Lt(t, x, y)
				res = BoolValue(r)
			case code.OpLeq:
				var r bool
				r, err = le(t, x, y)
				res = BoolValue(r)

			default:
				res, err = rareBinOp(t, opcode.GetX(), x, y)
			}
			if err != nil {
				c.pc = pc
				return nil, err
			}
			setReg(regs, cells, dst, res)
			pc++
			continue RunLoop
		}
		switch opcode.TypePfx() {
		case code.Type0Pfx:
			dst := opcode.GetA()
			if opcode.GetF() {
				// It's an etc
				setReg(regs, cells, dst, ArrayValue(c.acc))
			} else {
				setReg(regs, cells, dst, NilValue)
			}
			pc++
			continue RunLoop
		case code.Type2Pfx:
			reg := opcode.GetA()
			coll := getReg(regs, cells, opcode.GetB())
			idx := getReg(regs, cells, opcode.GetC())
			// Common case first: an integer key in the array part of a
			// table, with a non-nil value there.  Anything else goes the
			// general way.
			slot := arraySlot(coll, idx)
			if !opcode.GetF() {
				var val Value
				if slot != nil && !slot.IsNil() {
					t.RequireCPU(1)
					val = *slot
				} else {
					var err error
					val, err = Index(t, coll, idx)
					if err != nil {
						c.pc = pc
						return nil, err
					}
				}
				setReg(regs, cells, reg, val)
			} else {
				val := getReg(regs, cells, reg)
				if slot != nil && !slot.IsNil() && !val.IsNil() {
					t.RequireCPU(1)
					*slot = val
				} else if err := SetIndex(t, coll, idx, val); err != nil {
					c.pc = pc
					return nil, err
				}
			}
			pc++
			continue RunLoop
		case code.Type8Pfx:
			reg := opcode.GetA()
			coll := getReg(regs, cells, opcode.GetB())
			key := consts[opcode.GetK8()]
			if !opcode.GetF() {
				val, err := IndexString(t, coll, key)
				if err != nil {
					c.pc = pc
					return nil, err
				}
				setReg(regs, cells, reg, val)
			} else {
				err := SetIndexString(t, coll, key, getReg(regs, cells, reg))
				if err != nil {
					c.pc = pc
					return nil, err
				}
			}
			pc++
			continue RunLoop
		case code.Type3Pfx:
			n := opcode.GetN()
			var val Value
			switch opcode.GetY() {
			case code.OpInt16:
				val = IntValue(int64(int16(n)))
			case code.OpK:
				val = consts[n]
			default:
				val = c.rareConst(t, opcode)
			}
			dst := opcode.GetA()
			if opcode.GetF() {
				// dst must contain a continuation
				cont := getReg(regs, cells, dst).AsCont()
				cont.Push(t.Runtime, val)
			} else {
				setReg(regs, cells, dst, val)
			}
			pc++
			continue RunLoop
		case code.Type4Pfx:
			dst := opcode.GetA()
			var res Value
			var err error
			if opcode.HasType4a() {
				val := getReg(regs, cells, opcode.GetB())
				switch opcode.GetUnOp() {
				case code.OpLen:
					res, err = Len(t, val)
				case code.OpCont:
					var cont Cont
					cont, err = Continue(t, val, c)
					res = ContValue(cont)
				case code.OpTailCont:
					var cont Cont
					cont, err = Continue(t, val, c.Next())
					res = ContValue(cont)
				case code.OpId:
					res = val
				case code.OpTruth:
					res = BoolValue(Truth(val))
				case code.OpNot:
					res = BoolValue(!Truth(val))
				default:
					var done bool
					res, done, err = c.rareUnOp(t, opcode, val)
					if done && err == nil {
						pc++
						continue RunLoop
					}
				}
			} else {
				// Type 4b
				switch code.UnOpK(opcode.GetUnOp()) {
				case code.OpTable:
					res = TableValue(t.NewTable())
				case code.OpBool:
					res = BoolValue(opcode.GetL().ToBool())
				case code.OpNil:
					res = NilValue
				case code.OpClear:
					// Special case: clear reg
					c.clearReg(dst)
					pc++
					continue RunLoop
				default:
					res = c.rareConstK(opcode)
				}
			}
			if err != nil {
				c.pc = pc
				return nil, err
			}
			if opcode.GetF() {
				getReg(regs, cells, dst).AsCont().Push(t.Runtime, res)
			} else {
				setReg(regs, cells, dst, res)
			}
			pc++
			continue RunLoop
		case code.Type5Pfx:
			switch opcode.GetJ() {
			case code.OpJump:
				pc += int32(opcode.GetOffset())
				continue RunLoop
			case code.OpJumpIf:
				test := Truth(getReg(regs, cells, opcode.GetA()))
				if test == opcode.GetF() {
					pc += int32(opcode.GetOffset())
				} else {
					pc++
				}
				continue RunLoop
			case code.OpCall:
				pc++
				c.pc = pc
				c.acc = nil
				c.running = false
				contReg := opcode.GetA()
				isTail := opcode.GetF() // Can mean tail call or simple return
				next := getReg(regs, cells, contReg).AsCont()

				// We clear the register containing the continuation to allow
				// garbage collection.  A continuation can only be called once
				// anyway, so that's ok semantically.
				c.clearReg(contReg)

				if isTail {
					// As we're leaving this continuation for good, perform all
					// the pending close actions.  It must be done before debug
					// hooks are called.
					if err := t.cleanupCloseStack(c, c.closeStackBase, nil); err != nil {
						return nil, err
					}
				}

				if t.areFlagsEnabled(HookFlagCall | HookFlagReturn) {
					c.callHooks(t, contReg, isTail, next)
				}

				if isTail {
					// It's a tail call.  There is no error, so nothing will
					// reference c anymore, therefore we are safe to give it to
					// the pool for reuse.  It must be done after debug hooks
					// are called because they may use c.
					c.release(t.Runtime)
				}
				return next, nil
			case code.OpClStack:
				if err := c.clStack(t, opcode); err != nil {
					c.pc = pc
					return nil, err
				}
				pc++
				continue RunLoop
			default:
				panic("unsupported")
			}
		case code.Type6Pfx:
			if err := c.etcOp(t, opcode); err != nil {
				c.pc = pc
				return nil, err
			}
			pc++
			continue RunLoop
		case code.Type7Pfx:
			startReg, stopReg, stepReg := opcode.GetA(), opcode.GetB(), opcode.GetC()
			start := getReg(regs, cells, startReg)
			stop := getReg(regs, cells, stopReg)
			step := getReg(regs, cells, stepReg)
			if opcode.GetF() {
				// Advance for loop.  All registers are assumed to contain
				// numeric values because they have been prepared previously.
				// The loop is done if the next value has passed the limit, or
				// if the start has passed the next value, which means the
				// addition wrapped around.
				var (
					next Value
					done bool
				)
				if ifaceType(start.iface) == int64IfaceType && ifaceType(step.iface) == int64IfaceType {
					// An integer loop: preparation made the limit an integer
					// too, so this is all integer arithmetic.
					s, st, lim := start.AsInt(), step.AsInt(), stop.AsInt()
					n := s + st
					done = forPassedInt(n, lim, st) || forPassedInt(s, n, st)
					next = IntValue(n)
				} else {
					next, done = advForFloat(start, stop, step)
				}
				if done {
					next = NilValue
				}
				setReg(regs, cells, startReg, next)
			} else if err := c.prepFor(opcode); err != nil {
				c.pc = pc
				return nil, err
			}
			pc++
			continue RunLoop
		}
	}
	// return nil, errors.New("Invalid PC")
}

// forPassed returns whether v has passed the limit of a for loop with the
// given step: it is beyond the limit in the direction of the step.  The
// values must be numbers.
func forPassed(v, limit, step Value) bool {
	if isPositive(step) {
		return numIsLessThan(limit, v)
	}
	return numIsLessThan(v, limit)
}

// forPassedInt is forPassed for an integer loop.
func forPassedInt(v, limit, step int64) bool {
	if step > 0 {
		return v > limit
	}
	return v < limit
}

// forLimit returns the limit of an integer for loop as an integer.  The loop
// runs while its variable has not passed the limit, which may be a float, and
// stops rather than wrap around, so a float limit can be replaced by the
// nearest integer on the loop's side of it: rounded down for a positive step,
// up for a negative one.  A limit beyond the integer range in the direction
// of the loop is replaced by the end of the range, as the loop stops at
// overflow anyway; one beyond the range in the other direction can never be
// reached, so ok is false and the loop does not run.  NaN compares false
// with everything and is treated as below the range, which matches the
// reference interpreter.
func forLimit(limit Value, tlimit NumberType, step int64) (Value, bool) {
	if tlimit == IsInt {
		return limit, true
	}
	f := limit.AsFloat()
	if step > 0 {
		f = math.Floor(f)
	} else {
		f = math.Ceil(f)
	}
	if f >= -1<<63 && f < 1<<63 {
		return IntValue(int64(f)), true
	}
	if f > 0 {
		if step < 0 {
			return limit, false
		}
		return IntValue(math.MaxInt64), true
	}
	if step > 0 {
		return limit, false
	}
	return IntValue(math.MinInt64), true
}

// DebugInfo implements Cont.DebugInfo.
func (c *LuaCont) DebugInfo() *DebugInfo {
	pc := c.pc
	if !c.running {
		pc--
	}
	var currentLine int32 = -1
	if pc >= 0 && int(pc) < len(c.lines) {
		currentLine = c.lines[pc]
	}
	name := c.name
	if name == "" {
		name = "<lua function>"
	}
	return &DebugInfo{
		Source:      c.source,
		Name:        name,
		CurrentLine: currentLine,
	}
}

func (c *LuaCont) getRegCell(reg code.Reg) Cell {
	if reg.IsCell() {
		return c.cells[reg.Idx()]
	}
	panic("should be a cell")
}

func (c *LuaCont) clearReg(reg code.Reg) {
	if reg.IsCell() {
		c.cells[reg.Idx()] = newCell(NilValue)
	} else {
		c.registers[reg.Idx()] = NilValue
	}
}

func setReg(regs []Value, cells []Cell, reg code.Reg, val Value) {
	idx := reg.Idx()
	if reg.IsCell() {
		cells[idx].set(val)
	} else {
		regs[idx] = val
	}
}

func getReg(regs []Value, cells []Cell, reg code.Reg) Value {
	if reg.IsCell() {
		return *cells[reg.Idx()].ref
	}
	return regs[reg.Idx()]
}
