package mediator

import (
	"context"

	"github.com/cocosip/mediator/internal/typekey"
)

// RequestHandlerDelegate represents the next step in a request pipeline.
type RequestHandlerDelegate[TResponse any] func(context.Context) (TResponse, error)

// PipelineBehavior wraps request handling with additional behavior.
type PipelineBehavior[TRequest any, TResponse any] interface {
	Handle(context.Context, TRequest, RequestHandlerDelegate[TResponse]) (TResponse, error)
}

// PipelineBehaviorFunc adapts a function into a PipelineBehavior.
type PipelineBehaviorFunc[TRequest any, TResponse any] func(context.Context, TRequest, RequestHandlerDelegate[TResponse]) (TResponse, error)

// Handle implements PipelineBehavior.
func (f PipelineBehaviorFunc[TRequest, TResponse]) Handle(
	ctx context.Context,
	request TRequest,
	next RequestHandlerDelegate[TResponse],
) (TResponse, error) {
	return f(ctx, request, next)
}

type pipelineBehaviorExecutor interface {
	handle(context.Context, any, func(context.Context) (any, error)) (any, error)
}

type pipelineBehaviorAdapter[TRequest any, TResponse any] struct {
	behavior PipelineBehavior[TRequest, TResponse]
}

func (a pipelineBehaviorAdapter[TRequest, TResponse]) handle(
	ctx context.Context,
	request any,
	next func(context.Context) (any, error),
) (any, error) {
	return a.behavior.Handle(
		ctx,
		request.(TRequest),
		func(nextCtx context.Context) (TResponse, error) {
			response, err := next(nextCtx)
			if err != nil {
				var zero TResponse
				return zero, err
			}

			return response.(TResponse), nil
		},
	)
}

// RegisterPipelineBehavior registers a behavior for a request/response pair.
func RegisterPipelineBehavior[TRequest any, TResponse any](m *Mediator, behavior PipelineBehavior[TRequest, TResponse]) error {
	if behavior == nil {
		return InvalidHandlerError{
			Kind:         "pipeline",
			MessageType:  typekey.Of[TRequest](),
			ResponseType: typekey.Of[TResponse](),
		}
	}

	key := typekey.PairOf[TRequest, TResponse]()

	m.requestMu.Lock()
	defer m.requestMu.Unlock()

	m.requestBehaviors[key] = append(
		m.requestBehaviors[key],
		pipelineBehaviorAdapter[TRequest, TResponse]{behavior: behavior},
	)

	return nil
}
