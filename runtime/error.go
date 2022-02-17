package runtime

import (
	"errors"
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

// AsError check if err can be converted to an *Error and returns that if
// successful.
func AsError(err error) (rtErr *Error, ok bool) {
	ok = errors.As(err, &rtErr)
	return
}

// ToError turns err into an *Error instance.
func ToError(err error) *Error {
	if err == nil {
		return nil
	}
	rtErr, ok := AsError(err)
	if ok {
		return rtErr
	}
	return NewError(StringValue(err.Error()))
}

// ErrorValue extracts a Value from err.  If err is an *Error then it returns
// its Value(), otherwise it builds a Value from the error string.
func ErrorValue(err error) Value {
	if err == nil {
		return NilValue
	}
	rtErr, ok := AsError(err)
	if ok {
		return rtErr.Value()
	}
	return StringValue(err.Error())
}

// AddContext returns a new error with the lineno / source fields set if not
// already set.
func (e *Error) AddContext(c Cont, depth int) *Error {
	if e.lineno != 0 || e.handled {
		return e
	}
	e = &Error{
		lineno:  -1,
		source:  "?",
		message: e.message,
	}
	if depth == 0 {
		return e
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
		return e
	}
	info := c.DebugInfo()
	if info == nil {
		return e
	}
	if info.CurrentLine != 0 {
		e.lineno = int(info.CurrentLine)
	}
	e.source = info.Source
	s, ok := e.message.TryString()
	if ok && e.lineno > 0 {
		e.message = StringValue(fmt.Sprintf("%s:%d: %s", e.source, e.lineno, s))
	}
	return e
}

// Value returns the message of the error (which can be any Lua Value).
func (e *Error) Value() Value {
	if e == nil {
		return NilValue
	}
	return e.message
}

// Handled returns true if the error has been handled (i.e. the message handler
// has processed it).
func (e *Error) Handled() bool {
	return e.handled
}

// Error implements the error interface.
func (e *Error) Error() string {
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
		// log.Printf("XXX %T", c)
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
