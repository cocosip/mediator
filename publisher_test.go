package mediator_test

import (
	"context"
	"testing"

	"github.com/cocosip/mediator"
)

func TestDefaultSequentialPublisherPublishesThroughMediator(t *testing.T) {
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

	err = mediator.Publish(context.Background(), m, userCreatedNotification{ID: "user-1"})
	if err != nil {
		t.Fatalf("expected publish to succeed, got %v", err)
	}

	if callCount != 1 {
		t.Fatalf("expected call count 1, got %d", callCount)
	}
}
