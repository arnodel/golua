package ops

// Op represents a Lua binary or unary operator.  It also encode its precedence.
type Op uint

//go:generate stringer -type=Op

// OpOr is a logical or (precedence 0)
const OpOr Op = 0 + iota<<8

// OpAnd is a logical and (precedence 1)
const OpAnd Op = 1 + iota<<8

// Precedence 2 binary operators
const (
	OpLt Op = 2 + iota<<8
	OpLeq
	OpGt
	OpGeq
	OpEq
	OpNeq
)

// OpBitOr is bitwise or (precedence 3)
const OpBitOr Op = 3 + iota<<8

// OpBitXor is bitwise exclusive or (precedence 4)
const OpBitXor Op = 4 + iota<<8

// OpBitAnd is bitwise and (precedence 5)
const OpBitAnd Op = 5 + iota<<8

// Precedence 6 binary operators (bitwise shifts)
const (
	OpShiftL Op = 6 + iota<<8
	OpShiftR
)

// OpConcat is the concatenate operator (precedence 7)
const OpConcat Op = 7 + iota<<8

// Precendence 8 binary operators (add / subtract)
const (
	OpAdd Op = 8 + iota<<8
	OpSub
)

// Precedence 9 binary operators (multiplication / division / modulo)
const (
	OpMul Op = 9 + iota<<8
	OpDiv
	OpFloorDiv
	OpMod
)

// Unary operators have precedence 10
const (
	OpNeg Op = 10 + iota<<8
	OpNot
	OpLen
	OpBitNot
	OpId
)

// OpPow (power) is special, precendence 10.
const OpPow Op = 11 + iota<<8

// Precedence returns the precedence of an operator (higher means binds more
// tightly).
func (op Op) Precedence() int {
	return int(op & 0xff)
}

// Type returns the type of operator (which coincides with the precedence atm).
func (op Op) Type() Op {
	return op & 0xff
}
