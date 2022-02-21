package token

import (
	"fmt"
)

type Token struct {
	Type
	Lit []byte
	Pos
}

func (t *Token) String() string {
	if t == nil {
		return "nil"
	}
	return fmt.Sprintf("Token(type=%d, lit=%s, pos=%s", t.Type, t.Lit, t.Pos)
}

type Type int

const (
	INVALID Type = iota
	UNFINISHED
	EOF
	LONGSTRING
	NUMDEC
	STRING
	NUMHEX
	IDENT

	KwBreak
	KwGoto
	KwDo
	KwWhile
	KwEnd
	KwRepeat
	KwUntil
	KwThen
	KwElse
	KwElseIf
	KwIf
	KwFor
	KwIn
	KwFunction
	KwLocal
	KwNot
	KwNil
	KwTrue
	KwFalse
	KwReturn

	SgEtc

	SgOpenSquareBkt
	SgCloseSquareBkt
	SgOpenBkt
	SgCloseBkt
	SgOpenBrace
	SgCloseBrace
	SgSemicolon
	SgComma
	SgDot
	SgColon
	SgDoubleColon
	SgAssign
	SgHash

	beforeBinOp

	SgMinus
	SgPlus
	SgStar
	SgSlash
	SgSlashSlash
	SgPct
	SgPipe
	SgTilde
	SgAmpersand
	SgHat
	SgShiftRight
	SgShiftLeft
	SgEqual
	SgNotEqual
	SgLess
	SgLessEqual
	SgGreater
	SgGreaterEqual
	SgConcat
	KwAnd
	KwOr

	afterBinOp
)

func (tp Type) IsBinOp() bool {
	return tp > beforeBinOp && tp < afterBinOp
}

type Pos struct {
	Offset int
	Line   int
	Column int
}

func (p Pos) String() string {
	return fmt.Sprintf("Pos(offset=%d, line=%d, column=%d)", p.Offset, p.Line, p.Column)
}
