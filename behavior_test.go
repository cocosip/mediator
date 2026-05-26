package mediator_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/cocosip/mediator"
)

type behaviorRequest struct {
	Value string
}

type behaviorResponse struct {
	Value string
}

func TestSendExecutesPipelineBehaviorsInRegistrationOrder(t *testing.T) {
	m := mediator.New()
	steps := make([]string, 0, 5)

	err := mediator.RegisterRequestHandler(m, mediator.RequestHandlerFunc[behaviorRequest, behaviorResponse](
		func(ctx context.Context, request behaviorRequest) (behaviorResponse, error) {
			steps = append(steps, "handler")
			return behaviorResponse{Value: request.Value + "-handled"}, nil
		},
	))
	if err != nil {
		t.Fatalf("expected request handler registration to succeed, got %v", err)
	}

	err = mediator.RegisterPipelineBehavior(m, mediator.PipelineBehaviorFunc[behaviorRequest, behaviorResponse](
		func(ctx context.Context, request behaviorRequest, next mediator.RequestHandlerDelegate[behaviorResponse]) (behaviorResponse, error) {
			steps = append(steps, "first-before")
			response, err := next(ctx)
			steps = append(steps, "first-after")
			return response, err
		},
	))
	if err != nil {
		t.Fatalf("expected first behavior registration to succeed, got %v", err)
	}

	err = mediator.RegisterPipelineBehavior(m, mediator.PipelineBehaviorFunc[behaviorRequest, behaviorResponse](
		func(ctx context.Context, request behaviorRequest, next mediator.RequestHandlerDelegate[behaviorResponse]) (behaviorResponse, error) {
			steps = append(steps, "second-before")
			response, err := next(ctx)
			steps = append(steps, "second-after")
			return response, err
		},
	))
	if err != nil {
		t.Fatalf("expected second behavior registration to succeed, got %v", err)
	}

	response, err := mediator.Send[behaviorRequest, behaviorResponse](
		context.Background(),
		m,
		behaviorRequest{Value: "value"},
	)
	if err != nil {
		t.Fatalf("expected send to succeed, got %v", err)
	}

	if response.Value != "value-handled" {
		t.Fatalf("expected handled response, got %q", response.Value)
	}

	expected := []string{
		"first-before",
		"second-before",
		"handler",
		"second-after",
		"first-after",
	}
	if !reflect.DeepEqual(steps, expected) {
		t.Fatalf("expected steps %v, got %v", expected, steps)
	}
}

func TestSendPipelineBehaviorCanWrapBeforeAndAfterHandler(t *testing.T) {
	m := mediator.New()

	err := mediator.RegisterRequestHandler(m, mediator.RequestHandlerFunc[behaviorRequest, behaviorResponse](
		func(ctx context.Context, request behaviorRequest) (behaviorResponse, error) {
			return behaviorResponse{Value: request.Value + "-handler"}, nil
		},
	))
	if err != nil {
		t.Fatalf("expected request handler registration to succeed, got %v", err)
	}

	err = mediator.RegisterPipelineBehavior(m, mediator.PipelineBehaviorFunc[behaviorRequest, behaviorResponse](
		func(ctx context.Context, request behaviorRequest, next mediator.RequestHandlerDelegate[behaviorResponse]) (behaviorResponse, error) {
			response, err := next(ctx)
			if err != nil {
				return behaviorResponse{}, err
			}

			return behaviorResponse{Value: "before-" + response.Value + "-after"}, nil
		},
	))
	if err != nil {
		t.Fatalf("expected behavior registration to succeed, got %v", err)
	}

	response, err := mediator.Send[behaviorRequest, behaviorResponse](
		context.Background(),
		m,
		behaviorRequest{Value: "value"},
	)
	if err != nil {
		t.Fatalf("expected send to succeed, got %v", err)
	}

	if response.Value != "before-value-handler-after" {
		t.Fatalf("expected wrapped response, got %q", response.Value)
	}
}

