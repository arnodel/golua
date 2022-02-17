package runtime

import (
	"errors"
	"fmt"

	"github.com/arnodel/golua/parsing"
	"github.com/arnodel/golua/token"
)

// A SyntaxError is a lua syntax error.
type SyntaxError struct {
	File string
	Err  parsing.Error
}

// NewSyntaxError returns a pointer to a SyntaxError for the error err in file.
func NewSyntaxError(file string, err parsing.Error) *SyntaxError {
	return &SyntaxError{
		File: file,
		Err:  err,
	}
}

// Error implements the error interface.
func (e *SyntaxError) Error() string {
	return fmt.Sprintf("%s:%s", e.File, e.Err)
}

// IsUnexpectedEOF returns true if the error signals that EOF was encountered
// when further tokens were required.
func (e *SyntaxError) IsUnexpectedEOF() bool {
	return e.Err.Got.Type == token.EOF
}

func ErrorIsUnexpectedEOF(err error) bool {
	snErr, ok := AsSyntaxError(err)
	return ok && snErr.IsUnexpectedEOF()
}

func AsSyntaxError(err error) (snErr *SyntaxError, ok bool) {
	ok = errors.As(err, &snErr)
	return
}
