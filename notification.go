package mediator

import (
	"context"
	"errors"
	"reflect"
	"sync"

	"github.com/cocosip/mediator/internal/typekey"
)

// NotificationErrorStrategy controls how publishers handle handler errors.
type NotificationErrorStrategy int

const (
	StopOnFirstError NotificationErrorStrategy = iota
	ContinueOnError
)

// NotificationExecutor adapts a registered notification handler for publishers.
type NotificationExecutor interface {
	Handle(context.Context, any) error
}

// NotificationPublisher controls notification execution strategy.
type NotificationPublisher interface {
	Publish(context.Context, []NotificationExecutor, any) error
}

// SequentialPublisher executes handlers in registration order.
type SequentialPublisher struct {
	ErrorStrategy NotificationErrorStrategy
}

// ParallelPublisher executes handlers concurrently.
type ParallelPublisher struct {
	ErrorStrategy NotificationErrorStrategy
}

// WithNotificationPublisher replaces the mediator notification publisher.
func WithNotificationPublisher(publisher NotificationPublisher) Option {
	return func(m *Mediator) {
		if publisher == nil {
			m.notificationPublisher = SequentialPublisher{
				ErrorStrategy: StopOnFirstError,
			}
			return
		}

		m.notificationPublisher = publisher
	}
}

// NotificationHandler handles a notification.
type NotificationHandler[TNotification any] interface {
	Handle(context.Context, TNotification) error
}

// NotificationHandlerFunc adapts a function into a NotificationHandler.
type NotificationHandlerFunc[TNotification any] func(context.Context, TNotification) error

// Handle implements NotificationHandler.
func (f NotificationHandlerFunc[TNotification]) Handle(ctx context.Context, notification TNotification) error {
	return f(ctx, notification)
}

type notificationHandlerAdapter[TNotification any] struct {
	handler NotificationHandler[TNotification]
}

func (a notificationHandlerAdapter[TNotification]) handle(ctx context.Context, notification any) error {
	return a.handler.Handle(ctx, notification.(TNotification))
}

// Handle implements NotificationExecutor.
func (a notificationHandlerAdapter[TNotification]) Handle(ctx context.Context, notification any) error {
	return a.handle(ctx, notification)
}

// Publish implements NotificationPublisher.
func (p SequentialPublisher) Publish(ctx context.Context, handlers []NotificationExecutor, notification any) error {
	var errs []error

	for _, handler := range handlers {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := handler.Handle(ctx, notification)
		if err == nil {
			continue
		}

		if p.ErrorStrategy != ContinueOnError {
			return err
		}

		errs = append(errs, err)
	}

	if len(errs) == 0 {
		return nil
	}

	return errors.Join(errs...)
}

// Publish implements NotificationPublisher.
func (p ParallelPublisher) Publish(ctx context.Context, handlers []NotificationExecutor, notification any) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	results := make(chan error, len(handlers))
	var wg sync.WaitGroup

	for _, handler := range handlers {
		if err := ctx.Err(); err != nil {
			return err
		}

		wg.Add(1)
		go func(handler NotificationExecutor) {
			defer wg.Done()
			results <- handler.Handle(ctx, notification)
		}(handler)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var errs []error
	for err := range results {
		if err == nil {
			continue
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}

		if p.ErrorStrategy != ContinueOnError {
			return err
		}

		errs = append(errs, err)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if len(errs) == 0 {
		return nil
	}

	return errors.Join(errs...)
}

// RegisterNotificationHandler registers a notification handler.
func RegisterNotificationHandler[TNotification any](m *Mediator, handler NotificationHandler[TNotification]) error {
	if handler == nil {
		return InvalidHandlerError{
			Kind:        "notification",
			MessageType: typekey.Of[TNotification](),
		}
	}

	key := typekey.Of[TNotification]()

	m.notificationMu.Lock()
	defer m.notificationMu.Unlock()

	m.notificationHandlers[key] = append(
		m.notificationHandlers[key],
		notificationHandlerAdapter[TNotification]{handler: handler},
	)

	return nil
}

// Publish sends the notification to the registered handlers.
func Publish[TNotification any](ctx context.Context, m *Mediator, notification TNotification) error {
	key := reflect.TypeFor[TNotification]()

	m.notificationMu.RLock()
	registered := m.notificationHandlers[key]
	handlers := make([]NotificationExecutor, 0, len(registered))
	for _, handler := range registered {
		handlers = append(handlers, handler)
	}
	publisher := m.notificationPublisher
	m.notificationMu.RUnlock()

	if len(handlers) == 0 {
		return nil
	}

	if publisher == nil {
		publisher = SequentialPublisher{
			ErrorStrategy: StopOnFirstError,
		}
	}

	return publisher.Publish(ctx, handlers, notification)
}
