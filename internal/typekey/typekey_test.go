package typekey_test

import (
	"testing"

	"github.com/cocosip/mediator/internal/typekey"
)

type createUserRequest struct{}
type createUserResponse struct{}
type deleteUserResponse struct{}

func TestOfReturnsStableTypesForRepeatedLookups(t *testing.T) {
	first := typekey.Of[createUserRequest]()
	second := typekey.Of[createUserRequest]()

	if first != second {
		t.Fatalf("expected repeated lookups to return the same type, got %v and %v", first, second)
	}
}

func TestPairOfDistinguishesRequestAndResponseTypes(t *testing.T) {
	base := typekey.PairOf[createUserRequest, createUserResponse]()
	same := typekey.PairOf[createUserRequest, createUserResponse]()
	differentRequest := typekey.PairOf[*createUserRequest, createUserResponse]()
	differentResponse := typekey.PairOf[createUserRequest, deleteUserResponse]()

	if base != same {
		t.Fatalf("expected identical request/response pairs to match, got %v and %v", base, same)
	}

	if base == differentRequest {
		t.Fatalf("expected different request types to produce different keys, got %v and %v", base, differentRequest)
	}

	if base == differentResponse {
		t.Fatalf("expected different response types to produce different keys, got %v and %v", base, differentResponse)
	}
}
