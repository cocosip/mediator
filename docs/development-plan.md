# Go Mediator Development Plan

## Document Purpose

This document is the project development plan for the Go mediator framework described in `docs/design.md`.

It is not an implementation script and does not contain full source-code examples. It defines delivery phases, scope, milestones, acceptance criteria, and verification expectations so implementation can proceed in controlled increments.

## Project Goal

Build a Go-style in-process message communication framework inspired by MediatR, with a small dependency-free core and optional extension packages.

The first usable release should support:

- request/response dispatch
- notification publishing
- explicit generic registration
- request pipeline behaviors
- configurable notification publishing strategies
- idiomatic `context.Context` and `error` handling

Advanced capabilities such as stream requests, recover helpers, pre/post processor helpers, and DI integrations are planned after the core API is stable.

## Development Principles

- Keep the core package standard-library only.
- Implement in small phases with working tests at the end of each phase.
- Favor explicit APIs over automatic scanning.
- Keep MediatR concepts where useful, but choose Go-native semantics.
- Treat `docs/design.md` as the source of truth for API shape and behavior.
- Do not add DI or integration packages until the core API has been exercised.
- Avoid speculative features in the first working release.

## Target Repository Shape

Initial core files:

- `go.mod`
- `mediator.go`
- `request.go`
- `notification.go`
- `behavior.go`
- `errors.go`
- `internal/typekey/typekey.go`

Initial test files:

- `request_test.go`
- `notification_test.go`
- `behavior_test.go`
- `publisher_test.go`
- `errors_test.go`
- `internal/typekey/typekey_test.go`

Project docs:

- `docs/design.md`
- `docs/development-plan.md`

Optional future packages:

- `registry`
- `dig`
- `fx`
- `wire`

## Milestone Overview

| Milestone | Name | Outcome |
| --- | --- | --- |
| M1 | Project scaffold and type keys | Module exists with internal type-key helpers and basic error model. |
| M2 | Core dispatch | `Send` and `Publish` work with explicit registrations. |
| M3 | Request pipeline | Request behaviors can wrap, short-circuit, and transform handler execution. |
| M4 | Publisher strategies | Notification execution supports sequential/parallel strategies and error aggregation. |
| M5 | Core stabilization | Public API, docs, examples, race tests, and compatibility checks are complete. |
| M6 | Advanced extensions | Stream, recover, processors, registry, and DI integrations are evaluated and implemented selectively. |

## Phase 1: Project Scaffold

### Scope

Create the minimal Go module and internal foundations required by the mediator core.

### Checklist

- [x] Confirm the final module path.
- [x] Confirm the minimum supported Go version.
- [x] Create the Go module.
- [x] Create the internal type-key helper.
- [x] Add tests for stable generic type keys.
- [x] Add public sentinel errors.
- [x] Add structured errors for missing and duplicate handlers.
- [x] Add tests for `errors.Is` compatibility.
- [x] Run formatting.
- [x] Run the full test suite.
- [x] Update this checklist with completed items.

### Deliverables

- Go module initialized.
- Internal type-key helper based on `reflect.Type`.
- Public sentinel errors defined.
- Structured errors for common failure cases.
- Initial test suite layout in place.

### Decisions Required

- Final module path.
- Minimum supported Go version. Proposed minimum: Go 1.22, because `reflect.TypeFor` simplifies generic type-key implementation.

### Acceptance Criteria

- Type keys are stable for repeated generic type lookups.
- Request/response pair keys distinguish request and response types.
- Sentinel errors support `errors.Is`.
- The package builds with standard Go tooling.

### Verification

- `go test ./...`
- `gofmt` on all Go files.

## Phase 2: Core Request and Notification Dispatch

### Scope

Implement the first usable mediator core.

### Checklist

- [x] Define `Mediator`, constructor, options shell, and internal registries.
- [x] Add concurrency protection for registry reads and writes.
- [x] Define request handler interface and function adapter.
- [x] Implement request handler registration.
- [x] Implement type-safe request dispatch with `Send`.
- [x] Add tests for successful request dispatch.
- [x] Add tests for missing request handlers.
- [x] Add tests for duplicate request registrations.
- [x] Define notification handler interface and function adapter.
- [x] Implement notification handler registration.
- [x] Implement type-safe notification publishing with `Publish`.
- [x] Implement default sequential notification publisher.
- [x] Add tests for ordered notification handler execution.
- [x] Add tests for notifications with no handlers.
- [x] Add tests for stop-on-first-error notification behavior.
- [x] Run formatting.
- [x] Run the full test suite.
- [x] Run race tests if supported locally.
- [x] Update this checklist with completed items.

