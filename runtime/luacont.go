package runtime

import (
	"errors"
	"fmt"
	"math"
	"unsafe"

	"github.com/arnodel/golua/code"
)

// LuaCont is a Lua continuation, made from a closure, values for registers and
// some state.
type LuaCont struct {
	*Closure
	registers      []Value
	pc             int32
	acc            []Value
	running        bool
	closeStackBase int
}

var _ Cont = (*LuaCont)(nil)

// NewLuaCont returns a new LuaCont from a closure and next, a continuation to
// push results into.
func NewLuaCont(t *Thread, clos *Closure, next Cont) *LuaCont {
	if clos.upvalueIndex < len(clos.Upvalues) {
		panic("Closure not ready")
	}
	t.RequireArrSize(unsafe.Sizeof(Value{}), int(clos.RegCount))
	registers := t.regPool.getIn(clos.regClass, clos.regSize, int(clos.RegCount))
	registers[0] = ContValue(next)
	// The upvalues the function uses: the box of one it may write or
	// capture again, the contents of a fixed one, which cannot change while
	// the function runs, so they are read once, here.
	for _, l := range clos.entryLoads {
		if l.Box {
			registers[l.Reg] = BoxValue(clos.Upvalues[l.Up])
		} else {
			registers[l.Reg] = *clos.Upvalues[l.Up].ref
		}
	}
	cont := t.luaContPool.get()
	t.RequireSize(unsafe.Sizeof(LuaCont{}))
	// A continuation from the pool is all zeros, as release left it, and so
	// is a new one: only the fields that aren't need setting.  Assigning a
	// whole LuaCont would write every field again.
	cont.Closure = clos
	cont.registers = registers
	cont.closeStackBase = t.closeStack.size()
	return cont
}

