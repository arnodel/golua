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
	handled bool
	lineno  int
	source  string
}

// NewError returns a new error with the given message and no context.
func NewError(message Value) *Error {
	return &Error{message: message}
}

func newHandledError(message Value) *Error {
	return &Error{message: message, handled: true}
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

// AddContext sets the lineno / source fields of the error if not already set
func (e *Error) AddContext(c Cont, depth int) {
	if e.lineno != 0 || e.handled {
		return
	}
	e.lineno = -1
	e.source = "?"
	if depth == 0 {
		return
	}
	if depth > 0 {
		for depth > 1 && c != nil {
			c = c.Parent()
			depth--
		}
	} else {
		for c != nil {
			if _, ok := c.(*LuaCont); ok {
				break
			}
			c = c.Parent()
		}
	}
	if c == nil {
		return
	}
	info := c.DebugInfo()
	if info == nil {
		return
	}
	if info.CurrentLine != 0 {
		e.lineno = int(info.CurrentLine)
	}
	e.source = info.Source
	s, ok := e.message.TryString()
	if ok && e.lineno > 0 {
		e.message = StringValue(fmt.Sprintf("%s:%d: %s", e.source, e.lineno, s))
	}
}

// Value returns the message of the error (which can be any Lua Value).
func (e *Error) Value() Value {
	if e == nil {
		return NilValue
	}
	return e.message
}

func (e *Error) Handled() bool {
	return e.handled
}

// Error implements the error interface.
func (e *Error) Error() string {
	// TODO
	s, _ := e.message.ToString()
	return fmt.Sprintf("error: %s", s)
}

// Traceback produces a traceback string of the continuation, requiring memory
// for the string.
func (r *Runtime) Traceback(pfx string, c Cont) string {
	sb := strings.Builder{}
	needNewline := false
	if len(pfx) > 0 {
		r.RequireBytes(len(pfx))
		sb.WriteString(pfx)
		needNewline = true
	}
	for c != nil {
		info := c.DebugInfo()
		if info != nil {
			if needNewline {
				r.RequireBytes(1)
				sb.WriteByte('\n')
			}
			sourceInfo := info.Source
			if info.CurrentLine > 0 {
				sourceInfo = fmt.Sprintf("%s:%d", sourceInfo, info.CurrentLine)
			}
			line := fmt.Sprintf("in function %s (file %s)", info.Name, sourceInfo)
			r.RequireBytes(len(line))
			sb.WriteString(line)
			needNewline = true
		}
		c = c.Parent()
	}
	return sb.String()
}
