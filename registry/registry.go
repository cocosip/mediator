// Package registry groups explicit mediator registrations.
package registry

import "github.com/cocosip/mediator"

// Registry stores startup registrations that can be applied together.
type Registry struct {
	registrations []func(*mediator.Mediator) error
}

// New creates an empty registry.
func New() *Registry {
	return &Registry{}
}

// AddRequestHandler adds a request handler registration.
func AddRequestHandler[TRequest any, TResponse any](
	r *Registry,
	handler mediator.RequestHandler[TRequest, TResponse],
) {
	r.add(func(m *mediator.Mediator) error {
		return mediator.RegisterRequestHandler(m, handler)
	})
}

// AddNotificationHandler adds a notification handler registration.
func AddNotificationHandler[TNotification any](
	r *Registry,
	handler mediator.NotificationHandler[TNotification],
) {
	r.add(func(m *mediator.Mediator) error {
		return mediator.RegisterNotificationHandler(m, handler)
	})
}

// AddPipelineBehavior adds a request pipeline behavior registration.
func AddPipelineBehavior[TRequest any, TResponse any](
	r *Registry,
	behavior mediator.PipelineBehavior[TRequest, TResponse],
) {
	r.add(func(m *mediator.Mediator) error {
		return mediator.RegisterPipelineBehavior(m, behavior)
	})
}

// AddStreamHandler adds a stream handler registration.
func AddStreamHandler[TRequest any, TItem any](
	r *Registry,
	handler mediator.StreamHandler[TRequest, TItem],
) {
	r.add(func(m *mediator.Mediator) error {
		return mediator.RegisterStreamHandler(m, handler)
	})
}

// Apply runs all grouped registrations in the order they were added.
func (r *Registry) Apply(m *mediator.Mediator) error {
	if r == nil {
		return nil
	}

	for _, register := range r.registrations {
		if err := register(m); err != nil {
			return err
		}
	}

	return nil
}

func (r *Registry) add(register func(*mediator.Mediator) error) {
	r.registrations = append(r.registrations, register)
}
