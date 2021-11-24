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
	message   Value
	traceback []*DebugInfo
}

// NewError returns a new error with the given message and no context.
func NewError(message Value) *Error {
	return &Error{message: message}
}

// NewErrorS returns a new error with a string message and no context.
func NewErrorS(msg string) *Error {
	return NewError(StringValue(msg))
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
	info := cont.DebugInfo()
	if info != nil {
		e.traceback = append(e.traceback, info)
	}
	return e
}

// Value returns the message of the error (which can be any Lua Value).
func (e *Error) Value() Value {
	return e.message
}

// Error implements the error interface.
func (e *Error) Error() string {
	// TODO
	s, _ := e.message.ToString()
	return fmt.Sprintf("error: %s", s)
}

// Traceback returns a string that represents the traceback of the error using
// its context.
func (e *Error) Traceback() string {
	// TODO: consume CPU and mem?
	sb := strings.Builder{}
	sb.WriteString(e.Error())
	sb.WriteByte('\n')
	for _, info := range e.traceback {
		sourceInfo := info.Source
		if info.CurrentLine > 0 {
			sourceInfo = fmt.Sprintf("%s:%d", sourceInfo, info.CurrentLine)
		}
		sb.WriteString(fmt.Sprintf("in function %s (file %s)\n", info.Name, sourceInfo))
	}
	return sb.String()
}
