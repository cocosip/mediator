# Go Mediator Design

## Purpose

This project implements a Go-style in-process message communication framework inspired by the C# MediatR library. The goal is not to port MediatR line by line. The goal is to keep the useful mediator concepts while making the API idiomatic for Go: explicit registration, `context.Context`, generic helper functions, normal `error` handling, and a small dependency-free core.

MediatR's public model includes request/response messages, commands, queries, notifications/events, stream requests, pipeline behaviors, pre/post processors, exception handling hooks, and notification publisher strategies. This project keeps those concepts where they fit Go, and adjusts the mechanics where C# patterns would feel unnatural.

## Design Principles

- Keep the core package dependency-free.
- Prefer explicit registration over reflection-based assembly scanning.
- Use generic package functions for type-safe APIs.
- Do not require message structs to implement framework marker interfaces.
- Use `context.Context` for cancellation, deadlines, and request-scoped values.
- Use Go `error` values instead of exception-specific abstractions.
- Keep optional DI, scanning, and integration behavior outside the core package.
- Make the first implementation small, testable, and stable before adding advanced features.

## Package Direction

Initial package layout:

```text
mediator/
  go.mod
  mediator.go          // Mediator, Options, Send, Publish
  request.go           // request handlers and request execution
  notification.go      // notification handlers and publisher strategies
  behavior.go          // request pipeline behavior types and composition
  errors.go            // public sentinel and structured errors
  internal/typekey/    // reflect.Type based keys for generic registrations
```

Optional integration packages can be added later:

```text
mediator/registry       // batch registration helpers
mediator/dig            // dig integration, if needed
mediator/fx             // fx integration, if needed
mediator/wire           // wire provider helpers, if needed
```

The core package must not depend on these integration packages.

## Core API

Go does not support generic methods, so the primary type-safe operations are
package-level generic functions that receive a `*Mediator`.

To reduce direct coupling to the full mediator implementation, the package also
provides typed facade interfaces and adapter functions. This keeps the
package-level generic entry points for strong typing while still letting
application code depend on narrower capabilities.

### Mediator

```go
type Mediator struct {
    // internal request handlers, notification handlers, behaviors, publisher
}

func New(options ...Option) *Mediator
```

`Mediator` owns handler registrations and execution options. It is safe for concurrent calls to `Send` and `Publish`, and registration APIs avoid map read/write panics. Applications are still encouraged to finish registration during startup before serving traffic so runtime behavior stays predictable.

Typed facade adapters project a `*Mediator` into narrower capabilities:

```go
type Sender[TRequest any, TResponse any] interface {
    Send(context.Context, TRequest) (TResponse, error)
}

type Publisher[TNotification any] interface {
    Publish(context.Context, TNotification) error
}

type RequestRegistrar[TRequest any, TResponse any] interface {
    Register(RequestHandler[TRequest, TResponse]) error
}

type NotificationRegistrar[TNotification any] interface {
    Register(NotificationHandler[TNotification]) error
}

type BehaviorRegistrar[TRequest any, TResponse any] interface {
    Register(PipelineBehavior[TRequest, TResponse]) error
}
```

Adapter functions expose these typed capabilities:

```go
func RequestSender[TRequest any, TResponse any](m *Mediator) Sender[TRequest, TResponse]
func NotificationPublisherOf[TNotification any](m *Mediator) Publisher[TNotification]
func RequestRegistration[TRequest any, TResponse any](m *Mediator) RequestRegistrar[TRequest, TResponse]
func NotificationRegistration[TNotification any](m *Mediator) NotificationRegistrar[TNotification]
func BehaviorRegistration[TRequest any, TResponse any](m *Mediator) BehaviorRegistrar[TRequest, TResponse]
```

### Request/Response

```go
type RequestHandler[TRequest any, TResponse any] interface {
    Handle(context.Context, TRequest) (TResponse, error)
}

type RequestHandlerFunc[TRequest any, TResponse any] func(
    context.Context,
    TRequest,
) (TResponse, error)

func (f RequestHandlerFunc[TRequest, TResponse]) Handle(
    ctx context.Context,
    request TRequest,
) (TResponse, error)

func RegisterRequestHandler[TRequest any, TResponse any](
    m *Mediator,
    handler RequestHandler[TRequest, TResponse],
) error

func Send[TRequest any, TResponse any](
    ctx context.Context,
    m *Mediator,
    request TRequest,
) (TResponse, error)
```

Rules:

- One request type and response type pair has exactly one handler.
- Duplicate request handler registration returns `ErrDuplicateHandler`.
- Missing request handler returns an error that satisfies `errors.Is(err, ErrHandlerNotFound)`.
- Request messages do not need to implement `IRequest`-style marker interfaces.
- Registrations match the exact generic type pair used at registration and dispatch time, so value and pointer request types are distinct.

### Notifications

