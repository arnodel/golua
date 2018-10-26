package runtime

import (
	"fmt"

	"github.com/arnodel/golua/parsing"
	"github.com/arnodel/golua/token"
)

const (
	ErrSyntaxError = iota
	ErrSyntaxInvalidToken
	ErrSyntaxEOF
)

type SyntaxError struct {
	File         string
	Line, Column int
	Message      string
	Token        string
	Type         int
}

func NewSyntaxErrorFromCCError(file string, err parsing.Error) *SyntaxError {
	pos := err.Got.Pos
	msg := "syntax error"
	errType := ErrSyntaxError
	switch err.Got.Type {
	case token.INVALID:
		msg = "unexpected symbol"
		errType = ErrSyntaxInvalidToken
	case token.EOF:
		msg = "unexpected EOF"
		errType = ErrSyntaxEOF
	}
	if err.Got.Type == token.INVALID {
		msg = "unexpected symbol"
	}
	return &SyntaxError{
		File:    file,
		Line:    pos.Line,
		Column:  pos.Column,
		Token:   string(err.Got.Lit),
		Type:    errType,
		Message: msg,
	}
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s near %q", e.File, e.Line, e.Column, e.Message, e.Token)
}
