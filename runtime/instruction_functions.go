package runtime

import "github.com/arnodel/golua/code"

func convertOpcode(opcode code.Opcode) func(*LuaCont, *Thread) (int, error) {
	if opcode.HasType1() {
		return type1Instr(opcode)
	}
	switch opcode.TypePfx() {
	case code.Type0Pfx:
		return type0Instr(opcode)
	case code.Type2Pfx:
		return type2Instr(opcode)
	case code.Type3Pfx:
		return type3Instr(opcode)
	case code.Type4Pfx:
		return type4Instr(opcode)
	case code.Type5Pfx:
		return type5Instr(opcode)
	case code.Type6Pfx:
		return type6Instr(opcode)
	case code.Type7Pfx:
		return type7Instr(opcode)
	default:
		panic("unsupported opcode")
	}
}
func type1Instr(opcode code.Opcode) func(*LuaCont, *Thread) (int, error) {
	dst := opcode.GetA()
	r1 := opcode.GetB()
	r2 := opcode.GetC()
	switch opcode.GetX() {

	// Arithmetic

	case code.OpAdd:
		return func(c *LuaCont, t *Thread) (int, error) {
			x := getReg(c.registers, c.cells, r1)
			y := getReg(c.registers, c.cells, r2)
			res, ok := Add(x, y)
			if !ok {
				var err error
				res, err = binaryArithFallback(t, "__add", x, y)
				if err != nil {
					return 0, err
				}
			}
			setReg(c.registers, c.cells, dst, res)
			return 1, nil
		}
	case code.OpSub:
		return func(c *LuaCont, t *Thread) (int, error) {
			x := getReg(c.registers, c.cells, r1)
			y := getReg(c.registers, c.cells, r2)
			res, ok := Sub(x, y)
			if !ok {
				var err error
				res, err = binaryArithFallback(t, "__sub", x, y)
				if err != nil {
					return 0, err
				}
			}
			setReg(c.registers, c.cells, dst, res)
			return 1, nil
		}
	case code.OpMul:
		return func(c *LuaCont, t *Thread) (int, error) {
			x := getReg(c.registers, c.cells, r1)
			y := getReg(c.registers, c.cells, r2)
			res, ok := Mul(x, y)
			if !ok {
				var err error
				res, err = binaryArithFallback(t, "__mul", x, y)
				if err != nil {
					return 0, err
				}
			}
			setReg(c.registers, c.cells, dst, res)
			return 1, nil
		}
	case code.OpDiv:
		return func(c *LuaCont, t *Thread) (int, error) {
			x := getReg(c.registers, c.cells, r1)
			y := getReg(c.registers, c.cells, r2)
			res, ok := Div(x, y)
			if !ok {
				var err error
				res, err = binaryArithFallback(t, "__div", x, y)
				if err != nil {
					return 0, err
				}
			}
			setReg(c.registers, c.cells, dst, res)
			return 1, nil
		}
	case code.OpFloorDiv:
		return func(c *LuaCont, t *Thread) (int, error) {
			x := getReg(c.registers, c.cells, r1)
			y := getReg(c.registers, c.cells, r2)
			res, ok, err := Idiv(x, y)
			if !ok {
				res, err = binaryArithFallback(t, "__idiv", x, y)
			}
			if err != nil {
				return 0, err
			}
			setReg(c.registers, c.cells, dst, res)
			return 1, nil
		}
	case code.OpMod:
		return func(c *LuaCont, t *Thread) (int, error) {
			x := getReg(c.registers, c.cells, r1)
			y := getReg(c.registers, c.cells, r2)
			res, ok, err := Mod(x, y)
			if !ok {
				res, err = binaryArithFallback(t, "__mod", x, y)
			}
			if err != nil {
				return 0, err
			}
			setReg(c.registers, c.cells, dst, res)
			return 1, nil
		}
	case code.OpPow:
		return func(c *LuaCont, t *Thread) (int, error) {
			x := getReg(c.registers, c.cells, r1)
			y := getReg(c.registers, c.cells, r2)
			res, ok := Pow(x, y)
			if !ok {
				var err error
				res, err = binaryArithFallback(t, "__pow", x, y)
				if err != nil {
					return 0, err
				}
			}
			setReg(c.registers, c.cells, dst, res)
			return 1, nil
		}

	// Bitwise

	case code.OpBitAnd:
		return func(c *LuaCont, t *Thread) (int, error) {
			x := getReg(c.registers, c.cells, r1)
			y := getReg(c.registers, c.cells, r2)
			res, err := band(t, x, y)
			if err != nil {
				return 0, err
			}
			setReg(c.registers, c.cells, dst, res)
			return 1, nil
		}
	case code.OpBitOr:
		return func(c *LuaCont, t *Thread) (int, error) {
			x := getReg(c.registers, c.cells, r1)
			y := getReg(c.registers, c.cells, r2)
			res, err := bor(t, x, y)
			if err != nil {
				return 0, err
			}
			setReg(c.registers, c.cells, dst, res)
			return 1, nil
		}
	case code.OpBitXor:
		return func(c *LuaCont, t *Thread) (int, error) {
			x := getReg(c.registers, c.cells, r1)
			y := getReg(c.registers, c.cells, r2)
			res, err := bxor(t, x, y)
			if err != nil {
				return 0, err
			}
			setReg(c.registers, c.cells, dst, res)
			return 1, nil
		}
	case code.OpShiftL:
		return func(c *LuaCont, t *Thread) (int, error) {
			x := getReg(c.registers, c.cells, r1)
			y := getReg(c.registers, c.cells, r2)
			res, err := shl(t, x, y)
			if err != nil {
				return 0, err
			}
			setReg(c.registers, c.cells, dst, res)
			return 1, nil
		}
	case code.OpShiftR:
		return func(c *LuaCont, t *Thread) (int, error) {
			x := getReg(c.registers, c.cells, r1)
			y := getReg(c.registers, c.cells, r2)
			res, err := shr(t, x, y)
			if err != nil {
				return 0, err
			}
			setReg(c.registers, c.cells, dst, res)
			return 1, nil
		}

	// Comparison

	case code.OpEq:
		return func(c *LuaCont, t *Thread) (int, error) {
			x := getReg(c.registers, c.cells, r1)
			y := getReg(c.registers, c.cells, r2)
			r, err := eq(t, x, y)
			if err != nil {
				return 0, err
			}
			res := BoolValue(r)
			setReg(c.registers, c.cells, dst, res)
			return 1, nil
		}
	case code.OpLt:
		return func(c *LuaCont, t *Thread) (int, error) {
			x := getReg(c.registers, c.cells, r1)
			y := getReg(c.registers, c.cells, r2)
			r, err := Lt(t, x, y)
			if err != nil {
				return 0, err
			}
			res := BoolValue(r)
			setReg(c.registers, c.cells, dst, res)
			return 1, nil
		}
	case code.OpLeq:
		return func(c *LuaCont, t *Thread) (int, error) {
			x := getReg(c.registers, c.cells, r1)
			y := getReg(c.registers, c.cells, r2)
			r, err := le(t, x, y)
			if err != nil {
				return 0, err
			}
			res := BoolValue(r)
			setReg(c.registers, c.cells, dst, res)
			return 1, nil
		}

	// Concatenation

	case code.OpConcat:
		return func(c *LuaCont, t *Thread) (int, error) {
			x := getReg(c.registers, c.cells, r1)
			y := getReg(c.registers, c.cells, r2)
			res, err := Concat(t, x, y)
			if err != nil {
				return 0, err
			}
			setReg(c.registers, c.cells, dst, res)
			return 1, nil
		}
	default:
		panic("unsupported opcode")
	}
}

