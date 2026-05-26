package mediator

import (
	"context"
	"reflect"

	"github.com/cocosip/mediator/internal/typekey"
)

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

type sequentialNotificationPublisher struct{}

func (sequentialNotificationPublisher) publish(ctx context.Context, handlers []notificationExecutor, notification any) error {
	for _, handler := range handlers {
		if err := handler.handle(ctx, notification); err != nil {
			return err
		}
	}

	return nil
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

	if m.notificationPublisher == nil {
		m.notificationPublisher = sequentialNotificationPublisher{}
	}

	return nil
}

// Publish sends the notification to the registered handlers.
func Publish[TNotification any](ctx context.Context, m *Mediator, notification TNotification) error {
	key := reflect.TypeFor[TNotification]()

	m.notificationMu.RLock()
	registered := m.notificationHandlers[key]
	handlers := append([]notificationExecutor(nil), registered...)
	publisher := m.notificationPublisher
	m.notificationMu.RUnlock()

	if len(handlers) == 0 {
		return nil
	}

	if publisher == nil {
		publisher = sequentialNotificationPublisher{}
	}

	return publisher.publish(ctx, handlers, notification)
}
