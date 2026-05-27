// Package mediator provides a small, dependency-free in-process mediator.
//
// The package centers on explicit registration and type-safe generic helpers:
//
//   - RegisterRequestHandler and Send for request/response flows
//   - RegisterNotificationHandler and Publish for notifications
//   - RegisterPipelineBehavior for request pipeline composition
//   - RegisterStreamHandler and Stream for callback-based streaming
//
// Registrations are safe against concurrent map access, but applications should
// generally complete registration during startup before calling Send, Publish,
// or Stream from serving goroutines.
package mediator
