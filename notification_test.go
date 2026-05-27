package mediator_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/cocosip/mediator"
)

type userCreatedNotification struct {
	ID string
}

func TestPublishReturnsNilWhenNoHandlersAreRegistered(t *testing.T) {
	m := mediator.New()

	err := mediator.Publish(context.Background(), m, userCreatedNotification{ID: testUserID})
	if err != nil {
		t.Fatalf("expected publish with no handlers to succeed, got %v", err)
	}
}

func TestPublishRunsHandlersInRegistrationOrder(t *testing.T) {
	m := mediator.New()
	var calls []string

	err := mediator.RegisterNotificationHandler(m, mediator.NotificationHandlerFunc[userCreatedNotification](
		func(_ context.Context, notification userCreatedNotification) error {
			calls = append(calls, "first:"+notification.ID)
			return nil
		},
	))
	if err != nil {
		t.Fatalf("expected first registration to succeed, got %v", err)
	}

	err = mediator.RegisterNotificationHandler(m, mediator.NotificationHandlerFunc[userCreatedNotification](
		func(_ context.Context, notification userCreatedNotification) error {
			calls = append(calls, "second:"+notification.ID)
			return nil
		},
	))
	if err != nil {
		t.Fatalf("expected second registration to succeed, got %v", err)
	}

	err = mediator.Publish(context.Background(), m, userCreatedNotification{ID: testUserID})
	if err != nil {
		t.Fatalf("expected publish to succeed, got %v", err)
	}

	want := []string{"first:user-1", "second:user-1"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("expected calls %v, got %v", want, calls)
	}
}

func TestPublishStopsOnFirstHandlerError(t *testing.T) {
	m := mediator.New()
	boom := errors.New("boom")
	calledSecond := false

	err := mediator.RegisterNotificationHandler(m, mediator.NotificationHandlerFunc[userCreatedNotification](
		func(_ context.Context, _ userCreatedNotification) error {
			return boom
		},
	))
	if err != nil {
		t.Fatalf("expected first registration to succeed, got %v", err)
	}

	err = mediator.RegisterNotificationHandler(m, mediator.NotificationHandlerFunc[userCreatedNotification](
		func(_ context.Context, _ userCreatedNotification) error {
			calledSecond = true
			return nil
		},
	))
	if err != nil {
		t.Fatalf("expected second registration to succeed, got %v", err)
	}

	err = mediator.Publish(context.Background(), m, userCreatedNotification{ID: testUserID})
	if !errors.Is(err, boom) {
		t.Fatalf("expected boom error, got %v", err)
	}

	if calledSecond {
		t.Fatal("expected publish to stop before second handler")
	}
}

func TestRegisterNotificationHandlerRejectsNilHandler(t *testing.T) {
	m := mediator.New()

	err := mediator.RegisterNotificationHandler[userCreatedNotification](m, nil)
	if err == nil {
		t.Fatal("expected invalid handler error, got nil")
	}

	if !errors.Is(err, mediator.ErrInvalidHandler) {
		t.Fatalf("expected ErrInvalidHandler, got %v", err)
	}
}
