package mediator

import "context"

// Sender dispatches a typed request through a mediator runtime.
type Sender[TRequest any, TResponse any] interface {
	Send(context.Context, TRequest) (TResponse, error)
}

// Publisher publishes a typed notification through a mediator runtime.
type Publisher[TNotification any] interface {
	Publish(context.Context, TNotification) error
}

// RequestRegistrar registers a typed request handler.
type RequestRegistrar[TRequest any, TResponse any] interface {
	Register(RequestHandler[TRequest, TResponse]) error
}

// NotificationRegistrar registers a typed notification handler.
type NotificationRegistrar[TNotification any] interface {
	Register(NotificationHandler[TNotification]) error
}

// BehaviorRegistrar registers a typed request pipeline behavior.
type BehaviorRegistrar[TRequest any, TResponse any] interface {
	Register(PipelineBehavior[TRequest, TResponse]) error
}

type requestSenderFacade[TRequest any, TResponse any] struct {
	mediator *Mediator
}

func (f requestSenderFacade[TRequest, TResponse]) Send(
	ctx context.Context,
	request TRequest,
) (TResponse, error) {
	return Send[TRequest, TResponse](ctx, f.mediator, request)
}

type notificationPublisherFacade[TNotification any] struct {
	mediator *Mediator
}

func (f notificationPublisherFacade[TNotification]) Publish(
	ctx context.Context,
	notification TNotification,
) error {
	return Publish(ctx, f.mediator, notification)
}

type requestRegistrationFacade[TRequest any, TResponse any] struct {
	mediator *Mediator
}

func (f requestRegistrationFacade[TRequest, TResponse]) Register(
	handler RequestHandler[TRequest, TResponse],
) error {
	return RegisterRequestHandler(f.mediator, handler)
}

type notificationRegistrationFacade[TNotification any] struct {
	mediator *Mediator
}

func (f notificationRegistrationFacade[TNotification]) Register(
	handler NotificationHandler[TNotification],
) error {
	return RegisterNotificationHandler(f.mediator, handler)
}

type behaviorRegistrationFacade[TRequest any, TResponse any] struct {
	mediator *Mediator
}

func (f behaviorRegistrationFacade[TRequest, TResponse]) Register(
	behavior PipelineBehavior[TRequest, TResponse],
) error {
	return RegisterPipelineBehavior(f.mediator, behavior)
}

// RequestSender projects a mediator into a typed request-sending capability.
func RequestSender[TRequest any, TResponse any](m *Mediator) Sender[TRequest, TResponse] {
	return requestSenderFacade[TRequest, TResponse]{mediator: m}
}

// NotificationPublisherOf projects a mediator into a typed notification-publishing capability.
func NotificationPublisherOf[TNotification any](m *Mediator) Publisher[TNotification] {
	return notificationPublisherFacade[TNotification]{mediator: m}
}

// RequestRegistration projects a mediator into a typed request-registration capability.
func RequestRegistration[TRequest any, TResponse any](m *Mediator) RequestRegistrar[TRequest, TResponse] {
	return requestRegistrationFacade[TRequest, TResponse]{mediator: m}
}

// NotificationRegistration projects a mediator into a typed notification-registration capability.
func NotificationRegistration[TNotification any](m *Mediator) NotificationRegistrar[TNotification] {
	return notificationRegistrationFacade[TNotification]{mediator: m}
}

// BehaviorRegistration projects a mediator into a typed behavior-registration capability.
func BehaviorRegistration[TRequest any, TResponse any](m *Mediator) BehaviorRegistrar[TRequest, TResponse] {
	return behaviorRegistrationFacade[TRequest, TResponse]{mediator: m}
}
