package runtime

import (
	"unsafe"

	"github.com/arnodel/golua/code"
)

// LuaCont is a Lua continuation, made from a closure, values for registers and
// some state.
type LuaCont struct {
	*Closure
	registers      []Value
	cells          []Cell
	pc             int16
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
			cells[i] = NewCell(NilValue)
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
func (c *LuaCont) RunInThread(t *Thread) (Cont, *Error) {
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
			var err *Error
			var ok bool
			switch opcode.GetX() {

			// Arithmetic

			case code.OpAdd:
				res, ok = Add(x, y)
				if !ok {
					res, err = BinaryArithFallback(t, "__add", x, y)
				}
			case code.OpSub:
				res, ok = Sub(x, y)
				if !ok {
					res, err = BinaryArithFallback(t, "__sub", x, y)
				}
			case code.OpMul:
				res, ok = Mul(x, y)
				if !ok {
					res, err = BinaryArithFallback(t, "__mul", x, y)
				}
			case code.OpDiv:
				res, ok = Div(x, y)
				if !ok {
					res, err = BinaryArithFallback(t, "__div", x, y)
				}
			case code.OpFloorDiv:
				res, ok, err = Idiv(x, y)
				if !ok {
					res, err = BinaryArithFallback(t, "__idiv", x, y)
				}
			case code.OpMod:
				res, ok, err = Mod(x, y)
				if !ok {
					res, err = BinaryArithFallback(t, "__mod", x, y)
				}
			case code.OpPow:
				res, ok = Pow(x, y)
				if !ok {
					res, err = BinaryArithFallback(t, "__pow", x, y)
				}

			// Bitwise

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

			// Concatenation

			case code.OpConcat:
				res, err = Concat(t, x, y)
			default:
				panic("unsupported")
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
			if !opcode.GetF() {
				val, err := Index(t, coll, idx)
				if err != nil {
					c.pc = pc
					return nil, err
				}
				setReg(regs, cells, reg, val)
			} else {
				err := SetIndex(t, coll, idx, getReg(regs, cells, reg))
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
			case code.OpStr2:
				val = StringValue(string(code.Lit16(n).ToStr2()))
			case code.OpK:
				val = consts[n]
			case code.OpClosureK:
				val = FunctionValue(NewClosure(t.Runtime, consts[n].AsCode()))
			default:
				panic("Unsupported opcode")
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
			var ok bool
			var err *Error
			if opcode.HasType4a() {
				val := getReg(regs, cells, opcode.GetB())
				switch opcode.GetUnOp() {
				case code.OpNeg:
					res, ok = Unm(val)
					if !ok {
						res, err = UnaryArithFallback(t, "__unm", val)
					}
				case code.OpBitNot:
					res, err = bnot(t, val)
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
				case code.OpEtcId:
					// We assume it's a push?
					cont := getReg(regs, cells, dst).AsCont()
					cont.PushEtc(t.Runtime, val.AsArray())
					pc++
					continue RunLoop
				case code.OpTruth:
					res = BoolValue(Truth(val))
				case code.OpNot:
					res = BoolValue(!Truth(val))
				case code.OpUpvalue:
					// TODO: wasteful as we already have got getReg
					cell := c.getRegCell(opcode.GetB())
					getReg(regs, cells, dst).AsClosure().AddUpvalue(cell)
					pc++
					continue RunLoop
				default:
					panic("unsupported")
				}
			} else {
				// Type 4b
				switch code.UnOpK(opcode.GetUnOp()) {
				case code.OpCC:
					res = ContValue(c)
				case code.OpTable:
					res = TableValue(NewTable())
				case code.OpStr0:
					res = StringValue("")
				case code.OpStr1:
					res = StringValue(string(opcode.GetL().ToStr1()))
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
					panic("unsupported")
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
				pc += int16(opcode.GetOffset())
				continue RunLoop
			case code.OpJumpIf:
				test := Truth(getReg(regs, cells, opcode.GetA()))
				if test == opcode.GetF() {
					pc += int16(opcode.GetOffset())
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
					switch {
					case contReg == code.ValueReg(0):
						_ = t.triggerReturn(t, c)
					case isTail:
						_ = t.triggerTailCall(t, next)
					default:
						_ = t.triggerCall(t, next)
					}
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
				if opcode.GetF() {
					// Push to close stack
					v := getReg(regs, cells, opcode.GetA())
					if Truth(v) && t.metaGetS(v, "__close").IsNil() {
						c.pc = pc
						return nil, NewErrorS("to be closed value missing a __close metamethod")
					}
					t.closeStack.push(v)
				} else {
					// Truncate close stack
					h := c.closeStackBase + int(opcode.GetClStackOffset())
					if err := t.cleanupCloseStack(c, h, nil); err != nil {
						c.pc = pc
						return nil, err
					}
				}
				pc++
				continue RunLoop
			default:
				panic("unsupported")
			}
		case code.Type6Pfx:
			dst := opcode.GetA()
			etc := getReg(regs, cells, opcode.GetB()).AsArray()
			idx := int(opcode.GetM())
			var val Value
			if idx < len(etc) {
				val = etc[idx]
			}
			if opcode.GetF() {
				tbl := getReg(regs, cells, dst).AsTable()
				for i, v := range etc {
					t.SetTable(tbl, IntValue(int64(i+idx)), v)
				}
			} else {
				setReg(regs, cells, dst, val)
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
				nextStart, _ := Add(start, step)

				// Check if the loop is done.  It can be done if we have gone
				// over the stop value or if there has been overflow /
				// underflow.
				var done bool
				if isPositive(step) {
					done = numIsLessThan(stop, nextStart) || numIsLessThan(nextStart, start)
				} else {
					done = numIsLessThan(nextStart, stop) || numIsLessThan(start, nextStart)
				}
				if done {
					nextStart = NilValue
				}
				setReg(regs, cells, startReg, nextStart)
			} else {
				// Prepare for loop
				start, tstart := ToNumberValue(start)
				stop, tstop := ToNumberValue(stop)
				step, tstep := ToNumberValue(step)
				if tstart == NaN || tstop == NaN || tstep == NaN {
					c.pc = pc
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
					return nil, NewErrorF("'for' %s: expected number, got %s", role, val.CustomTypeName())
				}
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
					c.pc = pc
					return nil, NewErrorS("'for' step is zero")
				}
				// Check the loop is not already finished. If so, startReg is
				// set to nil.
				var done bool
				if isPositive(step) {
					done, _ = isLessThan(stop, start)
				} else {
					done, _ = isLessThan(start, stop)
				}
				if done {
					start = NilValue
				}
				setReg(regs, cells, startReg, start)
				setReg(regs, cells, stopReg, stop)
				setReg(regs, cells, stepReg, step)
			}
			pc++
			continue RunLoop
		}
	}
	// return nil, errors.New("Invalid PC")
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
		c.cells[reg.Idx()] = NewCell(NilValue)
	} else {
		c.registers[reg.Idx()] = NilValue
	}
}

func setReg(regs []Value, cells []Cell, reg code.Reg, val Value) {
	idx := reg.Idx()
	if reg.IsCell() {
		cells[idx].Set(val)
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