```go
type NotificationHandler[TNotification any] interface {
    Handle(context.Context, TNotification) error
}

type NotificationHandlerFunc[TNotification any] func(
    context.Context,
    TNotification,
) error

func (f NotificationHandlerFunc[TNotification]) Handle(
    ctx context.Context,
    notification TNotification,
) error

func RegisterNotificationHandler[TNotification any](
    m *Mediator,
    handler NotificationHandler[TNotification],
) error

func Publish[TNotification any](
    ctx context.Context,
    m *Mediator,
    notification TNotification,
) error
```

Rules:

- A notification type can have zero or more handlers.
- If no notification handler is registered, `Publish` returns nil.
- By default, notification handlers run sequentially in registration order.
- The default error behavior stops on the first handler error.
- Registrations match the exact notification type used at registration and publish time, so value and pointer notification types are distinct.

### Pipeline Behaviors

```go
type RequestHandlerDelegate[TResponse any] func(context.Context) (TResponse, error)

type PipelineBehavior[TRequest any, TResponse any] interface {
    Handle(
        context.Context,
        TRequest,
        RequestHandlerDelegate[TResponse],
    ) (TResponse, error)
}

type PipelineBehaviorFunc[TRequest any, TResponse any] func(
    context.Context,
    TRequest,
    RequestHandlerDelegate[TResponse],
) (TResponse, error)

func (f PipelineBehaviorFunc[TRequest, TResponse]) Handle(
    ctx context.Context,
    request TRequest,
    next RequestHandlerDelegate[TResponse],
) (TResponse, error)

func RegisterPipelineBehavior[TRequest any, TResponse any](
    m *Mediator,
    behavior PipelineBehavior[TRequest, TResponse],
) error
```

Pipeline rules:

- Behaviors apply to matching `TRequest` and `TResponse`.
- Behaviors execute in registration order.
- The first registered behavior is the outermost behavior.
- A behavior may call `next(ctx)` exactly once, multiple times, or not at all.
- A behavior may wrap errors, replace responses, short-circuit execution, or implement cross-cutting concerns such as logging, validation, tracing, retries, and transactions.

Example execution order:

```text
RegisterPipelineBehavior(m, logging)
RegisterPipelineBehavior(m, validation)

Send(...)

logging before
  validation before
    handler
  validation after
logging after
```

## Notification Publishers

The notification execution strategy should be replaceable.

```go
type NotificationExecutor interface {
    Handle(context.Context, any) error
}

type NotificationPublisher interface {
    Publish(
        ctx context.Context,
        handlers []NotificationExecutor,
        notification any,
    ) error
}

func WithNotificationPublisher(publisher NotificationPublisher) Option
```

Built-in publishers:

- `SequentialPublisher`: runs handlers in registration order.
- `ParallelPublisher`: runs handlers concurrently and waits for completion according to its error strategy.

Error strategy:

```go
type NotificationErrorStrategy int

const (
    StopOnFirstError NotificationErrorStrategy = iota
    ContinueOnError
)
```

Rules:

- `StopOnFirstError` returns the first error.
- `ContinueOnError` attempts all handlers and returns `errors.Join(...)` when one or more handlers fail.
- `ParallelPublisher` should respect context cancellation and avoid goroutine leaks.
- Publisher implementations should return context errors when cancellation prevents meaningful completion.
- The default mediator publisher is `SequentialPublisher{ErrorStrategy: StopOnFirstError}`.

## Error Model

Public sentinel errors:

```go
var (
    ErrHandlerNotFound  = errors.New("mediator: handler not found")
    ErrDuplicateHandler = errors.New("mediator: duplicate handler")
    ErrInvalidHandler   = errors.New("mediator: invalid handler")
)
```

Structured errors wrap sentinels so callers can use `errors.Is` while still receiving useful details.

```go
type HandlerNotFoundError struct {
    RequestType  reflect.Type
    ResponseType reflect.Type
}

func (e HandlerNotFoundError) Error() string
func (e HandlerNotFoundError) Unwrap() error { return ErrHandlerNotFound }
```

`DuplicateHandlerError` and `InvalidHandlerError` follow the same pattern and also unwrap to their sentinel errors.

Go error handling replaces MediatR's exception-specific abstractions:

- Error wrapping, retry, fallback, and logging should usually be implemented as pipeline behaviors.
- Panic recovery should not happen by default.
- A future `RecoverBehavior` helper can convert panics to errors when applications explicitly register it.

Possible future helper:

```go
func RecoverBehavior[TRequest any, TResponse any](
    onPanic func(ctx context.Context, request TRequest, recovered any) error,
) PipelineBehavior[TRequest, TResponse]
```

## Stream Requests

Stream support should be designed now but implemented after the core request and notification paths are stable.

Possible API:

```go
type StreamHandler[TRequest any, TItem any] interface {
    Handle(context.Context, TRequest) (<-chan TItem, <-chan error)
}

func RegisterStreamHandler[TRequest any, TItem any](
    m *Mediator,
    handler StreamHandler[TRequest, TItem],
) error

func Stream[TRequest any, TItem any](
    ctx context.Context,
    m *Mediator,
    request TRequest,
) (<-chan TItem, <-chan error)
```

