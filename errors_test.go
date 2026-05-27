package mediator_test

import (
	"errors"
	"strings"
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

func TestHandlerNotFoundErrorFormatsRequestAndResponseTypes(t *testing.T) {
	err := mediator.HandlerNotFoundError{
		RequestType:  typekey.Of[missingRequest](),
		ResponseType: typekey.Of[missingResponse](),
	}

	message := err.Error()
	if !strings.Contains(message, "missingRequest") || !strings.Contains(message, "missingResponse") {
		t.Fatalf("expected error message to include request and response types, got %q", message)
	}
}

func TestDuplicateHandlerErrorFormatsRequestAndResponseTypes(t *testing.T) {
	err := mediator.DuplicateHandlerError{
		RequestType:  typekey.Of[missingRequest](),
		ResponseType: typekey.Of[missingResponse](),
	}

	message := err.Error()
	if !strings.Contains(message, "missingRequest") || !strings.Contains(message, "missingResponse") {
		t.Fatalf("expected error message to include request and response types, got %q", message)
	}
}

func TestInvalidHandlerErrorFormatsRequestKinds(t *testing.T) {
	err := mediator.InvalidHandlerError{
		Kind:         "pipeline",
		MessageType:  typekey.Of[missingRequest](),
		ResponseType: typekey.Of[missingResponse](),
	}

	message := err.Error()
	if !strings.Contains(message, "invalid pipeline handler") {
		t.Fatalf("expected error message to include explicit kind, got %q", message)
	}
}

func TestInvalidHandlerErrorFallsBackForNotificationKindsAndNilTypes(t *testing.T) {
	err := mediator.InvalidHandlerError{}

	message := err.Error()
	if !strings.Contains(message, "invalid notification handler") {
		t.Fatalf("expected notification fallback kind, got %q", message)
	}

	if !strings.Contains(message, "<nil>") {
		t.Fatalf("expected nil types to format as <nil>, got %q", message)
	}
}
