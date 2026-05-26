package mediator_test

import (
	"context"
	"errors"
	"testing"
	"time"

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

type recordingNotificationPublisher struct {
	called       bool
	handlers     []mediator.NotificationExecutor
	notification any
}

func (p *recordingNotificationPublisher) Publish(
	ctx context.Context,
	handlers []mediator.NotificationExecutor,
	notification any,
) error {
	p.called = true
	p.handlers = append([]mediator.NotificationExecutor(nil), handlers...)
	p.notification = notification

	if len(handlers) != 1 {
		return errors.New("unexpected handler count")
	}

	return handlers[0].Handle(ctx, notification)
}

func TestWithNotificationPublisherUsesCustomPublisher(t *testing.T) {
	publisher := &recordingNotificationPublisher{}
	m := mediator.New(mediator.WithNotificationPublisher(publisher))
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

	notification := userCreatedNotification{ID: "user-1"}
	err = mediator.Publish(context.Background(), m, notification)
	if err != nil {
		t.Fatalf("expected publish to succeed, got %v", err)
	}

	if !publisher.called {
		t.Fatal("expected custom publisher to be used")
	}

	if len(publisher.handlers) != 1 {
		t.Fatalf("expected custom publisher to receive 1 handler, got %d", len(publisher.handlers))
	}

	if publisher.notification != notification {
		t.Fatalf("expected custom publisher to receive notification %#v, got %#v", notification, publisher.notification)
	}

	if callCount != 1 {
		t.Fatalf("expected handler to be executed once, got %d", callCount)
	}
}

func TestSequentialPublisherContinueOnErrorAggregatesFailures(t *testing.T) {
	m := mediator.New(mediator.WithNotificationPublisher(mediator.SequentialPublisher{
		ErrorStrategy: mediator.ContinueOnError,
	}))
	firstErr := errors.New("first failed")
	secondErr := errors.New("second failed")
	callCount := 0

	err := mediator.RegisterNotificationHandler(m, mediator.NotificationHandlerFunc[userCreatedNotification](
		func(ctx context.Context, notification userCreatedNotification) error {
			callCount++
			return firstErr
		},
	))
	if err != nil {
		t.Fatalf("expected first registration to succeed, got %v", err)
	}

	err = mediator.RegisterNotificationHandler(m, mediator.NotificationHandlerFunc[userCreatedNotification](
		func(ctx context.Context, notification userCreatedNotification) error {
			callCount++
			return secondErr
		},
	))
	if err != nil {
		t.Fatalf("expected second registration to succeed, got %v", err)
	}

	err = mediator.Publish(context.Background(), m, userCreatedNotification{ID: "user-1"})
	if err == nil {
		t.Fatal("expected aggregated error, got nil")
	}

	if !errors.Is(err, firstErr) {
		t.Fatalf("expected aggregated error to include first error, got %v", err)
	}

	if !errors.Is(err, secondErr) {
		t.Fatalf("expected aggregated error to include second error, got %v", err)
	}

	if callCount != 2 {
		t.Fatalf("expected both handlers to run, got %d calls", callCount)
	}
}

func TestParallelPublisherRunsHandlersConcurrently(t *testing.T) {
	m := mediator.New(mediator.WithNotificationPublisher(mediator.ParallelPublisher{}))
	started := make(chan string, 2)
	release := make(chan struct{})
	done := make(chan error, 1)

	err := mediator.RegisterNotificationHandler(m, mediator.NotificationHandlerFunc[userCreatedNotification](
		func(ctx context.Context, notification userCreatedNotification) error {
			started <- "first"
			<-release
			return nil
		},
	))
	if err != nil {
		t.Fatalf("expected first registration to succeed, got %v", err)
	}

	err = mediator.RegisterNotificationHandler(m, mediator.NotificationHandlerFunc[userCreatedNotification](
		func(ctx context.Context, notification userCreatedNotification) error {
			started <- "second"
			<-release
			return nil
		},
	))
	if err != nil {
		t.Fatalf("expected second registration to succeed, got %v", err)
	}

	go func() {
		done <- mediator.Publish(context.Background(), m, userCreatedNotification{ID: "user-1"})
	}()

	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("expected both handlers to start before release")
		}
	}

	close(release)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected parallel publish to succeed, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("expected parallel publish to complete")
	}
}

func TestParallelPublisherContinueOnErrorAggregatesFailures(t *testing.T) {
	m := mediator.New(mediator.WithNotificationPublisher(mediator.ParallelPublisher{
		ErrorStrategy: mediator.ContinueOnError,
	}))
	firstErr := errors.New("first failed")
	secondErr := errors.New("second failed")

	err := mediator.RegisterNotificationHandler(m, mediator.NotificationHandlerFunc[userCreatedNotification](
		func(ctx context.Context, notification userCreatedNotification) error {
			return firstErr
		},
	))
	if err != nil {
		t.Fatalf("expected first registration to succeed, got %v", err)
	}

	err = mediator.RegisterNotificationHandler(m, mediator.NotificationHandlerFunc[userCreatedNotification](
		func(ctx context.Context, notification userCreatedNotification) error {
			return secondErr
		},
	))
	if err != nil {
		t.Fatalf("expected second registration to succeed, got %v", err)
	}

	err = mediator.Publish(context.Background(), m, userCreatedNotification{ID: "user-1"})
	if err == nil {
		t.Fatal("expected aggregated error, got nil")
	}

	if !errors.Is(err, firstErr) {
		t.Fatalf("expected aggregated error to include first error, got %v", err)
	}

	if !errors.Is(err, secondErr) {
		t.Fatalf("expected aggregated error to include second error, got %v", err)
	}
}

func TestParallelPublisherReturnsContextErrorOnCancellation(t *testing.T) {
	m := mediator.New(mediator.WithNotificationPublisher(mediator.ParallelPublisher{}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{}, 2)
	done := make(chan error, 1)

	handler := mediator.NotificationHandlerFunc[userCreatedNotification](
		func(ctx context.Context, notification userCreatedNotification) error {
			started <- struct{}{}
			<-ctx.Done()
			return ctx.Err()
		},
	)

	err := mediator.RegisterNotificationHandler(m, handler)
	if err != nil {
		t.Fatalf("expected first registration to succeed, got %v", err)
	}

	err = mediator.RegisterNotificationHandler(m, handler)
	if err != nil {
		t.Fatalf("expected second registration to succeed, got %v", err)
	}

	go func() {
		done <- mediator.Publish(ctx, m, userCreatedNotification{ID: "user-1"})
	}()

	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("expected both handlers to start")
		}
	}

	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context canceled error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("expected parallel publish to return after cancellation")
	}
}
