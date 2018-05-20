package ast

import (
	"bytes"
	"regexp"
	"strconv"

	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

type String struct {
	Location
	val []byte
}

func NewString(id *token.Token) String {
	s := id.Lit
	return String{
		Location: LocFromToken(id),
		val:      escapeSeqs.ReplaceAllFunc(s[1:len(s)-1], replaceEscapeSeq),
	}
}

func NewLongString(id *token.Token) String {
	s := id.Lit
	idx := bytes.IndexByte(s[1:], '[') + 2
	contents := s[idx : len(s)-idx]
	if contents[0] == '\n' {
		contents = contents[1:]
	}
	return String{
		Location: LocFromToken(id),
		val:      contents,
	}
}

func (s String) HWrite(w HWriter) {
	w.Writef("%q", s.val)
}

func (s String) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	ir.EmitConstant(c, ir.String(s.val), dst)
	return dst
}

var escapeSeqs = regexp.MustCompile(`\\\d{1,3}|\\[xX][0-9a-fA-F]{2}|\\[abtnvfr]|\\z\w*|\[uU]\{[0-9a-fA-F]+\}|\\.`)

func replaceEscapeSeq(e []byte) []byte {
	switch e[1] {
	case 'a':
		return []byte{7}
	case 'b':
		return []byte{8}
	case 't':
		return []byte{9}
	case 'n':
		return []byte{10}
	case 'v':
		return []byte{11}
	case 'f':
		return []byte{12}
	case 'r':
		return []byte{13}
	case 'z':
		return []byte{}
	case 'x', 'X':
		b, err := strconv.ParseInt(string(e[2:]), 16, 64)
		if err != nil {
			panic(err)
		}
		return []byte{byte(b)}
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		b, err := strconv.ParseInt(string(e[2:]), 10, 64)
		if err != nil {
			panic(err)
		}
		return []byte{byte(b)}
	case 'u', 'U':
		i, err := strconv.ParseInt(string(e[3:len(e)-1]), 16, 32)
		if err != nil {
			panic(err)
		}
		return []byte(string(i))
	default:
		return e[1:]
	}
}
