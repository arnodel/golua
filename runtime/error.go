package runtime

import "fmt"

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
