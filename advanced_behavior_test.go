package mediator_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cocosip/mediator"
)

type advancedRequest struct {
	Value string
}

type advancedResponse struct {
	Value string
}

func TestSendDoesNotRecoverPanicsByDefault(t *testing.T) {
	m := mediator.New()

	err := mediator.RegisterRequestHandler(m, mediator.RequestHandlerFunc[advancedRequest, advancedResponse](
		func(_ context.Context, _ advancedRequest) (advancedResponse, error) {
			panic(testHandlerPanic)
		},
	))
	if err != nil {
		t.Fatalf("expected request handler registration to succeed, got %v", err)
	}

	defer func() {
		recovered := recover()
		if recovered != testHandlerPanic {
			t.Fatalf("expected handler panic to propagate, got %v", recovered)
		}
	}()

	_, _ = mediator.Send[advancedRequest, advancedResponse](
		context.Background(),
		m,
		advancedRequest{Value: testValue},
	)
}

func TestRecoverBehaviorConvertsPanicToError(t *testing.T) {
	m := mediator.New()
	panicErr := errors.New("request failed after panic")
	var recoveredValue any

	err := mediator.RegisterRequestHandler(m, mediator.RequestHandlerFunc[advancedRequest, advancedResponse](
		func(_ context.Context, _ advancedRequest) (advancedResponse, error) {
			panic(testHandlerPanic)
		},
	))
	if err != nil {
		t.Fatalf("expected request handler registration to succeed, got %v", err)
	}

	err = mediator.RegisterPipelineBehavior(m, mediator.RecoverBehavior[advancedRequest, advancedResponse](
		func(_ context.Context, request advancedRequest, recovered any) error {
			if request.Value != testValue {
				t.Fatalf("expected request value to be passed to recover callback, got %q", request.Value)
			}

			recoveredValue = recovered
			return panicErr
		},
	))
	if err != nil {
		t.Fatalf("expected recover behavior registration to succeed, got %v", err)
	}

	_, err = mediator.Send[advancedRequest, advancedResponse](
		context.Background(),
		m,
		advancedRequest{Value: testValue},
	)
	if err == nil {
		t.Fatal("expected recovered panic error, got nil")
	}

	if !errors.Is(err, panicErr) {
		t.Fatalf("expected recovered panic error, got %v", err)
	}

	if recoveredValue != testHandlerPanic {
		t.Fatalf("expected recovered value to be captured, got %v", recoveredValue)
	}
}

func TestRecoverBehaviorLeavesHandlerErrorsUnchanged(t *testing.T) {
	m := mediator.New()
	handlerErr := errors.New("handler failed")

	err := mediator.RegisterRequestHandler(m, mediator.RequestHandlerFunc[advancedRequest, advancedResponse](
		func(_ context.Context, _ advancedRequest) (advancedResponse, error) {
			return advancedResponse{}, handlerErr
		},
	))
	if err != nil {
		t.Fatalf("expected request handler registration to succeed, got %v", err)
	}

	err = mediator.RegisterPipelineBehavior(m, mediator.RecoverBehavior[advancedRequest, advancedResponse](
		func(_ context.Context, _ advancedRequest, _ any) error {
			return errors.New("unexpected panic")
		},
	))
	if err != nil {
		t.Fatalf("expected recover behavior registration to succeed, got %v", err)
	}

	_, err = mediator.Send[advancedRequest, advancedResponse](
		context.Background(),
		m,
		advancedRequest{Value: testValue},
	)
	if err == nil {
		t.Fatal("expected handler error, got nil")
	}

	if !errors.Is(err, handlerErr) {
		t.Fatalf("expected original handler error, got %v", err)
	}
}

