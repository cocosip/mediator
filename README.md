# mediator

`mediator` is a small, dependency-free in-process message mediator for Go.

It provides:

- type-safe request/response dispatch with `Send`
- notification publishing with configurable execution strategies
- request pipeline behaviors for cross-cutting concerns
- callback-based stream request dispatch
- typed facade interfaces for narrower dependencies on mediator capabilities

## Install

```bash
go get github.com/cocosip/mediator
```

## CI/CD

The repository uses GitHub Actions for validation and tagged releases.

- Pushes and pull requests to `master` run `go test ./...`, `go test -race ./...`,
  and `golangci-lint`.
- Pushing a tag that matches `v*` creates a GitHub Release.
- Release notes are generated from commit subjects between the previous tag and
  the current tag. If there is no previous tag, the notes include all commits
  reachable from the first tag.

Tag a release with:

```bash
git tag v0.1.0
git push origin v0.1.0
```

## Quick Start

### Request/response with `RequestHandlerFunc`

```go
package main

import (
	"context"
	"fmt"

	"github.com/cocosip/mediator"
)

type Ping struct {
	Message string
}

func main() {
	m := mediator.New()

	_ = mediator.RegisterRequestHandler(
		m,
		mediator.RequestHandlerFunc[Ping, string](
			func(ctx context.Context, request Ping) (string, error) {
				return "pong:" + request.Message, nil
			},
		),
	)

	response, _ := mediator.Send[Ping, string](context.Background(), m, Ping{Message: "hello"})
	fmt.Println(response)
}
```

`RequestHandlerFunc` is the most convenient option for small handlers and
simple glue code.

### Request/response with a handler type

For non-trivial business logic, implement `RequestHandler` with your own type
and keep dependencies on fields:

```go
type UserRepository interface {
	Create(context.Context, string) (string, error)
}

type CreateUser struct {
	Name string
}

type CreateUserResult struct {
	ID string
}

type CreateUserHandler struct {
	repo UserRepository
}

func (h *CreateUserHandler) Handle(
	ctx context.Context,
	request CreateUser,
) (CreateUserResult, error) {
	id, err := h.repo.Create(ctx, request.Name)
	if err != nil {
		return CreateUserResult{}, err
	}

	return CreateUserResult{ID: id}, nil
}

func registerHandler(m *mediator.Mediator, repo UserRepository) error {
	return mediator.RegisterRequestHandler(
		m,
		&CreateUserHandler{repo: repo},
	)
}
```

`RegisterRequestHandler` already accepts the `RequestHandler` interface, so
`RequestHandlerFunc` is only one implementation choice.

### Typed facades

If you want business code to depend on a narrower capability instead of the
full `*Mediator`, project it into a typed facade:

```go
sender := mediator.RequestSender[Ping, string](m)
response, _ := sender.Send(context.Background(), Ping{Message: "hello"})
```

Available facades include:

- `RequestSender[TRequest, TResponse]`
- `NotificationPublisherOf[TNotification]`
- `RequestRegistration[TRequest, TResponse]`
- `NotificationRegistration[TNotification]`
- `BehaviorRegistration[TRequest, TResponse]`

This is useful when a service should only depend on one mediator capability.
For example, a component that only sends `Ping -> string` requests can depend
on `Sender[Ping, string]` instead of depending on the full `*Mediator`.

```go
type PingService struct {
	sender mediator.Sender[Ping, string]
}

func (s PingService) Execute(ctx context.Context, message string) (string, error) {
	return s.sender.Send(ctx, Ping{Message: message})
}
```

### Notifications

#### Minimal publish/subscribe

```go
type UserCreated struct {
	ID string
}

func main() {
	m := mediator.New()

	_ = mediator.RegisterNotificationHandler(
		m,
		mediator.NotificationHandlerFunc[UserCreated](
			func(ctx context.Context, notification UserCreated) error {
				fmt.Println("welcome", notification.ID)
				return nil
			},
		),
	)

	_ = mediator.Publish(context.Background(), m, UserCreated{ID: "user-1"})
}
```

#### Multiple subscribers

Register the same notification type more than once to fan out work to multiple
subscribers:

```go
_ = mediator.RegisterNotificationHandler(
	m,
	mediator.NotificationHandlerFunc[UserCreated](
		func(ctx context.Context, notification UserCreated) error {
			fmt.Println("send welcome email", notification.ID)
			return nil
		},
	),
)

_ = mediator.RegisterNotificationHandler(
	m,
	mediator.NotificationHandlerFunc[UserCreated](
		func(ctx context.Context, notification UserCreated) error {
			fmt.Println("write audit log", notification.ID)
			return nil
		},
	),
)
```

By default, subscribers run sequentially in registration order and stop on the
first error.

#### Publisher strategies

```go
m := mediator.New(
	mediator.WithNotificationPublisher(mediator.SequentialPublisher{
		ErrorStrategy: mediator.ContinueOnError,
	}),
)
```

Use `SequentialPublisher` when you want deterministic in-order execution.