### Deliverables

- `Mediator` constructor and options shell.
- Request handler interface and function adapter.
- Request handler registration.
- Type-safe `Send[TRequest,TResponse]`.
- Notification handler interface and function adapter.
- Notification handler registration.
- Type-safe `Publish[TNotification]`.
- Default sequential notification publisher.
- Concurrent-safe access to registration maps.

### Request Dispatch Rules

- One request/response type pair has one handler.
- Duplicate request handler registration fails.
- Missing request handler returns a structured error wrapping `ErrHandlerNotFound`.
- Handler errors return unchanged unless a later pipeline behavior wraps them.

### Notification Dispatch Rules

- A notification type can have zero or more handlers.
- Publishing a notification with no handlers succeeds.
- Default publishing runs handlers sequentially in registration order.
- Default publishing stops on the first handler error.

### Acceptance Criteria

- Registered request handlers receive the correct request value and return the expected response.
- Missing request handlers are detectable with `errors.Is`.
- Duplicate request registrations are rejected.
- Multiple notification handlers run in registration order.
- Notification publish with no handlers returns nil.
- Notification publish stops on first error by default.
- Concurrent `Send` and `Publish` calls do not race against read-only registrations.

### Verification

- `go test ./...`
- `go test -race ./...` when supported locally.

## Phase 3: Request Pipeline Behaviors

### Scope

Add MediatR-style request pipeline behavior using Go generics and `context.Context`.

### Checklist

- [ ] Define request handler delegate type.
- [ ] Define pipeline behavior interface.
- [ ] Define pipeline behavior function adapter.
- [ ] Implement pipeline behavior registration.
- [ ] Compose registered behaviors during `Send`.
- [ ] Preserve the existing request handler dispatch path when no behaviors are registered.
- [ ] Add tests for behavior ordering.
- [ ] Add tests for before/after behavior execution.
- [ ] Add tests for behavior short-circuiting.
- [ ] Add tests for behavior error wrapping.
- [ ] Run existing Phase 2 tests to check for regressions.
- [ ] Run formatting.
- [ ] Run the full test suite.
- [ ] Run race tests if concurrency-sensitive code changed.
- [ ] Update this checklist with completed items.

### Deliverables

- Pipeline behavior interface.
- Pipeline behavior function adapter.
- Request handler delegate type.
- Pipeline behavior registration.
- Behavior composition during `Send`.

### Behavior Rules

- Behaviors are registered per request/response type pair.
- Behaviors execute in registration order.
- The first registered behavior is the outermost behavior.
- Behaviors can run code before and after the next delegate.
- Behaviors can short-circuit by not calling the next delegate.
- Behaviors can wrap or replace errors and responses.

### Acceptance Criteria

- Multiple behaviors execute in the documented order.
- Handler execution is surrounded by registered behaviors.
- A behavior can prevent the handler from running.
- A behavior can wrap a handler error.
- Existing Phase 2 request and notification behavior remains unchanged.

### Verification

- `go test ./...`
- `go test -race ./...` when behavior registration or dispatch concurrency changes.

## Phase 4: Notification Publisher Strategies

### Scope

Make notification execution strategy configurable.

### Checklist

- [ ] Define public notification publisher interface.
- [ ] Define public notification executor abstraction.
- [ ] Add option for replacing the notification publisher.
- [ ] Add notification error strategy enum.
- [ ] Update sequential publisher to support stop-on-first-error.
- [ ] Update sequential publisher to support continue-on-error.
- [ ] Add tests for custom publisher configuration.
- [ ] Add tests for sequential continue-on-error aggregation.
- [ ] Implement parallel publisher.
- [ ] Add tests for parallel handler execution.
- [ ] Add tests for parallel error aggregation.
- [ ] Add tests for context cancellation behavior.
- [ ] Review goroutine lifecycle for leaks.
- [ ] Run formatting.
- [ ] Run the full test suite.
- [ ] Run race tests.
- [ ] Update this checklist with completed items.