The exact stream API should be validated during the stream phase. A callback or iterator-style API may be preferable if channel-based errors become awkward. The stream design must clearly define cancellation, handler cleanup, backpressure, and error propagation before implementation.

## Pre/Post Processors

MediatR exposes pre-processors and post-processors as distinct pipeline components. In Go, these should be helpers built on top of `PipelineBehavior`, not a separate primary mechanism.

Possible helpers:

```go
func PreProcessor[TRequest any, TResponse any](
    fn func(context.Context, TRequest) error,
) PipelineBehavior[TRequest, TResponse]

func PostProcessor[TRequest any, TResponse any](
    fn func(context.Context, TRequest, TResponse) error,
) PipelineBehavior[TRequest, TResponse]
```

This keeps the core mental model small: request execution has handlers and behaviors.

## Type Keys

Internally, registrations need stable keys for generic type parameters. The practical Go approach is to use `reflect.Type`.

Request handler key:

```text
(TRequest, TResponse)
```

Notification handler key:

```text
TNotification
```

Pipeline behavior key:

```text
(TRequest, TResponse)
```

Implementation should hide this detail behind small internal helpers so the public API stays generic and simple.

## Development Phases

### Phase 1: Project Scaffold

Goal: establish the module, type keys, and public error model.

Scope:

- `go.mod`
- `internal/typekey`
- public sentinel and structured errors
- basic unit tests for type keys and `errors.Is`

Acceptance tests:

- Type keys are stable for repeated generic lookups.
- Request/response type pairs produce distinct registration keys.
- Structured errors satisfy `errors.Is` for their sentinels.

### Phase 2: Core Message Dispatch

Goal: build a usable, dependency-free mediator core.

Scope:

- `New`
- `Send[TRequest,TResponse]`
- `RegisterRequestHandler`
- `RequestHandlerFunc`
- `Publish[TNotification]`
- `RegisterNotificationHandler`
- `NotificationHandlerFunc`
- default sequential notification publishing
- concurrent-safe registration and dispatch maps

Acceptance tests:

- Request handler returns a response.
- Missing request handler returns an error matching `ErrHandlerNotFound`.
- Duplicate request handler registration returns an error matching `ErrDuplicateHandler`.
- Multiple notification handlers run in registration order.
- Notification with no handlers returns nil.
- Sequential notification publishing stops on the first handler error.

### Phase 3: Request Pipeline

Goal: add request pipeline behaviors for cross-cutting concerns.

Scope:

- `PipelineBehavior`
- `PipelineBehaviorFunc`
- `RequestHandlerDelegate`
- `RegisterPipelineBehavior`
- behavior composition during `Send`

Acceptance tests:

- Multiple behaviors execute in the documented order.
- Behavior code can run before and after the handler.
- A behavior can short-circuit without calling the handler.
- A behavior can wrap a handler error.

### Phase 4: Publisher Strategies

Goal: make notification publishing configurable.

Scope:

- `NotificationPublisher`
- `NotificationExecutor`
- `WithNotificationPublisher`
- configurable `SequentialPublisher`
- `ParallelPublisher`
- `StopOnFirstError`
- `ContinueOnError`
- `errors.Join` aggregation
- context cancellation tests

Acceptance tests:

- Parallel publisher executes all handlers when configured to continue on error.
- Continue strategy aggregates multiple errors.
- Stop strategy returns deterministically.
- Context cancellation is returned and does not leak work.

### Phase 5: Core Stabilization

Goal: document and stabilize the first practical release of the core package.

Scope:

- package-level documentation
- minimal usage examples
- README / getting-started guidance
- API review against implemented behavior
- concurrency and pointer/value message documentation

Acceptance tests:

- Example tests cover request, notification, and pipeline behavior usage.
- Public API names remain Go-like and consistent.
- Documentation matches the implemented Phases 1-4 behavior.

### Phase 6: Advanced Features and Integrations

Goal: add advanced MediatR-inspired capabilities without bloating the core.

Scope:

- stream request API and implementation
- stream pipeline behavior, if the final stream model supports it cleanly
- `RecoverBehavior`
- pre/post processor helpers
- optional `registry` package for batch registration
- optional DI integration packages only when a real target is selected

Acceptance tests:

- Stream handlers produce items and respond to context cancellation.
- Stream errors are observable by callers.
- Recover behavior converts registered request panics into errors.
- Pre/post helpers preserve the same behavior ordering rules.
- Integration packages depend only on the public core API.

## References

- MediatR repository: https://github.com/LuckyPennySoftware/MediatR
- MediatR README: https://github.com/LuckyPennySoftware/MediatR/blob/master/README.md
- MediatR wiki: https://github.com/LuckyPennySoftware/MediatR/wiki
- MediatR behaviors wiki mirror: https://github-wiki-see.page/m/LuckyPennySoftware/MediatR/wiki/Behaviors
