package mediator_test

import (
	"context"
	"testing"

	"github.com/cocosip/mediator"
)

func TestRequestSenderDispatchesRegisteredRequestHandler(t *testing.T) {
	m := mediator.New()

	err := mediator.RegisterRequestHandler(m, mediator.RequestHandlerFunc[createUserRequest, createUserResponse](
		func(ctx context.Context, request createUserRequest) (createUserResponse, error) {
			return createUserResponse{ID: "sender-" + request.Name}, nil
		},
	))
	if err != nil {
		t.Fatalf("expected registration to succeed, got %v", err)
	}

	sender := mediator.RequestSender[createUserRequest, createUserResponse](m)
	response, err := sender.Send(context.Background(), createUserRequest{Name: "alice"})
	if err != nil {
		t.Fatalf("expected sender to dispatch request, got %v", err)
	}

	if response.ID != "sender-alice" {
		t.Fatalf("expected sender response id sender-alice, got %q", response.ID)
	}
}

func TestNotificationPublisherOfPublishesNotification(t *testing.T) {
	m := mediator.New()
	callCount := 0

	err := mediator.RegisterNotificationHandler(m, mediator.NotificationHandlerFunc[userCreatedNotification](
		func(ctx context.Context, notification userCreatedNotification) error {
			callCount++
			return nil
		},
	))
	if err != nil {
		t.Fatalf("expected registration to succeed, got %v", err)
	}

	publisher := mediator.NotificationPublisherOf[userCreatedNotification](m)
	err = publisher.Publish(context.Background(), userCreatedNotification{ID: "user-1"})
	if err != nil {
		t.Fatalf("expected publisher facade to publish notification, got %v", err)
	}

	if callCount != 1 {
		t.Fatalf("expected publisher facade to call handler once, got %d", callCount)
	}
}

func TestRequestRegistrationRegistersRequestHandler(t *testing.T) {
	m := mediator.New()
	registration := mediator.RequestRegistration[createUserRequest, createUserResponse](m)

	err := registration.Register(mediator.RequestHandlerFunc[createUserRequest, createUserResponse](
		func(ctx context.Context, request createUserRequest) (createUserResponse, error) {
			return createUserResponse{ID: request.Name}, nil
		},
	))
	if err != nil {
		t.Fatalf("expected request registration facade to succeed, got %v", err)
	}

	response, err := mediator.Send[createUserRequest, createUserResponse](
		context.Background(),
		m,
		createUserRequest{Name: "registered"},
	)
	if err != nil {
		t.Fatalf("expected registered request handler to dispatch, got %v", err)
	}

	if response.ID != "registered" {
		t.Fatalf("expected response id registered, got %q", response.ID)
	}
}

func TestNotificationRegistrationRegistersNotificationHandler(t *testing.T) {
	m := mediator.New()
	callCount := 0
	registration := mediator.NotificationRegistration[userCreatedNotification](m)

	err := registration.Register(mediator.NotificationHandlerFunc[userCreatedNotification](
		func(ctx context.Context, notification userCreatedNotification) error {
			callCount++
			return nil
		},
	))
	if err != nil {
		t.Fatalf("expected notification registration facade to succeed, got %v", err)
	}

	err = mediator.Publish(context.Background(), m, userCreatedNotification{ID: "user-1"})
	if err != nil {
		t.Fatalf("expected publish to succeed, got %v", err)
	}

	if callCount != 1 {
		t.Fatalf("expected notification handler to run once, got %d", callCount)
	}
}

func TestBehaviorRegistrationRegistersPipelineBehavior(t *testing.T) {
	m := mediator.New()
	registration := mediator.BehaviorRegistration[behaviorRequest, behaviorResponse](m)

	err := mediator.RegisterRequestHandler(m, mediator.RequestHandlerFunc[behaviorRequest, behaviorResponse](
		func(ctx context.Context, request behaviorRequest) (behaviorResponse, error) {
			return behaviorResponse{Value: request.Value}, nil
		},
	))
	if err != nil {
		t.Fatalf("expected request handler registration to succeed, got %v", err)
	}

	err = registration.Register(mediator.PipelineBehaviorFunc[behaviorRequest, behaviorResponse](
		func(ctx context.Context, request behaviorRequest, next mediator.RequestHandlerDelegate[behaviorResponse]) (behaviorResponse, error) {
			response, err := next(ctx)
			if err != nil {
				return behaviorResponse{}, err
			}

			return behaviorResponse{Value: "wrapped:" + response.Value}, nil
		},
	))
	if err != nil {
		t.Fatalf("expected behavior registration facade to succeed, got %v", err)
	}

	response, err := mediator.Send[behaviorRequest, behaviorResponse](
		context.Background(),
		m,
		behaviorRequest{Value: "value"},
	)
	if err != nil {
		t.Fatalf("expected send to succeed, got %v", err)
	}

	if response.Value != "wrapped:value" {
		t.Fatalf("expected wrapped response, got %q", response.Value)
	}
}