### Deliverables

- Public notification publisher interface.
- Public notification executor abstraction.
- Option for replacing the publisher.
- Configurable sequential publisher.
- Parallel publisher.
- Error strategy enum.

### Publisher Rules

- `StopOnFirstError` returns the first handler error.
- `ContinueOnError` attempts all handlers and aggregates failures with `errors.Join`.
- Parallel publishing waits for handler completion.
- Context cancellation is respected before starting work and while collecting results.
- Publisher implementations must not leak goroutines.

### Acceptance Criteria

- A custom publisher can be supplied through options.
- Sequential publisher supports stop and continue strategies.
- Parallel publisher can run all handlers and aggregate errors.
- Context cancellation returns a context error.
- Default behavior remains sequential stop-on-first-error.

### Verification

- `go test ./...`
- `go test -race ./...`

## Phase 5: Core Stabilization

### Scope

Prepare the core package for first practical use.

### Checklist

- [ ] Review public API names against Go conventions.
- [ ] Review exported identifiers for required documentation comments.
- [ ] Add package-level documentation.
- [ ] Add minimal request/response usage example.
- [ ] Add minimal notification usage example.
- [ ] Add minimal pipeline behavior usage example.
- [ ] Review error messages for clarity and stability.
- [ ] Review pointer/value message type behavior and document it.
- [ ] Review concurrency behavior and document registration-time expectations.
- [ ] Check `docs/design.md` against implemented behavior.
- [ ] Update `docs/design.md` for any approved behavior changes.
- [ ] Add or update README/getting-started content if needed.
- [ ] Run formatting.
- [ ] Run the full test suite.
- [ ] Run race tests.
- [ ] Mark first usable release scope complete.

### Deliverables

- Package-level documentation.
- Minimal usage examples.
- API review against `docs/design.md`.
- Error message review.
- Concurrency review.
- Public API naming review.
- README or getting-started section if the repository needs one.

### Acceptance Criteria

- Public API names are consistent and Go-like.
- Documentation shows the intended request, notification, and behavior usage.
- All exported identifiers have useful comments if linting requires them.
- No optional DI or stream work is mixed into the core release.
- Tests cover the documented behavior from Phases 1-4.

### Verification

- `go test ./...`
- `go test -race ./...`
- Manual review of exported API surface.

## Phase 6: Advanced Capabilities

Advanced capabilities should be implemented only after the core has stabilized.

### 6.1 Recover Behavior

Purpose:

- Provide an opt-in behavior that converts panics into errors.

Checklist:

- [ ] Confirm recover behavior API shape.
- [ ] Document panic handling semantics.
- [ ] Implement recover behavior helper.
- [ ] Add tests proving panic is not swallowed by default.
- [ ] Add tests proving registered recover behavior converts panic to error.
- [ ] Run the full test suite.

Acceptance criteria:

- Panics are not swallowed unless the behavior is registered.
- Registered recover behavior returns an application-defined error.
- Normal handler errors still flow through unchanged unless explicitly wrapped.

### 6.2 Pre/Post Processor Helpers

Purpose:

- Provide convenience helpers for users who want MediatR-like pre/post processing without adding a separate execution mechanism.

Checklist:

- [ ] Confirm helper API shape.
- [ ] Implement pre-processor helper on top of pipeline behavior.
- [ ] Implement post-processor helper on top of pipeline behavior.
- [ ] Add tests for pre-processor success path.
- [ ] Add tests for pre-processor error short-circuit.
- [ ] Add tests for post-processor success path.
- [ ] Add tests for post-processor error behavior.
- [ ] Run the full test suite.

Acceptance criteria:

- Helpers are implemented on top of pipeline behaviors.
- Pre-processors can stop execution by returning an error.
- Post-processors run after successful handler execution.
- Ordering remains compatible with pipeline behavior ordering.

### 6.3 Stream Requests

Purpose:

- Add streaming request support after choosing the final Go-native stream shape.

Checklist:

