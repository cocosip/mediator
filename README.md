# mediator

`mediator` is a small, dependency-free in-process message mediator for Go.

It provides:

- type-safe request/response dispatch with `Send`
- notification publishing with configurable execution strategies
- request pipeline behaviors for cross-cutting concerns
- typed facade interfaces for narrower dependencies on mediator capabilities

## Install

```bash
go get github.com/cocosip/mediator
```

## Quick Start

### Request/response

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

### Notifications

```go
m := mediator.New(
	mediator.WithNotificationPublisher(mediator.SequentialPublisher{
		ErrorStrategy: mediator.ContinueOnError,
	}),
)
```

Register one or more notification handlers with `RegisterNotificationHandler`,
then call `Publish`.

### Pipeline behaviors

Register request behaviors with `RegisterPipelineBehavior` to add logging,
validation, retries, transactions, or error wrapping around handlers.

The original package-level APIs remain supported. The facade layer is an
additional abstraction for callers that prefer depending on narrow interfaces.

## Concurrency

Registration APIs use locks so concurrent reads and writes do not panic, but
the intended usage is still startup-time registration followed by runtime
dispatch.

## Message Types

Registrations match the exact generic type pair you use.

- `MyRequest` and `*MyRequest` are different request types
- `MyNotification` and `*MyNotification` are different notification types

Keep registration and dispatch consistent for pointer vs. value messages.