func (c *LuaCont) release(r *Runtime) {
	r.regPool.releaseIn(c.regClass, c.regSize, c.registers)
	r.ReleaseArrSize(unsafe.Sizeof(Value{}), int(c.RegCount))
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
			setReg(c.registers, dst, val)
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
		opcode := opcodeAt(opcodes, pc)
		if opcode.HasType1() {
			dst := opcode.GetA()
			x := getReg(regs, opcode.GetB())
			y := getReg(regs, opcode.GetC())
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
			setReg(regs, dst, res)
			pc++
			continue RunLoop
		}
		switch opcode.TypePfx() {
		case code.Type0Pfx:
			dst := opcode.GetA()
			if opcode.GetF() {
				// It's an etc
				setReg(regs, dst, ArrayValue(c.acc))
			} else {
				setReg(regs, dst, NilValue)
			}
			pc++
			continue RunLoop
		case code.Type2Pfx:
			reg := opcode.GetA()
			coll := getReg(regs, opcode.GetB())
			idx := getReg(regs, opcode.GetC())
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
				setReg(regs, reg, val)
			} else {
				val := getReg(regs, reg)
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
			coll := getReg(regs, opcode.GetB())
			key := constAt(consts, uintptr(opcode.GetK8()))
			hint := hintAt(c.slotCache, pc)
			if !opcode.GetF() {
				val, err := indexStringAt(t, coll, key, hint)
				if err != nil {
					c.pc = pc
					return nil, err
				}
				setReg(regs, reg, val)
			} else if err := setIndexStringAt(t, coll, key, getReg(regs, reg), hint); err != nil {
				c.pc = pc
				return nil, err
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
				val = constAt(consts, uintptr(n))
			case code.OpClosureK:
				val = FunctionValue(NewClosure(t.Runtime, constAt(consts, uintptr(n)).AsCode()))
			default:
				panic("Unsupported opcode")
			}
			dst := opcode.GetA()
			if opcode.GetF() {
				// dst must contain a continuation
				cont := getReg(regs, dst).AsCont()
				cont.Push(t.Runtime, val)
			} else {
				setReg(regs, dst, val)
			}
			pc++
			continue RunLoop
		case code.Type4Pfx:
			dst := opcode.GetA()
			var res Value
			var ok bool
			var err error
			if opcode.HasType4a() {
				val := getReg(regs, opcode.GetB())
				switch opcode.GetUnOp() {
				case code.OpNeg:
					res, ok = Unm(val)
					if !ok {
						res, err = unaryArithFallback(t, metaUnm, val)
					}
				case code.OpBitNot:
					res, err = bnot(t, val)
				case code.OpLen:
					res, err = Len(t, val)
				case code.OpCont, code.OpTailCont:
					// Continue, inlined by hand: the compiler won't, as
					// each call it can't inline costs most of its budget
					// (57 of 80), and this makes three.
					var next Cont = c
					if opcode.GetUnOp() == code.OpTailCont {
						next = c.Next()
					}
					switch f := val.iface.(type) {
					case *Closure:
						res = ContValue(NewLuaCont(t, f, next))
					case *GoFunction:
						res = ContValue(NewGoCont(t, f, next))
					default:
						var cont Cont
						cont, err = continueCall(t, val, next)
						res = ContValue(cont)
					}
				case code.OpId:
					res = val
				case code.OpLoad:
					// rB holds a box: its contents (ir.LowerCells).
					if box, isBox := val.iface.(Cell); isBox {
						res = *box.ref
					} else {
						err = errNotABox
					}
				case code.OpStore:
					// rA holds a box: val goes into it.
					if box, isBox := regs[dst].iface.(Cell); isBox {
						box.set(val)
						pc++
						continue RunLoop
					}
					err = errNotABox
				case code.OpUpvalue:
					// rB holds a box: the closure rA captures it.
					if box, isBox := val.iface.(Cell); isBox {
						regs[dst].AsClosure().AddUpvalue(box)
						pc++
						continue RunLoop
					}
					err = errNotABox
				case code.OpEtcId:
					cont := getReg(regs, dst).AsCont()
					var etc []Value
					etc, err = expandVarargs(t, val)
					if err != nil {
						break
					}
					cont.PushEtc(t.Runtime, etc)
					pc++
					continue RunLoop
				case code.OpTruth:
					res = BoolValue(Truth(val))
				case code.OpNot:
					res = BoolValue(!Truth(val))
				case code.OpMkVarargTable:
					tbl := TableValue(t.newVarargTable(val.AsArray()))
					// Overwrite the etc register so ... expansion reads from
					// the table (respecting t.n modifications).
					setReg(regs, opcode.GetB(), tbl)
					res = tbl
				case code.OpCheckNotDefined:
					// Lua 5.5: check that _ENV[name] is not already defined.
					// Used by "global x = value" declarations.  _ENV is
					// indexed as any value is, so it needn't be a table.
					env := getReg(regs, dst) // rA = _ENV
					// val is rB = index (name)
					v, err := Index(t, env, val)
					if err == nil && !v.IsNil() {
						err = fmt.Errorf("global '%s' already defined", val.AsString())
					}
					if err != nil {
						c.pc = pc
						return nil, err
					}
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
					res = TableValue(t.NewTable())
				case code.OpStr0:
					res = StringValue("")
				case code.OpStr1:
					res = StringValue(string(opcode.GetL().ToStr1()))
				case code.OpBool:
					res = BoolValue(opcode.GetL().ToBool())
				case code.OpNil:
					res = NilValue
				case code.OpNewBox:
					t.RequireSize(unsafe.Sizeof(Value{}))
					res = BoxValue(newCell(NilValue))
				default:
					panic("unsupported")
				}
			}
			if err != nil {
				c.pc = pc
				return nil, err
			}
			if opcode.GetF() {
				getReg(regs, dst).AsCont().Push(t.Runtime, res)
			} else {
				setReg(regs, dst, res)
			}
			pc++
			continue RunLoop
		case code.Type5Pfx:
			switch opcode.GetJ() {
			case code.OpJump:
				pc += int32(opcode.GetOffset())
				continue RunLoop
			case code.OpJumpIf:
				test := Truth(getReg(regs, opcode.GetA()))
				if test == opcode.GetF() {
					pc += int32(opcode.GetOffset())
				} else {
					pc++
				}
				continue RunLoop
			case code.OpCall:
				contReg := opcode.GetA()
				isTail := opcode.GetF() // Can mean tail call or simple return

				// The register holds either a continuation or the callee,
				// whose frame the call makes: Continue inlined by hand, as
				// for the cont opcodes.  r0 always holds the caller's
				// continuation, so "call r0", a return, needs no test; a
				// continuation anywhere else was made by a cont opcode.  A
				// callee stays in its register, which the call leaves alone:
				// it may be a local.
				callee := getReg(regs, contReg)
				var next Cont
				isCont := true
				if contReg == code.ValueReg(0) {
					next = callee.AsCont()
				} else if lc, ok := callee.iface.(*LuaCont); ok {
					next = lc
				} else {
					isCont = false
					var cnext Cont = c
					if isTail {
						cnext = c.Next()
					}
					switch f := callee.iface.(type) {
					case *Closure:
						next = NewLuaCont(t, f, cnext)
					case *GoFunction:
						next = NewGoCont(t, f, cnext)
					default:
						if next, isCont = callee.TryCont(); !isCont {
							var err error
							if next, err = continueCall(t, callee, cnext); err != nil {
								c.pc = pc
								return nil, err
							}
						}
					}
				}
				pc++

				// Push the arguments listed in the operand words, then skip
				// the words.  The receiver is usually a Lua function's
				// frame: a value its receive opcode puts in a register is
				// stored there directly, which is Push inlined by hand (the
				// compiler can't, as Push has to call out for varargs).
				if n := opcode.GetArgCount(); n > 0 {
					lnext, isLua := next.(*LuaCont)
					for ; n > 0; pc++ {
						w := opcodeAt(opcodes, pc)
						clear := w.ArgClears()
						for k, reg := range [3]code.Reg{w.GetA(), w.GetB(), w.GetC()} {
							if n == 0 {
								break
							}
							arg := getReg(regs, reg)
							if !isLua {
								next.Push(t.Runtime, arg)
							} else if r := lnext.code[lnext.pc]; r.HasType0() && !r.GetF() {
								t.RequireCPU(1)
								lnext.pc++
								setReg(lnext.registers, r.GetA(), arg)
							} else {
								lnext.Push(t.Runtime, arg)
							}
							if clear[k] {
								setReg(regs, reg, NilValue)
							}
							n--
						}
					}
				}
				c.pc = pc
				c.acc = nil
				c.running = false

				// We clear the register containing the continuation to allow
				// garbage collection.  A continuation can only be called once
				// anyway, so that's ok semantically.
				if isCont {
					c.clearReg(contReg)
				}

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
					v := getReg(regs, opcode.GetA())
					if Truth(v) && t.metaGet(v, metaClose).IsNil() {
						c.pc = pc
						return nil, errors.New("to be closed value missing a __close metamethod")
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
			etc, etcErr := expandVarargs(t, getReg(regs, opcode.GetB()))
			if etcErr != nil {
				c.pc = pc
				return nil, etcErr
			}
			idx := int(opcode.GetM())
			if opcode.GetF() {
				tbl := getReg(regs, dst).AsTable()
				for i, v := range etc {
					t.SetTable(tbl, IntValue(int64(i+idx)), v)
				}
			} else {
				var val Value
				if idx < len(etc) {
					val = etc[idx]
				}
				setReg(regs, dst, val)
			}
			pc++
			continue RunLoop
		case code.Type7Pfx:
			// An opcode that decides the jump following it, which it takes
			// itself or steps over (code/doc.go, Type 7).
			switch opcode.GetP() {
			case code.OpAdvFor:
				startReg, stopReg, stepReg := opcode.GetA(), opcode.GetB(), opcode.GetC()
				start := getReg(regs, startReg)
				stop := getReg(regs, stopReg)
				step := getReg(regs, stepReg)
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
					next, _ = Add(start, step)
					done = forPassed(next, stop, step) || forPassed(start, next, step)
				}
				if !done {
					setReg(regs, startReg, next)
				}
				pc = jumpAfter(opcodes, pc, !done)
				continue RunLoop
			case code.OpPrepFor:
				startReg, stopReg, stepReg := opcode.GetA(), opcode.GetB(), opcode.GetC()
				start, tstart := ToNumberValue(getReg(regs, startReg))
				stop, tstop := ToNumberValue(getReg(regs, stopReg))
				step, tstep := ToNumberValue(getReg(regs, stepReg))
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
					return nil, fmt.Errorf("'for' %s: expected number, got %s", role, val.CustomTypeName())
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
					c.pc = pc
					return nil, errors.New("'for' step is zero")
				}
				// The loop may have no iteration at all.
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
				setReg(regs, startReg, start)
				setReg(regs, stopReg, stop)
				setReg(regs, stepReg, step)
				pc = jumpAfter(opcodes, pc, done)
				continue RunLoop

			// The predicates: the comparison decides the jump.

			case code.OpIfEq:
				r, err := eq(t, getReg(regs, opcode.GetB()), getReg(regs, opcode.GetC()))
				if err != nil {
					c.pc = pc
					return nil, err
				}
				pc = jumpAfter(opcodes, pc, r != opcode.Negated())
				continue RunLoop
			case code.OpIfLt:
				r, err := Lt(t, getReg(regs, opcode.GetB()), getReg(regs, opcode.GetC()))
				if err != nil {
					c.pc = pc
					return nil, err
				}
				pc = jumpAfter(opcodes, pc, r != opcode.Negated())
				continue RunLoop
			case code.OpIfLeq:
				r, err := le(t, getReg(regs, opcode.GetB()), getReg(regs, opcode.GetC()))
				if err != nil {
					c.pc = pc
					return nil, err
				}
				pc = jumpAfter(opcodes, pc, r != opcode.Negated())
				continue RunLoop
			default:
				panic("unsupported")
			}
		}
	}
	// return nil, errors.New("Invalid PC")
}

