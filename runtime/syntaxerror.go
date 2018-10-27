package runtime

import (
	"fmt"

	"github.com/arnodel/golua/parsing"
	"github.com/arnodel/golua/token"
)

type SyntaxError struct {
	File string
	Err  parsing.Error
}

func NewSyntaxError(file string, err parsing.Error) *SyntaxError {
	return &SyntaxError{
		File: file,
		Err:  err,
	}
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("%s:%s", e.File, e.Err)
}

func (e *SyntaxError) IsUnexpectedEOF() bool {
	return e.Err.Got.Type == token.EOF
}
