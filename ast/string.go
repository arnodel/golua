package ast

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"

	"github.com/arnodel/golua/luastrings"
	"github.com/arnodel/golua/token"
)

// String is a string literal.
type String struct {
	Location
	Val []byte
}

var _ ExpNode = String{}

// NewString returns a String from a token containing a Lua short string
// literal, or an error if it couln't (although perhaps it should panic instead?
// Parsing should ensure well-formed string token).
func NewString(id *token.Token) (ss String, err error) {
	s := luastrings.NormalizeNewLines(id.Lit)
	defer func() {
		if r := recover(); r != nil {
			err2, ok := r.(error)
			if ok {
				err = err2
			}
		}
	}()
	return String{
		Location: LocFromToken(id),
		Val:      escapeSeqs.ReplaceAllFunc(s[1:len(s)-1], replaceEscapeSeq),
	}, nil
}

// NewLongString returns a String computed from a token containing a Lua long
// string.
func NewLongString(id *token.Token) String {
	s := luastrings.NormalizeNewLines(id.Lit)
	idx := bytes.IndexByte(s[1:], '[') + 2
	contents := s[idx : len(s)-idx]
	if contents[0] == '\n' {
		contents = contents[1:]
	}
	return String{
		Location: LocFromToken(id),
		Val:      contents,
	}
}

// ProcessExp uses the given ExpProcessor to process the receiver.
func (s String) ProcessExp(p ExpProcessor) {
	p.ProcessStringExp(s)
}

// HWrite prints a tree representation of the node.
func (s String) HWrite(w HWriter) {
	w.Writef("%q", s.Val)
}

// This function replaces escape sequences with the values they escape.
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
		b, err := strconv.ParseInt(string(e[1:]), 10, 64)
		if err != nil {
			panic(err)
		}
		if b >= 256 {
			panic(fmt.Errorf("decimal escape sequence out of range near '%s'", e))
		}
		return []byte{byte(b)}
	case 'u', 'U':
		i, err := strconv.ParseInt(string(e[3:len(e)-1]), 16, 32)
		if err != nil {
			panic(fmt.Errorf("unicode escape sequence out of range near '%s'", e))
		}
		var p [6]byte
		n := luastrings.UTF8EncodeInt32(p[:], int32(i))
		return p[:n]
	default:
		return e[1:]
	}
}

// This regex matches all the escape sequences that can be found in a Lua string.
var escapeSeqs = regexp.MustCompile(`(?s)\\\d{1,3}|\\[xX][0-9a-fA-F]{2}|\\[abtnvfr\\]|\\z[\s\v]*|\\[uU]{[0-9a-fA-F]+}|\\.`)