- [ ] Compare channel, callback, and iterator-style stream APIs.
- [ ] Select the stream API shape.
- [ ] Update `docs/design.md` with final stream semantics.
- [ ] Define stream handler registration behavior.
- [ ] Define stream dispatch behavior.
- [ ] Define stream cancellation behavior.
- [ ] Define stream error propagation behavior.
- [ ] Implement stream handler registration.
- [ ] Implement stream dispatch.
- [ ] Add tests for incremental item consumption.
- [ ] Add tests for stream errors.
- [ ] Add tests for context cancellation.
- [ ] Run the full test suite.
- [ ] Run race tests.

Open design decision:

- Channel API, callback API, or iterator-style API.

Before implementation, update `docs/design.md` with:

- cancellation behavior
- backpressure behavior
- error propagation
- cleanup responsibilities
- stream pipeline behavior rules, if supported

Acceptance criteria:

- Stream callers can consume items incrementally.
- Stream errors are observable.
- Context cancellation stops stream work.
- Handler cleanup is deterministic.

### 6.4 Registry Package

Purpose:

- Group explicit registrations without adding reflection scanning or DI dependencies.

Checklist:

- [ ] Confirm registry package is needed after core usage.
- [ ] Define registry API shape.
- [ ] Implement registration grouping.
- [ ] Add tests for successful grouped registration.
- [ ] Add tests for error propagation during grouped registration.
- [ ] Document registry package usage.
- [ ] Run the full test suite.

Acceptance criteria:

- Registry package depends only on the public core API.
- It simplifies startup registration without hiding errors.
- It remains optional.

### 6.5 DI Integration Packages

Purpose:

- Add integration packages only when a real DI target is selected.

Checklist:

- [ ] Select one DI integration target.
- [ ] Confirm the integration package belongs outside the core package.
- [ ] Define integration API shape.
- [ ] Implement integration using only public mediator APIs.
- [ ] Add integration tests or examples appropriate to the selected DI tool.
- [ ] Document dependency impact.
- [ ] Run the full test suite.

Candidate packages:

- `dig`
- `fx`
- `wire`

Acceptance criteria:

- Only one integration target is implemented at a time.
- Integration packages do not add dependencies to the core package.
- Integration code uses public APIs only.

## Release Boundaries

### First Usable Release

Includes:

- Phases 1-5.

Excludes:

- stream requests
- recover behavior
- pre/post helpers
- registry package
- DI integration packages

### Advanced Release

Includes selected Phase 6 capabilities after API review and usage feedback.

## Risks and Mitigations

### Go Generic API Ergonomics

Risk:

- Package-level generic functions may feel less object-oriented than MediatR's `mediator.Send`.

Mitigation:

- Keep function names short and consistent.
- Provide clear examples.
- Avoid dynamic `any` response APIs in the core path.

### Reflection Type-Key Edge Cases

Risk:

- Pointer and value message types are distinct keys.

Mitigation:

- Document that `CreateUser` and `*CreateUser` are different request types.
- Tests should cover value and pointer registrations.

### Pipeline Complexity

Risk:

- Behavior composition can become hard to reason about.

Mitigation:

- Keep one ordering rule: first registered behavior is outermost.
- Test ordering explicitly.
- Do not add separate pre/post execution machinery to the core path.

### Parallel Publisher Semantics

Risk:

- Stop-on-first-error is harder to define in concurrent execution.

Mitigation:

- Document exact behavior before implementation.
- Prefer deterministic aggregation for parallel continue-on-error.
- Ensure tests cover cancellation and error aggregation.

### Premature Integration Packages

Risk:

- DI adapters can distort the core API too early.

Mitigation:

- Ship core first.
- Add integrations only after a real target is chosen.

## Open Questions

- What is the final module path?
- Is Go 1.22 an acceptable minimum version?
- Should request handler keys include both request and response type, or should a request type map to exactly one response type? The current design uses both types.
- Should parallel stop-on-first-error cancel sibling handlers with a derived context, or only return the first observed error after started handlers complete?
- Which stream API shape best fits the intended users?
- Which DI integration, if any, is actually needed first?

## Definition of Done

A phase is complete when:

- its deliverables are implemented
- its acceptance criteria are covered by tests
- `go test ./...` passes
- race tests pass where relevant
- public behavior is reflected in `docs/design.md`
- no unrelated feature work is included

The first usable release is complete when Phases 1-5 meet this definition.
