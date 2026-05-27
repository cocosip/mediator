package registry_test

import (
	"context"
	"fmt"
	"log"

	"github.com/cocosip/mediator"
	"github.com/cocosip/mediator/registry"
)

type examplePing struct {
	Message string
}

func ExampleRegistry_Apply() {
	m := mediator.New()
	r := registry.New()

	registry.AddRequestHandler(
		r,
		mediator.RequestHandlerFunc[examplePing, string](
			func(ctx context.Context, request examplePing) (string, error) {
				return "pong:" + request.Message, nil
			},
		),
	)

	if err := r.Apply(m); err != nil {
		log.Fatal(err)
	}

	response, err := mediator.Send[examplePing, string](
		context.Background(),
		m,
		examplePing{Message: "hello"},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response)
	// Output:
	// pong:hello
}
