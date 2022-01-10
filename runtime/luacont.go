package runtime

import (
	"unsafe"

	"github.com/arnodel/golua/code"
)

// LuaCont is a Lua continuation, made from a closure, values for registers and
// some state.
type LuaCont struct {
	*Closure
	registers     []Value
	cells         []Cell
	pc            int16
	acc           []Value
	running       bool
	borrowedCells bool
	closeStack    []Value // Pending to-be-closed variables
}

var _ Cont = (*LuaCont)(nil)

// NewLuaCont returns a new LuaCont from a closure and next, a continuation to
// push results into.
func NewLuaCont(r *Runtime, clos *Closure, next Cont) *LuaCont {
	if clos.upvalueIndex < len(clos.Upvalues) {
		panic("Closure not ready")
	}
	var cells []Cell
	borrowCells := clos.UpvalueCount == clos.CellCount
	if borrowCells {
		cells = clos.Upvalues
	} else {
		cells = r.cellPool.get(int(clos.CellCount))
		copy(cells, clos.Upvalues)
		r.RequireArrSize(unsafe.Sizeof(Cell{}), int(clos.CellCount))
		for i := clos.UpvalueCount; i < clos.CellCount; i++ {
			cells[i] = NewCell(NilValue)
		}
	}
	r.RequireArrSize(unsafe.Sizeof(Value{}), int(clos.RegCount))
	registers := r.regPool.get(int(clos.RegCount))
	registers[0] = ContValue(next)
	cont := r.luaContPool.get()
	r.RequireSize(unsafe.Sizeof(LuaCont{}))
	*cont = LuaCont{
		Closure:       clos,
		registers:     registers,
		cells:         cells,
		borrowedCells: borrowCells,
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
	c.running = true
	opcodes := c.code
	regs := c.registers
	cells := c.cells
	// fmt.Println("START", c)
RunLoop:
	for {
		t.RequireCPU(1)
		// fmt.Println("PC", pc)
		// fmt.Println(c.DebugInfo().String())

		opcode := opcodes[pc]
		if opcode.HasType1() {
			dst := opcode.GetA()
			x := getReg(regs, cells, opcode.GetB())
			y := getReg(regs, cells, opcode.GetC())
			var res Value
			var err *Error
			switch opcode.GetX() {

			// Arithmetic

			case code.OpAdd:
				res, err = add(t, x, y)
			case code.OpSub:
				res, err = sub(t, x, y)
			case code.OpMul:
				res, err = mul(t, x, y)
			case code.OpDiv:
				res, err = div(t, x, y)
			case code.OpFloorDiv:
				res, err = idiv(t, x, y)
			case code.OpMod:
				res, err = Mod(t, x, y)
			case code.OpPow:
				res, err = pow(t, x, y)

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
			var err *Error
			if opcode.HasType4a() {
				val := getReg(regs, cells, opcode.GetB())
				switch opcode.GetUnOp() {
				case code.OpNeg:
					res, err = unm(t, val)
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
				case code.OpToNumber:
					var tp NumberType
					res, tp = ToNumberValue(val)
					if tp == NaN {
						err = NewErrorS("expected numeric value")
					}
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
				next := getReg(regs, cells, contReg).AsCont()
				// We clear the register containing the continuation to allow
				// garbage collection.  A continuation can only be called once
				// anyway, so that's ok semantically.
				c.clearReg(contReg)
				if t.areFlagsEnabled(HookFlagCall | HookFlagReturn) {
					switch {
					case contReg == code.ValueReg(0):
						_ = t.triggerReturn(t, c)
					case opcode.GetF():
						_ = t.triggerTailCall(t, next)
					default:
						_ = t.triggerCall(t, next)
					}
				}
				if opcode.GetF() {
					// It's a tail call.  There is no error, so nothing will
					// reference c anymore, therefore we are safe to give it to
					// the pool for reuse.
					if err := c.truncateCloseStack(t, 0, nil); err != nil {
						return nil, err
					}
					c.release(t.Runtime)
				}
				return next, nil
			case code.OpClStack:
				if opcode.GetF() {
					// Push to close stack
					v := getReg(regs, cells, opcode.GetA())
					c.closeStack = append(c.closeStack, v)
				} else {
					// Truncate close stack
					h := int(opcode.GetClStackOffset())
					if err := c.truncateCloseStack(t, h, nil); err != nil {
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

func (c *LuaCont) Cleanup(t *Thread, err *Error) *Error {
	return c.truncateCloseStack(t, 0, err)
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

func (c *LuaCont) truncateCloseStack(t *Thread, h int, err *Error) *Error {
	for i := len(c.closeStack) - 1; i >= h; i-- {
		v := c.closeStack[i]
		c.closeStack = c.closeStack[:i]
		if Truth(v) {
			closeErr, ok := Metacall(t, v, "__close", []Value{v, err.Value()}, NewTerminationWith(c, 0, false))
			if !ok {
				return NewErrorS("to be closed variable missing a __close metamethod")
			}
			if closeErr != nil {
				err = closeErr
			}
		}
	}
	return err
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