Use `ParallelPublisher` when subscribers can run independently:

```go
m := mediator.New(
	mediator.WithNotificationPublisher(mediator.ParallelPublisher{
		ErrorStrategy: mediator.ContinueOnError,
	}),
)
```

Register one or more notification handlers with `RegisterNotificationHandler`,
then call `Publish`.

If you want the same narrow-dependency style for notifications, use
`NotificationPublisherOf[T]` to obtain a `Publisher[T]`.

### Pipeline behaviors

Register request behaviors with `RegisterPipelineBehavior` to add logging,
validation, retries, transactions, or error wrapping around handlers.

#### Before / after behavior

```go
type CreateOrder struct {
	ID string
}

type CreateOrderHandler struct{}

func (h CreateOrderHandler) Handle(
	ctx context.Context,
	request CreateOrder,
) (string, error) {
	return "created:" + request.ID, nil
}

m := mediator.New()

_ = mediator.RegisterRequestHandler(
	m,
	CreateOrderHandler{},
)

_ = mediator.RegisterPipelineBehavior(
	m,
	mediator.PipelineBehaviorFunc[CreateOrder, string](
		func(
			ctx context.Context,
			request CreateOrder,
			next mediator.RequestHandlerDelegate[string],
		) (string, error) {
			fmt.Println("before", request.ID)

			response, err := next(ctx)
			if err != nil {
				return "", err
			}

			fmt.Println("after", request.ID)
			return response, nil
		},
	),
)

response, _ := mediator.Send[CreateOrder, string](
	context.Background(),
	m,
	CreateOrder{ID: "order-1"},
)

fmt.Println(response)
```

#### Validation short-circuit

Behaviors can stop execution before the handler runs:

```go
_ = mediator.RegisterPipelineBehavior(
	m,
	mediator.PipelineBehaviorFunc[CreateOrder, string](
		func(
			ctx context.Context,
			request CreateOrder,
			next mediator.RequestHandlerDelegate[string],
		) (string, error) {
			if request.ID == "" {
				return "", errors.New("order id is required")
			}

			return next(ctx)
		},
	),
)
```

The original package-level APIs remain supported. The facade layer is an
additional abstraction for callers that prefer depending on narrow interfaces.

### Advanced request helpers

`RecoverBehavior` is opt-in. Without it, handler panics propagate normally.

```go
_ = mediator.RegisterPipelineBehavior(
	m,
	mediator.RecoverBehavior[CreateOrder, string](
		func(ctx context.Context, request CreateOrder, recovered any) error {
			return fmt.Errorf("create order failed: %v", recovered)
		},
	),
)
```

Pre/post helpers are ordinary pipeline behaviors:

```go
_ = mediator.RegisterPipelineBehavior(
	m,
	mediator.PreProcessor[CreateOrder, string](
		func(ctx context.Context, request CreateOrder) error {
			if request.ID == "" {
				return errors.New("order id is required")
			}

			return nil
		},
	),
)
```

Use `PostProcessor` when work should run only after successful handler
execution.

```go
_ = mediator.RegisterPipelineBehavior(
	m,
	mediator.PostProcessor[CreateOrder, string](
		func(ctx context.Context, request CreateOrder, response string) error {
			fmt.Println("created", request.ID, response)
			return nil
		},
	),
)
```

### Stream requests

Stream handlers use a callback API. The handler yields one item at a time, and
`Stream` returns the first handler, yield, or context error.

```go
type ListOrders struct {
	UserID string
}

_ = mediator.RegisterStreamHandler(
	m,
	mediator.StreamHandlerFunc[ListOrders, string](
		func(ctx context.Context, request ListOrders, yield mediator.StreamYield[string]) error {
			for _, id := range []string{"order-1", "order-2"} {
				if err := yield(ctx, id); err != nil {
					return err
				}
			}

			return nil
		},
	),
)

err := mediator.Stream(
	context.Background(),
	m,
	ListOrders{UserID: "user-1"},
	func(ctx context.Context, orderID string) error {
		fmt.Println(orderID)
		return nil
	},
)
```

This shape gives synchronous backpressure: the handler does not continue until
the callback returns.

### Grouped registrations

The optional `registry` package groups explicit registrations without scanning
or DI dependencies:

```go
import "github.com/cocosip/mediator/registry"

r := registry.New()

registry.AddRequestHandler(
	r,
	mediator.RequestHandlerFunc[Ping, string](
		func(ctx context.Context, request Ping) (string, error) {
			return "pong:" + request.Message, nil
		},
	),
)

if err := r.Apply(m); err != nil {
	return err
}
```

## Concurrency

Registration APIs use locks so concurrent reads and writes do not panic, but
the intended usage is still startup-time registration followed by runtime
dispatch.

## Message Types

Registrations match the exact generic type pair you use.

- `MyRequest` and `*MyRequest` are different request types
- `MyNotification` and `*MyNotification` are different notification types

Keep registration and dispatch consistent for pointer vs. value messages.
