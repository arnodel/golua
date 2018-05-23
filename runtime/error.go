package runtime

import (
	"fmt"
	"strings"
)

type Error struct {
	message Value
	context []Cont
}

func NewError(message Value) *Error {
	return &Error{message: message}
}

func NewErrorS(msg string) *Error {
	return NewError(String(msg))
}

func NewErrorE(e error) *Error {
	return NewErrorS(e.Error())
}

func NewErrorF(msg string, args ...interface{}) *Error {
	return NewErrorS(fmt.Sprintf(msg, args...))
}

func (e *Error) AddContext(cont Cont) *Error {
	e.context = append(e.context, cont)
	return e
}

func (e *Error) Value() Value {
	return e.message
}

func (e *Error) Error() string {
	// TODO
	return fmt.Sprintf("error: %+v", e.message)
}

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