func TestRecoverBehaviorDoesNotSwallowPanicWhenCallbackReturnsNil(t *testing.T) {
	m := mediator.New()

	err := mediator.RegisterRequestHandler(m, mediator.RequestHandlerFunc[advancedRequest, advancedResponse](
		func(_ context.Context, _ advancedRequest) (advancedResponse, error) {
			panic(testHandlerPanic)
		},
	))
	if err != nil {
		t.Fatalf("expected request handler registration to succeed, got %v", err)
	}

	err = mediator.RegisterPipelineBehavior(m, mediator.RecoverBehavior[advancedRequest, advancedResponse](
		func(_ context.Context, _ advancedRequest, _ any) error {
			return nil
		},
	))
	if err != nil {
		t.Fatalf("expected recover behavior registration to succeed, got %v", err)
	}

	_, err = mediator.Send[advancedRequest, advancedResponse](
		context.Background(),
		m,
		advancedRequest{Value: testValue},
	)
	if err == nil {
		t.Fatal("expected recovered panic error, got nil")
	}
}

func TestRecoverBehaviorRePanicsWhenCallbackIsNil(t *testing.T) {
	m := mediator.New()

	err := mediator.RegisterRequestHandler(m, mediator.RequestHandlerFunc[advancedRequest, advancedResponse](
		func(_ context.Context, _ advancedRequest) (advancedResponse, error) {
			panic(testHandlerPanic)
		},
	))
	if err != nil {
		t.Fatalf("expected request handler registration to succeed, got %v", err)
	}

	err = mediator.RegisterPipelineBehavior(m, mediator.RecoverBehavior[advancedRequest, advancedResponse](nil))
	if err != nil {
		t.Fatalf("expected recover behavior registration to succeed, got %v", err)
	}

	defer func() {
		recovered := recover()
		if recovered != testHandlerPanic {
			t.Fatalf("expected handler panic to be re-panicked, got %v", recovered)
		}
	}()

	_, _ = mediator.Send[advancedRequest, advancedResponse](
		context.Background(),
		m,
		advancedRequest{Value: testValue},
	)
}

func TestPreProcessorRunsBeforeSuccessfulHandler(t *testing.T) {
	m := mediator.New()
	steps := make([]string, 0, 2)

	err := mediator.RegisterRequestHandler(m, mediator.RequestHandlerFunc[advancedRequest, advancedResponse](
		func(_ context.Context, request advancedRequest) (advancedResponse, error) {
			steps = append(steps, "handler:"+request.Value)
			return advancedResponse{Value: request.Value + "-handled"}, nil
		},
	))
	if err != nil {
		t.Fatalf("expected request handler registration to succeed, got %v", err)
	}

	err = mediator.RegisterPipelineBehavior(m, mediator.PreProcessor[advancedRequest, advancedResponse](
		func(_ context.Context, request advancedRequest) error {
			steps = append(steps, "pre:"+request.Value)
			return nil
		},
	))
	if err != nil {
		t.Fatalf("expected pre-processor registration to succeed, got %v", err)
	}

	response, err := mediator.Send[advancedRequest, advancedResponse](
		context.Background(),
		m,
		advancedRequest{Value: testValue},
	)
	if err != nil {
		t.Fatalf("expected send to succeed, got %v", err)
	}

	if response.Value != testValueHandled {
		t.Fatalf("expected handled response, got %q", response.Value)
	}

	expected := []string{"pre:value", "handler:value"}
	if !equalStrings(steps, expected) {
		t.Fatalf("expected steps %v, got %v", expected, steps)
	}
}

func TestPreProcessorErrorShortCircuitsHandler(t *testing.T) {
	m := mediator.New()
	preErr := errors.New("pre failed")
	handlerCalled := false

	err := mediator.RegisterRequestHandler(m, mediator.RequestHandlerFunc[advancedRequest, advancedResponse](
		func(_ context.Context, _ advancedRequest) (advancedResponse, error) {
			handlerCalled = true
			return advancedResponse{Value: "handled"}, nil
		},
	))
	if err != nil {
		t.Fatalf("expected request handler registration to succeed, got %v", err)
	}

	err = mediator.RegisterPipelineBehavior(m, mediator.PreProcessor[advancedRequest, advancedResponse](
		func(_ context.Context, _ advancedRequest) error {
			return preErr
		},
	))
	if err != nil {
		t.Fatalf("expected pre-processor registration to succeed, got %v", err)
	}

	_, err = mediator.Send[advancedRequest, advancedResponse](
		context.Background(),
		m,
		advancedRequest{Value: testValue},
	)
	if err == nil {
		t.Fatal("expected pre-processor error, got nil")
	}

	if !errors.Is(err, preErr) {
		t.Fatalf("expected pre-processor error, got %v", err)
	}

	if handlerCalled {
		t.Fatal("expected handler not to run after pre-processor error")
	}
}

