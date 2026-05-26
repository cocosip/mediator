package mediator

import (
	"context"
	"reflect"
	"sync"

	"github.com/cocosip/mediator/internal/typekey"
)

// Option mutates a mediator instance during construction.
type Option func(*Mediator)

// Mediator stores request and notification registrations.
type Mediator struct {
	requestMu       sync.RWMutex
	requestHandlers map[typekey.Pair]requestExecutor

	notificationMu        sync.RWMutex
	notificationHandlers  map[reflect.Type][]notificationExecutor
	notificationPublisher notificationPublisher
}

type requestExecutor interface {
	handle(context.Context, any) (any, error)
}

type notificationExecutor interface {
	handle(context.Context, any) error
}

type notificationPublisher interface {
	publish(context.Context, []notificationExecutor, any) error
}

// New creates a mediator with default registries and publisher.
func New(options ...Option) *Mediator {
	m := &Mediator{
		requestHandlers:      make(map[typekey.Pair]requestExecutor),
		notificationHandlers: make(map[reflect.Type][]notificationExecutor),
	}

	for _, option := range options {
		if option != nil {
			option(m)
		}
	}

	return m
}
