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
		func(_ context.Context, _ userCreatedNotification) error {
			callCount++
			return nil
		},
	))
	if err != nil {
		t.Fatalf("expected registration to succeed, got %v", err)
	}

	err = mediator.Publish(context.Background(), m, userCreatedNotification{ID: testUserID})
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
		func(_ context.Context, _ userCreatedNotification) error {
			callCount++
			return nil
		},
	))
	if err != nil {
		t.Fatalf("expected registration to succeed, got %v", err)
	}

	notification := userCreatedNotification{ID: testUserID}
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
		func(_ context.Context, _ userCreatedNotification) error {
			callCount++
			return firstErr
		},
	))
	if err != nil {
		t.Fatalf("expected first registration to succeed, got %v", err)
	}

	err = mediator.RegisterNotificationHandler(m, mediator.NotificationHandlerFunc[userCreatedNotification](
		func(_ context.Context, _ userCreatedNotification) error {
			callCount++
			return secondErr
		},
	))
	if err != nil {
		t.Fatalf("expected second registration to succeed, got %v", err)
	}

	err = mediator.Publish(context.Background(), m, userCreatedNotification{ID: testUserID})
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
		func(_ context.Context, _ userCreatedNotification) error {
			started <- testFirst
			<-release
			return nil
		},
	))
	if err != nil {
		t.Fatalf("expected first registration to succeed, got %v", err)
	}

	err = mediator.RegisterNotificationHandler(m, mediator.NotificationHandlerFunc[userCreatedNotification](
		func(_ context.Context, _ userCreatedNotification) error {
			started <- "second"
			<-release
			return nil
		},
	))
	if err != nil {
		t.Fatalf("expected second registration to succeed, got %v", err)
	}

	go func() {
		done <- mediator.Publish(context.Background(), m, userCreatedNotification{ID: testUserID})
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
		func(_ context.Context, _ userCreatedNotification) error {
			return firstErr
		},
	))
	if err != nil {
		t.Fatalf("expected first registration to succeed, got %v", err)
	}

	err = mediator.RegisterNotificationHandler(m, mediator.NotificationHandlerFunc[userCreatedNotification](
		func(_ context.Context, _ userCreatedNotification) error {
			return secondErr
		},
	))
	if err != nil {
		t.Fatalf("expected second registration to succeed, got %v", err)
	}

	err = mediator.Publish(context.Background(), m, userCreatedNotification{ID: testUserID})
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
		func(ctx context.Context, _ userCreatedNotification) error {
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
		done <- mediator.Publish(ctx, m, userCreatedNotification{ID: testUserID})
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

type stubNotificationExecutor func(context.Context, any) error

func (f stubNotificationExecutor) Handle(ctx context.Context, notification any) error {
	return f(ctx, notification)
}

func TestWithNotificationPublisherNilRestoresDefaultSequentialPublisher(t *testing.T) {
	m := mediator.New(mediator.WithNotificationPublisher(nil))
	callCount := 0

	err := mediator.RegisterNotificationHandler(m, mediator.NotificationHandlerFunc[userCreatedNotification](
		func(_ context.Context, _ userCreatedNotification) error {
			callCount++
			return nil
		},
	))
	if err != nil {
		t.Fatalf("expected registration to succeed, got %v", err)
	}

	err = mediator.Publish(context.Background(), m, userCreatedNotification{ID: testUserID})
	if err != nil {
		t.Fatalf("expected default sequential publisher to run, got %v", err)
	}

	if callCount != 1 {
		t.Fatalf("expected handler to run once, got %d", callCount)
	}
}

func TestSequentialPublisherReturnsContextErrorBeforeHandling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false

	err := (mediator.SequentialPublisher{}).Publish(
		ctx,
		[]mediator.NotificationExecutor{
			stubNotificationExecutor(func(context.Context, any) error {
				called = true
				return nil
			}),
		},
		userCreatedNotification{ID: testUserID},
	)
	if err == nil {
		t.Fatal("expected context cancellation, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	if called {
		t.Fatal("expected canceled context to stop handler execution")
	}
}

func TestParallelPublisherReturnsContextErrorBeforeStartingHandlers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false

	err := (mediator.ParallelPublisher{}).Publish(
		ctx,
		[]mediator.NotificationExecutor{
			stubNotificationExecutor(func(context.Context, any) error {
				called = true
				return nil
			}),
		},
		userCreatedNotification{ID: testUserID},
	)
	if err == nil {
		t.Fatal("expected context cancellation, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	if called {
		t.Fatal("expected canceled context to stop parallel handler startup")
	}
}

func TestParallelPublisherStopOnFirstErrorReturnsHandlerError(t *testing.T) {
	boom := errors.New("boom")

	err := (mediator.ParallelPublisher{}).Publish(
		context.Background(),
		[]mediator.NotificationExecutor{
			stubNotificationExecutor(func(context.Context, any) error {
				return boom
			}),
		},
		userCreatedNotification{ID: testUserID},
	)
	if err == nil {
		t.Fatal("expected handler error, got nil")
	}

	if !errors.Is(err, boom) {
		t.Fatalf("expected handler error, got %v", err)
	}
}

func TestParallelPublisherReturnsContextErrorAfterHandlersFinish(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := (mediator.ParallelPublisher{ErrorStrategy: mediator.ContinueOnError}).Publish(
		ctx,
		[]mediator.NotificationExecutor{
			stubNotificationExecutor(func(context.Context, any) error {
				cancel()
				return nil
			}),
			stubNotificationExecutor(func(context.Context, any) error {
				return nil
			}),
		},
		userCreatedNotification{ID: testUserID},
	)
	if err == nil {
		t.Fatal("expected context cancellation after handlers finish, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
