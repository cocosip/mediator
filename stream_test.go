package mediator_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/cocosip/mediator"
)

type streamRequest struct {
	Values []string
}

func TestStreamDispatchesRegisteredHandlerIncrementally(t *testing.T) {
	m := mediator.New()

	err := mediator.RegisterStreamHandler(m, mediator.StreamHandlerFunc[streamRequest, string](
		func(ctx context.Context, request streamRequest, yield mediator.StreamYield[string]) error {
			for _, value := range request.Values {
				if err := yield(ctx, value); err != nil {
					return err
				}
			}

			return nil
		},
	))
	if err != nil {
		t.Fatalf("expected stream handler registration to succeed, got %v", err)
	}

	var items []string
	err = mediator.Stream(
		context.Background(),
		m,
		streamRequest{Values: []string{testFirst, testSecond, "third"}},
		func(_ context.Context, item string) error {
			items = append(items, item)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("expected stream to succeed, got %v", err)
	}

	expected := []string{testFirst, testSecond, "third"}
	if !reflect.DeepEqual(items, expected) {
		t.Fatalf("expected items %v, got %v", expected, items)
	}
}

func TestStreamReturnsHandlerErrors(t *testing.T) {
	m := mediator.New()
	streamErr := errors.New("stream failed")

	err := mediator.RegisterStreamHandler(m, mediator.StreamHandlerFunc[streamRequest, string](
		func(ctx context.Context, _ streamRequest, yield mediator.StreamYield[string]) error {
			if err := yield(ctx, testFirst); err != nil {
				return err
			}

			return streamErr
		},
	))
	if err != nil {
		t.Fatalf("expected stream handler registration to succeed, got %v", err)
	}

	var items []string
	err = mediator.Stream(
		context.Background(),
		m,
		streamRequest{Values: []string{testFirst}},
		func(_ context.Context, item string) error {
			items = append(items, item)
			return nil
		},
	)
	if err == nil {
		t.Fatal("expected stream error, got nil")
	}

	if !errors.Is(err, streamErr) {
		t.Fatalf("expected stream error, got %v", err)
	}

	expected := []string{testFirst}
	if !reflect.DeepEqual(items, expected) {
		t.Fatalf("expected items %v, got %v", expected, items)
	}
}

func TestStreamReturnsContextCancellationFromYield(t *testing.T) {
	m := mediator.New()

	err := mediator.RegisterStreamHandler(m, mediator.StreamHandlerFunc[streamRequest, string](
		func(ctx context.Context, request streamRequest, yield mediator.StreamYield[string]) error {
			for _, value := range request.Values {
				if err := yield(ctx, value); err != nil {
					return err
				}
			}

			return nil
		},
	))
	if err != nil {
		t.Fatalf("expected stream handler registration to succeed, got %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var items []string
	err = mediator.Stream(
		ctx,
		m,
		streamRequest{Values: []string{testFirst, testSecond}},
		func(_ context.Context, item string) error {
			items = append(items, item)
			cancel()
			return nil
		},
	)
	if err == nil {
		t.Fatal("expected context cancellation, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	expected := []string{testFirst}
	if !reflect.DeepEqual(items, expected) {
		t.Fatalf("expected cancellation after items %v, got %v", expected, items)
	}
}

func TestStreamReturnsHandlerNotFoundForMissingHandler(t *testing.T) {
	m := mediator.New()

	err := mediator.Stream(
		context.Background(),
		m,
		streamRequest{Values: []string{testValue}},
		func(_ context.Context, _ string) error {
			return nil
		},
	)
	if err == nil {
		t.Fatal("expected missing stream handler error, got nil")
	}

	if !errors.Is(err, mediator.ErrHandlerNotFound) {
		t.Fatalf("expected ErrHandlerNotFound, got %v", err)
	}
}

func TestRegisterStreamHandlerRejectsDuplicateRegistration(t *testing.T) {
	m := mediator.New()

	handler := mediator.StreamHandlerFunc[streamRequest, string](
		func(_ context.Context, _ streamRequest, _ mediator.StreamYield[string]) error {
			return nil
		},
	)

	if err := mediator.RegisterStreamHandler(m, handler); err != nil {
		t.Fatalf("expected first registration to succeed, got %v", err)
	}

	err := mediator.RegisterStreamHandler(m, handler)
	if err == nil {
		t.Fatal("expected duplicate stream handler error, got nil")
	}

	if !errors.Is(err, mediator.ErrDuplicateHandler) {
		t.Fatalf("expected ErrDuplicateHandler, got %v", err)
	}
}

func TestRegisterStreamHandlerRejectsNilHandler(t *testing.T) {
	m := mediator.New()

	err := mediator.RegisterStreamHandler[streamRequest, string](m, nil)
	if err == nil {
		t.Fatal("expected invalid handler error, got nil")
	}

	if !errors.Is(err, mediator.ErrInvalidHandler) {
		t.Fatalf("expected ErrInvalidHandler, got %v", err)
	}
}

func TestRegisterStreamHandlerRejectsTypedNilFunctionHandler(t *testing.T) {
	m := mediator.New()
	var handler mediator.StreamHandlerFunc[streamRequest, string]

	err := mediator.RegisterStreamHandler(m, handler)
	if err == nil {
		t.Fatal("expected invalid handler error, got nil")
	}

	if !errors.Is(err, mediator.ErrInvalidHandler) {
		t.Fatalf("expected ErrInvalidHandler, got %v", err)
	}
}

func TestStreamReturnsContextErrorBeforeDispatch(t *testing.T) {
	m := mediator.New()
	handlerCalled := false

	err := mediator.RegisterStreamHandler(m, mediator.StreamHandlerFunc[streamRequest, string](
		func(_ context.Context, _ streamRequest, _ mediator.StreamYield[string]) error {
			handlerCalled = true
			return nil
		},
	))
	if err != nil {
		t.Fatalf("expected stream handler registration to succeed, got %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = mediator.Stream[streamRequest, string](ctx, m, streamRequest{Values: []string{testFirst}}, nil)
	if err == nil {
		t.Fatal("expected context cancellation, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	if handlerCalled {
		t.Fatal("expected handler not to run when context is already canceled")
	}
}

func TestStreamAllowsNilYieldCallback(t *testing.T) {
	m := mediator.New()

	err := mediator.RegisterStreamHandler(m, mediator.StreamHandlerFunc[streamRequest, string](
		func(ctx context.Context, request streamRequest, yield mediator.StreamYield[string]) error {
			for _, value := range request.Values {
				if err := yield(ctx, value); err != nil {
					return err
				}
			}

			return nil
		},
	))
	if err != nil {
		t.Fatalf("expected stream handler registration to succeed, got %v", err)
	}

	err = mediator.Stream[streamRequest, string](
		context.Background(),
		m,
		streamRequest{Values: []string{testFirst, testSecond}},
		nil,
	)
	if err != nil {
		t.Fatalf("expected nil yield callback to be allowed, got %v", err)
	}
}

func TestStreamReturnsYieldError(t *testing.T) {
	m := mediator.New()
	yieldErr := errors.New("yield failed")

	err := mediator.RegisterStreamHandler(m, mediator.StreamHandlerFunc[streamRequest, string](
		func(ctx context.Context, _ streamRequest, yield mediator.StreamYield[string]) error {
			return yield(ctx, testFirst)
		},
	))
	if err != nil {
		t.Fatalf("expected stream handler registration to succeed, got %v", err)
	}

	err = mediator.Stream(
		context.Background(),
		m,
		streamRequest{Values: []string{testFirst}},
		func(_ context.Context, _ string) error {
			return yieldErr
		},
	)
	if err == nil {
		t.Fatal("expected yield error, got nil")
	}

	if !errors.Is(err, yieldErr) {
		t.Fatalf("expected yield error, got %v", err)
	}
}

func TestStreamReturnsYieldContextErrorBeforeCallbackRuns(t *testing.T) {
	m := mediator.New()
	yieldCalled := false

	err := mediator.RegisterStreamHandler(m, mediator.StreamHandlerFunc[streamRequest, string](
		func(ctx context.Context, _ streamRequest, yield mediator.StreamYield[string]) error {
			yieldCtx, cancel := context.WithCancel(ctx)
			cancel()
			return yield(yieldCtx, testFirst)
		},
	))
	if err != nil {
		t.Fatalf("expected stream handler registration to succeed, got %v", err)
	}

	err = mediator.Stream(
		context.Background(),
		m,
		streamRequest{Values: []string{testFirst}},
		func(_ context.Context, _ string) error {
			yieldCalled = true
			return nil
		},
	)
	if err == nil {
		t.Fatal("expected context cancellation from yield context, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	if yieldCalled {
		t.Fatal("expected canceled yield context to prevent callback execution")
	}
}
