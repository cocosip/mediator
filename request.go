package mediator

import (
	"context"

	"github.com/cocosip/mediator/internal/typekey"
)

// RequestHandler handles a request and returns a response.
type RequestHandler[TRequest any, TResponse any] interface {
	Handle(context.Context, TRequest) (TResponse, error)
}

// RequestHandlerFunc adapts a function into a RequestHandler.
type RequestHandlerFunc[TRequest any, TResponse any] func(context.Context, TRequest) (TResponse, error)

// Handle implements RequestHandler.
func (f RequestHandlerFunc[TRequest, TResponse]) Handle(ctx context.Context, request TRequest) (TResponse, error) {
	return f(ctx, request)
}

type requestHandlerAdapter[TRequest any, TResponse any] struct {
	handler RequestHandler[TRequest, TResponse]
}

func (a requestHandlerAdapter[TRequest, TResponse]) handle(ctx context.Context, request any) (any, error) {
	response, err := a.handler.Handle(ctx, request.(TRequest))
	if err != nil {
		return nil, err
	}

	return response, nil
}

// RegisterRequestHandler registers the handler for a request/response pair.
func RegisterRequestHandler[TRequest any, TResponse any](m *Mediator, handler RequestHandler[TRequest, TResponse]) error {
	if handler == nil {
		return InvalidHandlerError{
			Kind:         "request",
			MessageType:  typekey.Of[TRequest](),
			ResponseType: typekey.Of[TResponse](),
		}
	}

	key := typekey.PairOf[TRequest, TResponse]()

	m.requestMu.Lock()
	defer m.requestMu.Unlock()

	if _, exists := m.requestHandlers[key]; exists {
		return DuplicateHandlerError{
			RequestType:  key.Request,
			ResponseType: key.Response,
		}
	}

	m.requestHandlers[key] = requestHandlerAdapter[TRequest, TResponse]{handler: handler}
	return nil
}

// Send dispatches the request to the registered handler.
func Send[TRequest any, TResponse any](ctx context.Context, m *Mediator, request TRequest) (TResponse, error) {
	var zero TResponse

	key := typekey.PairOf[TRequest, TResponse]()

	m.requestMu.RLock()
	handler, ok := m.requestHandlers[key]
	registeredBehaviors := m.requestBehaviors[key]
	behaviors := append([]pipelineBehaviorExecutor(nil), registeredBehaviors...)
	m.requestMu.RUnlock()
	if !ok {
		return zero, HandlerNotFoundError{
			RequestType:  key.Request,
			ResponseType: key.Response,
		}
	}

	if len(behaviors) == 0 {
		response, err := handler.handle(ctx, request)
		if err != nil {
			return zero, err
		}

		return response.(TResponse), nil
	}

	next := func(nextCtx context.Context) (any, error) {
		return handler.handle(nextCtx, request)
	}

	for i := len(behaviors) - 1; i >= 0; i-- {
		behavior := behaviors[i]
		currentNext := next
		next = func(nextCtx context.Context) (any, error) {
			return behavior.handle(nextCtx, request, currentNext)
		}
	}

	response, err := next(ctx)
	if err != nil {
		return zero, err
	}

	return response.(TResponse), nil
}
