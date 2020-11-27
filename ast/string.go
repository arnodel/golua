package ast

import (
	"bytes"
	"errors"
	"regexp"
	"strconv"

	"github.com/arnodel/golua/token"
)

// String is a string literal.
type String struct {
	Location
	Val []byte
}

var _ ExpNode = String{}

// NewString returns a String from a string token, or an error if it couln't.
func NewString(id *token.Token) (ss String, err error) {
	s := id.Lit
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

func NewStringArgs(id *token.Token) ([]ExpNode, error) {
	s, err := NewString(id)
	if err != nil {
		return nil, err
	}
	return []ExpNode{s}, nil
}

func NewLongString(id *token.Token) String {
	s := id.Lit
	idx := bytes.IndexByte(s[1:], '[') + 2
	contents := s[idx : len(s)-idx]
	// contents = newLines.ReplaceAllLiteral(contents, []byte{'\n'})
	if contents[0] == '\n' {
		contents = contents[1:]
	}
	return String{
		Location: LocFromToken(id),
		Val:      contents,
	}
}

func (s String) ProcessExp(p ExpProcessor) {
	p.ProcessStringExp(s)
}

func (s String) HWrite(w HWriter) {
	w.Writef("%q", s.Val)
}

var escapeSeqs = regexp.MustCompile(`(?s)\\\d{1,3}|\\[xX][0-9a-fA-F]{2}|\\[abtnvfr\\]|\\z[\s\v]*|\\[uU]{[0-9a-fA-F]+}|\\.`)

// var newLines = regexp.MustCompile(`(?s)\r\n|\n\r|\r|\n`)

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
			panic(errors.New("decimal escape sequence out of range"))
		}
		return []byte{byte(b)}
	case 'u', 'U':
		i, err := strconv.ParseInt(string(e[3:len(e)-1]), 16, 32)
		if err != nil {
			panic(err)
		}
		if i >= 0x110000 {
			panic(errors.New("unicode escape sequence out of range"))
		}
		return []byte(string(rune(i)))
	default:
		return e[1:]
	}
}
