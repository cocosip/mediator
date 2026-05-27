package mediator

import (
	"context"

	"github.com/cocosip/mediator/internal/typekey"
)

// StreamYield receives one streamed item from a stream handler.
type StreamYield[TItem any] func(context.Context, TItem) error

// StreamHandler handles a request by yielding zero or more response items.
type StreamHandler[TRequest any, TItem any] interface {
	Handle(context.Context, TRequest, StreamYield[TItem]) error
}

// StreamHandlerFunc adapts a function into a StreamHandler.
type StreamHandlerFunc[TRequest any, TItem any] func(context.Context, TRequest, StreamYield[TItem]) error

// Handle implements StreamHandler.
func (f StreamHandlerFunc[TRequest, TItem]) Handle(
	ctx context.Context,
	request TRequest,
	yield StreamYield[TItem],
) error {
	return f(ctx, request, yield)
}

type streamExecutor interface {
	handle(context.Context, any, func(context.Context, any) error) error
}

type streamHandlerAdapter[TRequest any, TItem any] struct {
	handler StreamHandler[TRequest, TItem]
}

func (a streamHandlerAdapter[TRequest, TItem]) handle(
	ctx context.Context,
	request any,
	yield func(context.Context, any) error,
) error {
	return a.handler.Handle(
		ctx,
		request.(TRequest),
		func(yieldCtx context.Context, item TItem) error {
			return yield(yieldCtx, item)
		},
	)
}

// RegisterStreamHandler registers the stream handler for a request/item pair.
func RegisterStreamHandler[TRequest any, TItem any](m *Mediator, handler StreamHandler[TRequest, TItem]) error {
	if handler == nil {
		return InvalidHandlerError{
			Kind:         "stream",
			MessageType:  typekey.Of[TRequest](),
			ResponseType: typekey.Of[TItem](),
		}
	}

	key := typekey.PairOf[TRequest, TItem]()

	m.requestMu.Lock()
	defer m.requestMu.Unlock()

	if _, exists := m.streamHandlers[key]; exists {
		return DuplicateHandlerError{
			RequestType:  key.Request,
			ResponseType: key.Response,
		}
	}

	m.streamHandlers[key] = streamHandlerAdapter[TRequest, TItem]{handler: handler}
	return nil
}

// Stream dispatches a stream request to its registered handler.
func Stream[TRequest any, TItem any](
	ctx context.Context,
	m *Mediator,
	request TRequest,
	yield StreamYield[TItem],
) error {
	key := typekey.PairOf[TRequest, TItem]()

	m.requestMu.RLock()
	handler, ok := m.streamHandlers[key]
	m.requestMu.RUnlock()
	if !ok {
		return HandlerNotFoundError{
			RequestType:  key.Request,
			ResponseType: key.Response,
		}
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	return handler.handle(
		ctx,
		request,
		func(yieldCtx context.Context, item any) error {
			if err := yieldCtx.Err(); err != nil {
				return err
			}

			if yield == nil {
				return nil
			}

			if err := yield(yieldCtx, item.(TItem)); err != nil {
				return err
			}

			return yieldCtx.Err()
		},
	)
}
