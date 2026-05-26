package typekey

import "reflect"

// Pair identifies a request/response registration key.
type Pair struct {
	Request  reflect.Type
	Response reflect.Type
}

// Of returns the reflect.Type for T using a generic-safe lookup.
func Of[T any]() reflect.Type {
	return reflect.TypeFor[T]()
}

// PairOf returns the request/response type pair for generic registrations.
func PairOf[TRequest any, TResponse any]() Pair {
	return Pair{
		Request:  Of[TRequest](),
		Response: Of[TResponse](),
	}
}
