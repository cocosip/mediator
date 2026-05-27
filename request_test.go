package mediator_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cocosip/mediator"
)

type createUserRequest struct {
	Name string
}

type createUserResponse struct {
	ID string
}

func TestSendDispatchesRegisteredRequestHandler(t *testing.T) {
	m := mediator.New()

	err := mediator.RegisterRequestHandler(m, mediator.RequestHandlerFunc[createUserRequest, createUserResponse](
		func(_ context.Context, request createUserRequest) (createUserResponse, error) {
			if request.Name != testAlice {
				t.Fatalf("expected request name alice, got %q", request.Name)
			}

			return createUserResponse{ID: testUserID}, nil
		},
	))
	if err != nil {
		t.Fatalf("expected registration to succeed, got %v", err)
	}

	response, err := mediator.Send[createUserRequest, createUserResponse](
		context.Background(),
		m,
		createUserRequest{Name: testAlice},
	)
	if err != nil {
		t.Fatalf("expected send to succeed, got %v", err)
	}

	if response.ID != testUserID {
		t.Fatalf("expected response id user-1, got %q", response.ID)
	}
}

func TestSendReturnsHandlerNotFoundForMissingRequestHandler(t *testing.T) {
	m := mediator.New()

	_, err := mediator.Send[createUserRequest, createUserResponse](
		context.Background(),
		m,
		createUserRequest{Name: testAlice},
	)
	if err == nil {
		t.Fatal("expected missing handler error, got nil")
	}

	if !errors.Is(err, mediator.ErrHandlerNotFound) {
		t.Fatalf("expected ErrHandlerNotFound, got %v", err)
	}
}

func TestRegisterRequestHandlerRejectsDuplicateRegistration(t *testing.T) {
	m := mediator.New()

	handler := mediator.RequestHandlerFunc[createUserRequest, createUserResponse](
		func(_ context.Context, request createUserRequest) (createUserResponse, error) {
			return createUserResponse{ID: request.Name}, nil
		},
	)

	if err := mediator.RegisterRequestHandler(m, handler); err != nil {
		t.Fatalf("expected first registration to succeed, got %v", err)
	}

	err := mediator.RegisterRequestHandler(m, handler)
	if err == nil {
		t.Fatal("expected duplicate handler error, got nil")
	}

	if !errors.Is(err, mediator.ErrDuplicateHandler) {
		t.Fatalf("expected ErrDuplicateHandler, got %v", err)
	}
}

func TestRegisterRequestHandlerRejectsNilHandler(t *testing.T) {
	m := mediator.New()

	err := mediator.RegisterRequestHandler[createUserRequest, createUserResponse](m, nil)
	if err == nil {
		t.Fatal("expected invalid handler error, got nil")
	}

	if !errors.Is(err, mediator.ErrInvalidHandler) {
		t.Fatalf("expected ErrInvalidHandler, got %v", err)
	}
}

func TestSendReturnsHandlerErrorWithoutPipelineBehaviors(t *testing.T) {
	m := mediator.New()
	handlerErr := errors.New("request failed")

	err := mediator.RegisterRequestHandler(m, mediator.RequestHandlerFunc[createUserRequest, createUserResponse](
		func(_ context.Context, _ createUserRequest) (createUserResponse, error) {
			return createUserResponse{}, handlerErr
		},
	))
	if err != nil {
		t.Fatalf("expected registration to succeed, got %v", err)
	}

	_, err = mediator.Send[createUserRequest, createUserResponse](
		context.Background(),
		m,
		createUserRequest{Name: testAlice},
	)
	if err == nil {
		t.Fatal("expected handler error, got nil")
	}

	if !errors.Is(err, handlerErr) {
		t.Fatalf("expected handler error, got %v", err)
	}
}