func type0Instr(opcode code.Opcode) func(*LuaCont, *Thread) (int, error) {
	dst := opcode.GetA()
	if opcode.GetF() {
		// It's an etc
		return func(c *LuaCont, t *Thread) (int, error) {
			setReg(c.registers, c.cells, dst, ArrayValue(c.acc))
			return 1, nil
		}
	} else {
		return func(c *LuaCont, t *Thread) (int, error) {
			setReg(c.registers, c.cells, dst, NilValue)
			return 1, nil
		}
	}
}

func type2Instr(opcode code.Opcode) func(*LuaCont, *Thread) (int, error) {
	reg := opcode.GetA()
	r1 := opcode.GetB()
	r2 := opcode.GetC()
	if !opcode.GetF() {
		return func(c *LuaCont, t *Thread) (int, error) {
			coll := getReg(c.registers, c.cells, r1)
			idx := getReg(c.registers, c.cells, r2)
			val, err := Index(t, coll, idx)
			if err != nil {
				return 0, err
			}
			setReg(c.registers, c.cells, reg, val)
			return 1, nil
		}
	} else {
		return func(c *LuaCont, t *Thread) (int, error) {
			coll := getReg(c.registers, c.cells, r1)
			idx := getReg(c.registers, c.cells, r2)
			err := SetIndex(t, coll, idx, getReg(c.registers, c.cells, reg))
			if err != nil {
				return 0, err
			}
			return 1, nil
		}
	}
}

