package runtime

import (
	"fmt"
	"strings"
)

// Error is the error type that can be produced by running continuations.  Each
// error has a message and a context, which is a slice of continuations.  There
// is no call stack, but you can imagine you "unwind" the call stack by
// iterating over this slice.
type Error struct {
	message Value
	context []Cont
}

// NewError returns a new error with the given message and no context.
func NewError(message Value) *Error {
	return &Error{message: message}
}

// NewErrorS returns a new error with a string message and no context.
func NewErrorS(msg string) *Error {
	return NewError(String(msg))
}

// NewErrorE returns a new error with a string message (the error message) and
// no context.
func NewErrorE(e error) *Error {
	return NewErrorS(e.Error())
}

// NewErrorF returns a new error with a string message (obtained by calling
// fmt.Sprintf on the arguments) and no context.
func NewErrorF(msg string, args ...interface{}) *Error {
	return NewErrorS(fmt.Sprintf(msg, args...))
}

// AddContext returns an error message with appended context.
func (e *Error) AddContext(cont Cont) *Error {
	e.context = append(e.context, cont)
	return e
}

// Value returns the message of the error (which can be any Lua Value).
func (e *Error) Value() Value {
	return e.message
}

// Error implements the error interface.
func (e *Error) Error() string {
	// TODO
	return fmt.Sprintf("error: %+v", e.message)
}

// Traceback returns a string that represents the traceback of the error using
// its context.
func (e *Error) Traceback() string {
	var tb []*DebugInfo
	for _, c := range e.context {
		tb = appendTraceback(tb, c)
	}
	sb := strings.Builder{}
	sb.WriteString(e.Error())
	sb.WriteByte('\n')
	for _, info := range tb {
		sourceInfo := info.Source
		if info.CurrentLine > 0 {
			sourceInfo = fmt.Sprintf("%s:%d", sourceInfo, info.CurrentLine)
		}
		sb.WriteString(fmt.Sprintf("in function %s (file %s)\n", info.Name, sourceInfo))
	}
	return sb.String()
}
