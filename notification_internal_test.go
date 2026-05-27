package mediator

import (
	"context"
	"testing"
)

type internalNotification struct {
	ID string
}

func TestPublishFallsBackToSequentialPublisherWhenFieldIsNil(t *testing.T) {
	m := New()
	m.notificationPublisher = nil
	callCount := 0

	err := RegisterNotificationHandler(m, NotificationHandlerFunc[internalNotification](
		func(_ context.Context, _ internalNotification) error {
			callCount++
			return nil
		},
	))
	if err != nil {
		t.Fatalf("expected registration to succeed, got %v", err)
	}

	err = Publish(context.Background(), m, internalNotification{ID: "internal-1"})
	if err != nil {
		t.Fatalf("expected fallback sequential publisher to run, got %v", err)
	}

	if callCount != 1 {
		t.Fatalf("expected handler to run once, got %d", callCount)
	}
}