func type3Instr(opcode code.Opcode) func(*LuaCont, *Thread) (int, error) {
	n := opcode.GetN()
	dst := opcode.GetA()
	f := opcode.GetF()
	switch opcode.GetY() {
	case code.OpInt16:
		return func(c *LuaCont, t *Thread) (int, error) {
			val := IntValue(int64(int16(n)))
			if f {
				cont := getReg(c.registers, c.cells, dst).AsCont()
				cont.Push(t.Runtime, val)
			} else {
				setReg(c.registers, c.cells, dst, val)
			}
			return 1, nil
		}
	case code.OpStr2:
		return func(c *LuaCont, t *Thread) (int, error) {
			val := StringValue(string(code.Lit16(n).ToStr2()))
			if f {
				cont := getReg(c.registers, c.cells, dst).AsCont()
				cont.Push(t.Runtime, val)
			} else {
				setReg(c.registers, c.cells, dst, val)
			}
			return 1, nil
		}
	case code.OpK:
		return func(c *LuaCont, t *Thread) (int, error) {
			val := c.consts[n]
			if f {
				cont := getReg(c.registers, c.cells, dst).AsCont()
				cont.Push(t.Runtime, val)
			} else {
				setReg(c.registers, c.cells, dst, val)
			}
			return 1, nil
		}
	case code.OpClosureK:
		return func(c *LuaCont, t *Thread) (int, error) {
			val := FunctionValue(NewClosure(t.Runtime, c.consts[n].AsCode()))
			if f {
				cont := getReg(c.registers, c.cells, dst).AsCont()
				cont.Push(t.Runtime, val)
			} else {
				setReg(c.registers, c.cells, dst, val)
			}
			return 1, nil
		}
	default:
		panic("unsupported opcode")
	}
}

func type4Instr(opcode code.Opcode) func(*LuaCont, *Thread) (int, error) {
	if opcode.HasType4a() {
		return type4aInstr(opcode)
	} else {
		return type4bInstr(opcode)
	}
}