func TestSendPipelineBehaviorCanShortCircuitHandler(t *testing.T) {
	m := mediator.New()
	handlerCalled := false

	err := mediator.RegisterRequestHandler(m, mediator.RequestHandlerFunc[behaviorRequest, behaviorResponse](
		func(ctx context.Context, request behaviorRequest) (behaviorResponse, error) {
			handlerCalled = true
			return behaviorResponse{Value: "handler"}, nil
		},
	))
	if err != nil {
		t.Fatalf("expected request handler registration to succeed, got %v", err)
	}

	err = mediator.RegisterPipelineBehavior(m, mediator.PipelineBehaviorFunc[behaviorRequest, behaviorResponse](
		func(ctx context.Context, request behaviorRequest, next mediator.RequestHandlerDelegate[behaviorResponse]) (behaviorResponse, error) {
			return behaviorResponse{Value: "short-circuit"}, nil
		},
	))
	if err != nil {
		t.Fatalf("expected behavior registration to succeed, got %v", err)
	}

	response, err := mediator.Send[behaviorRequest, behaviorResponse](
		context.Background(),
		m,
		behaviorRequest{Value: "value"},
	)
	if err != nil {
		t.Fatalf("expected send to succeed, got %v", err)
	}

	if response.Value != "short-circuit" {
		t.Fatalf("expected short-circuit response, got %q", response.Value)
	}

	if handlerCalled {
		t.Fatal("expected handler not to be called when behavior short-circuits")
	}
}

func TestSendPipelineBehaviorCanWrapHandlerErrors(t *testing.T) {
	m := mediator.New()
	handlerErr := errors.New("handler failed")

	err := mediator.RegisterRequestHandler(m, mediator.RequestHandlerFunc[behaviorRequest, behaviorResponse](
		func(ctx context.Context, request behaviorRequest) (behaviorResponse, error) {
			return behaviorResponse{}, handlerErr
		},
	))
	if err != nil {
		t.Fatalf("expected request handler registration to succeed, got %v", err)
	}

	err = mediator.RegisterPipelineBehavior(m, mediator.PipelineBehaviorFunc[behaviorRequest, behaviorResponse](
		func(ctx context.Context, request behaviorRequest, next mediator.RequestHandlerDelegate[behaviorResponse]) (behaviorResponse, error) {
			response, err := next(ctx)
			if err != nil {
				return behaviorResponse{}, errors.Join(errors.New("behavior wrapper"), err)
			}

			return response, nil
		},
	))
	if err != nil {
		t.Fatalf("expected behavior registration to succeed, got %v", err)
	}

	_, err = mediator.Send[behaviorRequest, behaviorResponse](
		context.Background(),
		m,
		behaviorRequest{Value: "value"},
	)
	if err == nil {
		t.Fatal("expected send to fail, got nil")
	}

	if !errors.Is(err, handlerErr) {
		t.Fatalf("expected wrapped error to include handler error, got %v", err)
	}
}

func TestRegisterPipelineBehaviorRejectsNilHandler(t *testing.T) {
	m := mediator.New()

	err := mediator.RegisterPipelineBehavior[behaviorRequest, behaviorResponse](m, nil)
	if err == nil {
		t.Fatal("expected invalid handler error, got nil")
	}

	if !errors.Is(err, mediator.ErrInvalidHandler) {
		t.Fatalf("expected ErrInvalidHandler, got %v", err)
	}
}

func TestSendWithoutPipelineBehaviorsUsesExistingHandlerPath(t *testing.T) {
	m := mediator.New()

	err := mediator.RegisterRequestHandler(m, mediator.RequestHandlerFunc[behaviorRequest, behaviorResponse](
		func(ctx context.Context, request behaviorRequest) (behaviorResponse, error) {
			return behaviorResponse{Value: request.Value + "-handled"}, nil
		},
	))
	if err != nil {
		t.Fatalf("expected request handler registration to succeed, got %v", err)
	}

	response, err := mediator.Send[behaviorRequest, behaviorResponse](
		context.Background(),
		m,
		behaviorRequest{Value: "value"},
	)
	if err != nil {
		t.Fatalf("expected send to succeed, got %v", err)
	}

	if response.Value != "value-handled" {
		t.Fatalf("expected handled response, got %q", response.Value)
	}
}
