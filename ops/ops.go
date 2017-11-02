package ops

type Op uint

//go:generate stringer -type=Op

const OpOr Op = iota << 8
const OpAnd Op = 1 + iota<<8
const (
	OpLt Op = 2 + iota<<8
	OpLeq
	OpGt
	OpGeq
	OpEq
	OpNeq
)
const OpBitOr Op = 3 + iota<<8
const OpBitXor Op = 4 + iota<<8
const OpBitAnd Op = 5 + iota<<8
const (
	OpShiftL Op = 6 + iota<<8
	OpShiftR
)
const OpConcat Op = 7 + iota<<8
const (
	OpAdd Op = 8 + iota<<8
	OpSub
)
const (
	OpMul Op = 9 + iota<<8
	OpDiv
	OpFloorDiv
	OpMod
)

const (
	OpNeg Op = 10 + iota<<8
	OpNot
	OpLen
	OpBitNot
)

const OpPow Op = 11 + iota<<8