func type4aInstr(opcode code.Opcode) func(*LuaCont, *Thread) (int, error) {
	dst := opcode.GetA()
	r1 := opcode.GetB()
	f := opcode.GetF()
	switch opcode.GetUnOp() {
	case code.OpNeg:
		return func(c *LuaCont, t *Thread) (int, error) {
			val := getReg(c.registers, c.cells, r1)
			res, ok := Unm(val)
			if !ok {
				var err error
				res, err = unaryArithFallback(t, "__unm", val)
				if err != nil {
					return 0, err
				}
			}
			if f {
				getReg(c.registers, c.cells, dst).AsCont().Push(t.Runtime, res)
			} else {
				setReg(c.registers, c.cells, dst, res)
			}
			return 1, nil
		}
	case code.OpBitNot:
		return func(c *LuaCont, t *Thread) (int, error) {
			val := getReg(c.registers, c.cells, r1)
			res, err := bnot(t, val)
			if err != nil {
				return 0, err
			}
			if f {
				getReg(c.registers, c.cells, dst).AsCont().Push(t.Runtime, res)
			} else {
				setReg(c.registers, c.cells, dst, res)
			}
			return 1, nil
		}
	case code.OpLen:
		return func(c *LuaCont, t *Thread) (int, error) {
			val := getReg(c.registers, c.cells, r1)
			res, err := Len(t, val)
			if err != nil {
				return 0, err
			}
			if f {
				getReg(c.registers, c.cells, dst).AsCont().Push(t.Runtime, res)
			} else {
				setReg(c.registers, c.cells, dst, res)
			}
			return 1, nil
		}
	case code.OpCont:
		return func(c *LuaCont, t *Thread) (int, error) {
			val := getReg(c.registers, c.cells, r1)
			cont, err := Continue(t, val, c.Next())
			res := ContValue(cont)
			if err != nil {
				return 0, err
			}
			if f {
				getReg(c.registers, c.cells, dst).AsCont().Push(t.Runtime, res)
			} else {
				setReg(c.registers, c.cells, dst, res)
			}
			return 1, nil
		}
	case code.OpId:
		return func(c *LuaCont, t *Thread) (int, error) {
			res := getReg(c.registers, c.cells, r1)
			if f {
				getReg(c.registers, c.cells, dst).AsCont().Push(t.Runtime, res)
			} else {
				setReg(c.registers, c.cells, dst, res)
			}
			return 1, nil
		}
	case code.OpEtcId:
		return func(c *LuaCont, t *Thread) (int, error) {
			val := getReg(c.registers, c.cells, r1)
			// We assume it's a push?
			cont := getReg(c.registers, c.cells, dst).AsCont()
			cont.PushEtc(t.Runtime, val.AsArray())
			return 1, nil
		}
	case code.OpTruth:
		return func(c *LuaCont, t *Thread) (int, error) {
			val := getReg(c.registers, c.cells, r1)
			res := BoolValue(Truth(val))
			if f {
				getReg(c.registers, c.cells, dst).AsCont().Push(t.Runtime, res)
			} else {
				setReg(c.registers, c.cells, dst, res)
			}
			return 1, nil
		}
	case code.OpNot:
		return func(c *LuaCont, t *Thread) (int, error) {
			val := getReg(c.registers, c.cells, r1)
			res := BoolValue(!Truth(val))
			if f {
				getReg(c.registers, c.cells, dst).AsCont().Push(t.Runtime, res)
			} else {
				setReg(c.registers, c.cells, dst, res)
			}
			return 1, nil
		}
	case code.OpUpvalue:
		return func(c *LuaCont, t *Thread) (int, error) {
			cell := c.getRegCell(opcode.GetB())
			getReg(c.registers, c.cells, dst).AsClosure().AddUpvalue(cell)
			return 1, nil
		}
	default:
		panic("unsupported opcode")
	}
}

func type4bInstr(opcode code.Opcode) func(*LuaCont, *Thread) (int, error) {
	dst := opcode.GetA()
	f := opcode.GetF()
	switch opcode.GetUnOpK() {
	case code.OpCC:
		return func(c *LuaCont, t *Thread) (int, error) {
			res := ContValue(c)
			if f {
				getReg(c.registers, c.cells, dst).AsCont().Push(t.Runtime, res)
			} else {
				setReg(c.registers, c.cells, dst, res)
			}
			return 1, nil
		}
	case code.OpTable:
		return func(c *LuaCont, t *Thread) (int, error) {
			res := TableValue(NewTable())
			if f {
				getReg(c.registers, c.cells, dst).AsCont().Push(t.Runtime, res)
			} else {
				setReg(c.registers, c.cells, dst, res)
			}
			return 1, nil
		}
	case code.OpStr0:
		emptyStr := StringValue("")
		return constInstr(emptyStr, f, dst)
	case code.OpStr1:
		str1 := StringValue(string(opcode.GetL().ToStr1()))
		return constInstr(str1, f, dst)
	case code.OpBool:
		boolVal := BoolValue(opcode.GetL().ToBool())
		return constInstr(boolVal, f, dst)
	case code.OpNil:
		return constInstr(NilValue, f, dst)
	case code.OpClear:
		return func(c *LuaCont, t *Thread) (int, error) {
			c.clearReg(dst)
			return 1, nil
		}
	default:
		panic("unsupported opcode")
	}
}

func constInstr(val Value, f bool, dst code.Reg) func(*LuaCont, *Thread) (int, error) {
	if f {
		return func(c *LuaCont, t *Thread) (int, error) {
			getReg(c.registers, c.cells, dst).AsCont().Push(t.Runtime, val)
			return 1, nil
		}
	} else {
		return func(c *LuaCont, t *Thread) (int, error) {
			setReg(c.registers, c.cells, dst, val)
			return 1, nil
		}
	}
}

func type5Instr(opcode code.Opcode) func(*LuaCont, *Thread) (int, error) {
	panic("unsupported opcode")
}

func type6Instr(opcode code.Opcode) func(*LuaCont, *Thread) (int, error) {
	panic("unsupported opcode")
}

func type7Instr(opcode code.Opcode) func(*LuaCont, *Thread) (int, error) {
	panic("unsupported opcode")
}
