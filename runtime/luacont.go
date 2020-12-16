package runtime

import (
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
}

// NewLuaCont returns a new LuaCont from a closure and next, a continuation to
// push results into.
func NewLuaCont(clos *Closure, next Cont) *LuaCont {
	if clos.upvalueIndex < len(clos.Upvalues) {
		panic("Closure not ready")
	}
	var cells []Cell
	borrowCells := clos.UpvalueCount == clos.CellCount
	if borrowCells {
		cells = clos.Upvalues
	} else {
		cells = globalRegPool.getCells(int(clos.CellCount))
		copy(cells, clos.Upvalues)
		for i := clos.UpvalueCount; i < clos.CellCount; i++ {
			cells[i] = NewCell(NilValue)
		}
	}
	registers := globalRegPool.getRegs(int(clos.RegCount))
	registers[0] = ContValue(next)
	cont := globalLuaContPool.get()
	*cont = LuaCont{
		Closure:       clos,
		registers:     registers,
		cells:         cells,
		borrowedCells: borrowCells,
	}
	return cont
}

func (c *LuaCont) release() {
	globalRegPool.releaseRegs(c.registers)
	if !c.borrowedCells {
		globalRegPool.releaseCells(c.cells)
	}
	globalLuaContPool.release(c)
}

// Push implements Cont.Push.
func (c *LuaCont) Push(val Value) {
	opcode := c.code[c.pc]
	if opcode.HasType0() {
		dst := opcode.GetA()
		if opcode.GetF() {
			// It's an etc
			c.acc = append(c.acc, val)
		} else {
			c.pc++
			c.setReg(dst, val)
		}
	}
}

// PushEtc implements Cont.PushEtc.  TODO: optimise.
func (c *LuaCont) PushEtc(vals []Value) {
	for _, val := range vals {
		c.Push(val)
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

// RunInThread implements Cont.RunInThread.
func (c *LuaCont) RunInThread(t *Thread) (Cont, *Error) {
	pc := c.pc
	consts := c.consts
	c.running = true
	// fmt.Println("START", c)
RunLoop:
	for {
		// fmt.Println("PC", pc)
		opcode := c.code[pc]
		if opcode.HasType1() {
			dst := opcode.GetA()
			x := c.getReg(opcode.GetB())
			y := c.getReg(opcode.GetC())
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
				return nil, err.AddContext(c)
			}
			c.setReg(dst, res)
			pc++
			continue RunLoop
		}
		switch opcode.TypePfx() {
		case code.Type0Pfx:
			dst := opcode.GetA()
			if opcode.GetF() {
				// It's an etc
				c.setReg(dst, ArrayValue(c.acc))
			} else {
				c.setReg(dst, NilValue)
			}
			pc++
			continue RunLoop
		case code.Type2Pfx:
			reg := opcode.GetA()
			coll := c.getReg(opcode.GetB())
			idx := c.getReg(opcode.GetC())
			if !opcode.GetF() {
				val, err := Index(t, coll, idx)
				if err != nil {
					return nil, err.AddContext(c)
				}
				c.setReg(reg, val)
			} else {
				err := SetIndex(t, coll, idx, c.getReg(reg))
				if err != nil {
					return nil, err.AddContext(c)
				}
			}
			pc++
			continue RunLoop
		case code.Type3Pfx:
			n := opcode.GetN()
			var val Value
			switch opcode.GetY() {
			case code.OpInt16:
				val = IntValue(int64(n))
			case code.OpStr2:
				val = StringValue(string(code.Lit16(n).ToStr2()))
			case code.OpK:
				val = consts[n].Value()
			case code.OpClosureK:
				val = FunctionValue(NewClosure(consts[n].Value().AsCode()))
			default:
				panic("Unsupported opcode")
			}
			dst := opcode.GetA()
			if opcode.GetF() {
				// dst must contain a continuation
				cont := c.getReg(dst).AsCont()
				cont.Push(val)
			} else {
				c.setReg(dst, val)
			}
			pc++
			continue RunLoop
		case code.Type4Pfx:
			dst := opcode.GetA()
			var res Value
			var err *Error
			if opcode.HasType4a() {
				val := c.getReg(opcode.GetB())
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
					cont := c.getReg(dst).AsCont()
					cont.PushEtc(val.AsArray())
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
					c.getReg(dst).AsClosure().AddUpvalue(cell)
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
				return nil, err.AddContext(c)
			}
			if opcode.GetF() {
				c.getReg(dst).AsCont().Push(res)
			} else {
				c.setReg(dst, res)
			}
			pc++
			continue RunLoop
		case code.Type5Pfx:
			switch opcode.GetJ() {
			case code.OpJump:
				pc += int16(opcode.GetOffset())
				continue RunLoop
			case code.OpJumpIf:
				test := Truth(c.getReg(opcode.GetA()))
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
				next := c.getReg(contReg).AsCont()
				// We clear the register containing the continuation to allow
				// garbage collection.  A continuation can only be called once
				// anyway, so that's ok semantically.
				c.clearReg(contReg)
				if opcode.GetF() {
					// It's a tail call
					c.release()
				}
				return next, nil
			default:
				panic("unsupported")
			}
		case code.Type6Pfx:
			dst := opcode.GetA()
			etc := c.getReg(opcode.GetB()).AsArray()
			idx := int(opcode.GetM())
			var val Value
			if idx < len(etc) {
				val = etc[idx]
			}
			if opcode.GetF() {
				tbl := c.getReg(dst).AsTable()
				for i, v := range etc {
					tbl.Set(IntValue(int64(i+idx)), v)
				}
			} else {
				c.setReg(dst, val)
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

func (c *LuaCont) setReg(reg code.Reg, val Value) {
	idx := reg.Idx()
	if reg.IsCell() {
		c.cells[idx].Set(val)
	} else {
		c.registers[idx] = val
	}
}

func (c *LuaCont) getReg(reg code.Reg) Value {
	if reg.IsCell() {
		return *c.cells[reg.Idx()].ref
	}
	return c.registers[reg.Idx()]
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