func TestPostProcessorRunsAfterSuccessfulHandler(t *testing.T) {
	m := mediator.New()
	steps := make([]string, 0, 2)

	err := mediator.RegisterRequestHandler(m, mediator.RequestHandlerFunc[advancedRequest, advancedResponse](
		func(_ context.Context, request advancedRequest) (advancedResponse, error) {
			steps = append(steps, "handler:"+request.Value)
			return advancedResponse{Value: request.Value + "-handled"}, nil
		},
	))
	if err != nil {
		t.Fatalf("expected request handler registration to succeed, got %v", err)
	}

	err = mediator.RegisterPipelineBehavior(m, mediator.PostProcessor[advancedRequest, advancedResponse](
		func(_ context.Context, request advancedRequest, response advancedResponse) error {
			steps = append(steps, "post:"+request.Value+":"+response.Value)
			return nil
		},
	))
	if err != nil {
		t.Fatalf("expected post-processor registration to succeed, got %v", err)
	}

	response, err := mediator.Send[advancedRequest, advancedResponse](
		context.Background(),
		m,
		advancedRequest{Value: testValue},
	)
	if err != nil {
		t.Fatalf("expected send to succeed, got %v", err)
	}

	if response.Value != testValueHandled {
		t.Fatalf("expected handled response, got %q", response.Value)
	}

	expected := []string{"handler:value", "post:value:value-handled"}
	if !equalStrings(steps, expected) {
		t.Fatalf("expected steps %v, got %v", expected, steps)
	}
}

func TestPostProcessorReturnsErrorAfterSuccessfulHandler(t *testing.T) {
	m := mediator.New()
	postErr := errors.New("post failed")

	err := mediator.RegisterRequestHandler(m, mediator.RequestHandlerFunc[advancedRequest, advancedResponse](
		func(_ context.Context, request advancedRequest) (advancedResponse, error) {
			return advancedResponse{Value: request.Value + "-handled"}, nil
		},
	))
	if err != nil {
		t.Fatalf("expected request handler registration to succeed, got %v", err)
	}

	err = mediator.RegisterPipelineBehavior(m, mediator.PostProcessor[advancedRequest, advancedResponse](
		func(_ context.Context, _ advancedRequest, response advancedResponse) error {
			if response.Value != testValueHandled {
				t.Fatalf("expected post-processor to receive handler response, got %q", response.Value)
			}

			return postErr
		},
	))
	if err != nil {
		t.Fatalf("expected post-processor registration to succeed, got %v", err)
	}

	_, err = mediator.Send[advancedRequest, advancedResponse](
		context.Background(),
		m,
		advancedRequest{Value: testValue},
	)
	if err == nil {
		t.Fatal("expected post-processor error, got nil")
	}

	if !errors.Is(err, postErr) {
		t.Fatalf("expected post-processor error, got %v", err)
	}
}

func TestPostProcessorAllowsNilCallback(t *testing.T) {
	m := mediator.New()

	err := mediator.RegisterRequestHandler(m, mediator.RequestHandlerFunc[advancedRequest, advancedResponse](
		func(_ context.Context, request advancedRequest) (advancedResponse, error) {
			return advancedResponse{Value: request.Value + "-handled"}, nil
		},
	))
	if err != nil {
		t.Fatalf("expected request handler registration to succeed, got %v", err)
	}

	err = mediator.RegisterPipelineBehavior(m, mediator.PostProcessor[advancedRequest, advancedResponse](nil))
	if err != nil {
		t.Fatalf("expected post-processor registration to succeed, got %v", err)
	}

	response, err := mediator.Send[advancedRequest, advancedResponse](
		context.Background(),
		m,
		advancedRequest{Value: testValue},
	)
	if err != nil {
		t.Fatalf("expected send to succeed, got %v", err)
	}

	if response.Value != testValueHandled {
		t.Fatalf("expected handled response, got %q", response.Value)
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}

	return true
}