// jumpAfter returns where execution goes after the opcode at pc and the
// jump that follows it: the jump's target if it is taken, else the opcode
// after the jump.
func jumpAfter(opcodes []code.Opcode, pc int32, taken bool) int32 {
	pc++
	if taken {
		return pc + int32(opcodeAt(opcodes, pc).GetOffset())
	}
	return pc + 1
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

func (c *LuaCont) clearReg(reg code.Reg) {
	c.registers[reg] = NilValue
}

// The run loop reads and writes registers, and reads the words of a
// function's code, its constants and its lookup hints, on every opcode.  The
// accessors below index without a bounds check, and rely on what Code.verify
// has established before the function could run:
//
//   - every register an opcode names (or lists as an argument of a call) is
//     within the register count that NewLuaCont sizes the frame from;
//   - execution never leaves the code: every jump lands inside it, and
//     wherever execution goes on to after an opcode is there;
//   - every constant an opcode names is in the constant table;
//   - there is a lookup hint for every word of code (the two ways of loading
//     a function make Code.slotCache as long as its code).

// setReg and getReg access a register.  A box's contents are reached only
// through the load and store opcodes (ir.LowerCells).
func setReg(regs []Value, reg code.Reg, val Value) {
	*(*Value)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(regs)), uintptr(reg)*unsafe.Sizeof(Value{}))) = val
}

func getReg(regs []Value, reg code.Reg) Value {
	return *(*Value)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(regs)), uintptr(reg)*unsafe.Sizeof(Value{})))
}

// opcodeAt, constAt and hintAt read the word at pc of a function's code, a
// constant, and the lookup hint of the opcode at pc.
func opcodeAt(opcodes []code.Opcode, pc int32) code.Opcode {
	return *(*code.Opcode)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(opcodes)), uintptr(pc)*unsafe.Sizeof(code.Opcode(0))))
}

func constAt(consts []Value, k uintptr) Value {
	return *(*Value)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(consts)), k*unsafe.Sizeof(Value{})))
}

func hintAt(hints []lookupHint, pc int32) *lookupHint {
	return (*lookupHint)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(hints)), uintptr(pc)*unsafe.Sizeof(lookupHint{})))
}
