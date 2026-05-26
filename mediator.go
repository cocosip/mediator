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
	requestMu        sync.RWMutex
	requestHandlers  map[typekey.Pair]requestExecutor
	requestBehaviors map[typekey.Pair][]pipelineBehaviorExecutor

	notificationMu        sync.RWMutex
	notificationHandlers  map[reflect.Type][]NotificationExecutor
	notificationPublisher NotificationPublisher
}

type requestExecutor interface {
	handle(context.Context, any) (any, error)
}

// New creates a mediator with default registries and publisher.
func New(options ...Option) *Mediator {
	m := &Mediator{
		requestHandlers:      make(map[typekey.Pair]requestExecutor),
		requestBehaviors:     make(map[typekey.Pair][]pipelineBehaviorExecutor),
		notificationHandlers: make(map[reflect.Type][]NotificationExecutor),
		notificationPublisher: SequentialPublisher{
			ErrorStrategy: StopOnFirstError,
		},
	}

	for _, option := range options {
		if option != nil {
			option(m)
		}
	}

	return m
}
