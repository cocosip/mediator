package mediator_test

import (
	"errors"
	"testing"

	mediator "github.com/cocosip/mediator"
	"github.com/cocosip/mediator/internal/typekey"
)

type missingRequest struct{}
type missingResponse struct{}

func TestHandlerNotFoundErrorSupportsErrorsIs(t *testing.T) {
	err := mediator.HandlerNotFoundError{
		RequestType:  typekey.Of[missingRequest](),
		ResponseType: typekey.Of[missingResponse](),
	}

	if !errors.Is(err, mediator.ErrHandlerNotFound) {
		t.Fatalf("expected errors.Is(%T, ErrHandlerNotFound) to be true", err)
	}
}

func TestDuplicateHandlerErrorSupportsErrorsIs(t *testing.T) {
	err := mediator.DuplicateHandlerError{
		RequestType:  typekey.Of[missingRequest](),
		ResponseType: typekey.Of[missingResponse](),
	}

	if !errors.Is(err, mediator.ErrDuplicateHandler) {
		t.Fatalf("expected errors.Is(%T, ErrDuplicateHandler) to be true", err)
	}
}

func TestInvalidHandlerErrorSupportsErrorsIs(t *testing.T) {
	err := mediator.InvalidHandlerError{
		Kind:         "request",
		MessageType:  typekey.Of[missingRequest](),
		ResponseType: typekey.Of[missingResponse](),
	}

	if !errors.Is(err, mediator.ErrInvalidHandler) {
		t.Fatalf("expected errors.Is(%T, ErrInvalidHandler) to be true", err)
	}
}
