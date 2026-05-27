package registry_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cocosip/mediator"
	"github.com/cocosip/mediator/registry"
)

type pingRequest struct {
	Message string
}

type pingNotification struct {
	Message string
}

func TestRegistryAppliesGroupedRegistrations(t *testing.T) {
	m := mediator.New()
	r := registry.New()

	registry.AddRequestHandler(r, mediator.RequestHandlerFunc[pingRequest, string](
		func(_ context.Context, request pingRequest) (string, error) {
			return "pong:" + request.Message, nil
		},
	))

	notifications := make([]string, 0, 1)
	registry.AddNotificationHandler(r, mediator.NotificationHandlerFunc[pingNotification](
		func(_ context.Context, notification pingNotification) error {
			notifications = append(notifications, notification.Message)
			return nil
		},
	))

	if err := r.Apply(m); err != nil {
		t.Fatalf("expected grouped registration to succeed, got %v", err)
	}

	response, err := mediator.Send[pingRequest, string](
		context.Background(),
		m,
		pingRequest{Message: testHello},
	)
	if err != nil {
		t.Fatalf("expected send to succeed, got %v", err)
	}

	if response != "pong:hello" {
		t.Fatalf("expected response pong:hello, got %q", response)
	}

	err = mediator.Publish(context.Background(), m, pingNotification{Message: "created"})
	if err != nil {
		t.Fatalf("expected publish to succeed, got %v", err)
	}

	if len(notifications) != 1 || notifications[0] != "created" {
		t.Fatalf("expected notification to be handled once, got %v", notifications)
	}
}

func TestRegistryAppliesPipelineBehaviorRegistrations(t *testing.T) {
	m := mediator.New()
	r := registry.New()

	registry.AddRequestHandler(r, mediator.RequestHandlerFunc[pingRequest, string](
		func(_ context.Context, request pingRequest) (string, error) {
			return request.Message, nil
		},
	))

	registry.AddPipelineBehavior(r, mediator.PipelineBehaviorFunc[pingRequest, string](
		func(ctx context.Context, _ pingRequest, next mediator.RequestHandlerDelegate[string]) (string, error) {
			response, err := next(ctx)
			if err != nil {
				return "", err
			}

			return "wrapped:" + response, nil
		},
	))

	if err := r.Apply(m); err != nil {
		t.Fatalf("expected grouped registration to succeed, got %v", err)
	}

	response, err := mediator.Send[pingRequest, string](
		context.Background(),
		m,
		pingRequest{Message: testHello},
	)
	if err != nil {
		t.Fatalf("expected send to succeed, got %v", err)
	}

	if response != "wrapped:hello" {
		t.Fatalf("expected wrapped response, got %q", response)
	}
}

func TestRegistryAppliesStreamHandlerRegistrations(t *testing.T) {
	m := mediator.New()
	r := registry.New()

	registry.AddStreamHandler(r, mediator.StreamHandlerFunc[pingRequest, string](
		func(ctx context.Context, request pingRequest, yield mediator.StreamYield[string]) error {
			return yield(ctx, request.Message)
		},
	))

	if err := r.Apply(m); err != nil {
		t.Fatalf("expected grouped registration to succeed, got %v", err)
	}

	var items []string
	err := mediator.Stream(
		context.Background(),
		m,
		pingRequest{Message: testHello},
		func(_ context.Context, item string) error {
			items = append(items, item)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("expected stream to succeed, got %v", err)
	}

	if len(items) != 1 || items[0] != testHello {
		t.Fatalf("expected streamed item hello, got %v", items)
	}
}

func TestRegistryReturnsRegistrationErrors(t *testing.T) {
	m := mediator.New()
	r := registry.New()

	handler := mediator.RequestHandlerFunc[pingRequest, string](
		func(_ context.Context, request pingRequest) (string, error) {
			return request.Message, nil
		},
	)

	registry.AddRequestHandler(r, handler)
	registry.AddRequestHandler(r, handler)

	err := r.Apply(m)
	if err == nil {
		t.Fatal("expected duplicate registration error, got nil")
	}

	if !errors.Is(err, mediator.ErrDuplicateHandler) {
		t.Fatalf("expected ErrDuplicateHandler, got %v", err)
	}
}
