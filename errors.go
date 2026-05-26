package mediator

import (
	"errors"
	"fmt"
	"reflect"
)

var (
	ErrHandlerNotFound  = errors.New("mediator: handler not found")
	ErrDuplicateHandler = errors.New("mediator: duplicate handler")
	ErrInvalidHandler   = errors.New("mediator: invalid handler")
)

// HandlerNotFoundError reports a missing request handler registration.
type HandlerNotFoundError struct {
	RequestType  reflect.Type
	ResponseType reflect.Type
}

func (e HandlerNotFoundError) Error() string {
	return fmt.Sprintf(
		"mediator: handler not found for request %s -> %s",
		typeString(e.RequestType),
		typeString(e.ResponseType),
	)
}

func (e HandlerNotFoundError) Unwrap() error {
	return ErrHandlerNotFound
}

// DuplicateHandlerError reports a duplicate request handler registration.
type DuplicateHandlerError struct {
	RequestType  reflect.Type
	ResponseType reflect.Type
}

func (e DuplicateHandlerError) Error() string {
	return fmt.Sprintf(
		"mediator: duplicate handler for request %s -> %s",
		typeString(e.RequestType),
		typeString(e.ResponseType),
	)
}

func (e DuplicateHandlerError) Unwrap() error {
	return ErrDuplicateHandler
}

func typeString(t reflect.Type) string {
	if t == nil {
		return "<nil>"
	}

	return t.String()
}
