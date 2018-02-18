package code

func switchSkeleton(code []Opcode) {
	i := 0
	for {
		c := code[i]
		if c.HasType1() {
			// Type1
			rA := c.GetA()
			rB := c.GetB()
			rC := c.GetC()
			switch c.GetX() {
			case OpAdd:
			case OpSub:
			case OpMul:
			case OpDiv:
			case OpFloorDiv:
			case OpMod:
			case OpPow:
			case OpBitAnd:
			case OpBitOr:
			case OpShiftL:
			case OpShiftR:
			case OpLt:
			case OpLeq:
			case OpConcat:
			}
			_, _, _ = rA, rB, rC
		} else if c.HasType2or4() {
			rA := c.GetA()
			rB := c.GetB()
			rC := c.GetC()
			f := c.GetF()
			if c.HasSubtypeFlagSet() {
				// Type2
				_, _, _, _ = rA, rB, rC, f
			} else if c != 0 {
				// Type4a
				_, _, _, _ = rA, rB, rC, f
			} else {
				// Type4b
				_, _, _, _ = rA, rB, rC, f
			}
		} else {
			rA := c.GetA()
			n := c.GetN()
			f := c.GetF()
			y := c.GetY()
			if c.HasSubtypeFlagSet() {
				// Type3
				_, _, _, _ = rA, n, f, y
			} else {
				// Type5
				_, _, _, _ = rA, n, f, y
			}
		}
	}
}
