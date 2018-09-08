package runtime

import (
	"fmt"

	ccerrors "github.com/arnodel/golua/errors"
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

func NewSyntaxErrorFromCCError(file string, err *ccerrors.Error) *SyntaxError {
	pos := err.ErrorToken.Pos
	msg := "syntax error"
	errType := ErrSyntaxError
	switch err.ErrorToken.Type {
	case token.INVALID:
		msg = "unexpected symbol"
		errType = ErrSyntaxInvalidToken
	case token.EOF:
		msg = "unexpected EOF"
		errType = ErrSyntaxEOF
	}
	if err.ErrorToken.Type == token.INVALID {
		msg = "unexpected symbol"
	}
	return &SyntaxError{
		File:    file,
		Line:    pos.Line,
		Column:  pos.Column,
		Token:   string(err.ErrorToken.Lit),
		Type:    errType,
		Message: msg,
	}
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s near %q", e.File, e.Line, e.Column, e.Message, e.Token)
}
