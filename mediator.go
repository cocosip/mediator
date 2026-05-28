package mediator

import (
	"context"
	"fmt"
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
	streamHandlers   map[typekey.Pair]streamExecutor

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
		streamHandlers:       make(map[typekey.Pair]streamExecutor),
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

func isNilValue(value any) bool {
	if value == nil {
		return true
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

func castResponse[T any](response any) (T, error) {
	var zero T
	if response == nil {
		return zero, nil
	}

	typed, ok := response.(T)
	if !ok {
		return zero, fmt.Errorf("mediator: invalid response type %T", response)
	}

	return typed, nil
}
