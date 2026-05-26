package mediator_test

import (
	"context"
	"fmt"
	"log"

	"github.com/cocosip/mediator"
)

type pingRequest struct {
	Message string
}

type userRegisteredNotification struct {
	ID string
}

func ExampleSend() {
	m := mediator.New()

	err := mediator.RegisterRequestHandler(
		m,
		mediator.RequestHandlerFunc[pingRequest, string](
			func(ctx context.Context, request pingRequest) (string, error) {
				return "pong:" + request.Message, nil
			},
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	response, err := mediator.Send[pingRequest, string](
		context.Background(),
		m,
		pingRequest{Message: "hello"},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response)
	// Output:
	// pong:hello
}

func ExamplePublish() {
	m := mediator.New()

	err := mediator.RegisterNotificationHandler(
		m,
		mediator.NotificationHandlerFunc[userRegisteredNotification](
			func(ctx context.Context, notification userRegisteredNotification) error {
				fmt.Println("welcome", notification.ID)
				return nil
			},
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := mediator.Publish(
		context.Background(),
		m,
		userRegisteredNotification{ID: "user-1"},
	); err != nil {
		log.Fatal(err)
	}

	// Output:
	// welcome user-1
}

func ExampleRegisterPipelineBehavior() {
	m := mediator.New()

	err := mediator.RegisterRequestHandler(
		m,
		mediator.RequestHandlerFunc[pingRequest, string](
			func(ctx context.Context, request pingRequest) (string, error) {
				return request.Message, nil
			},
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = mediator.RegisterPipelineBehavior(
		m,
		mediator.PipelineBehaviorFunc[pingRequest, string](
			func(ctx context.Context, request pingRequest, next mediator.RequestHandlerDelegate[string]) (string, error) {
				fmt.Println("before")
				response, err := next(ctx)
				if err != nil {
					return "", err
				}
				fmt.Println("after")
				return "wrapped:" + response, nil
			},
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	response, err := mediator.Send[pingRequest, string](
		context.Background(),
		m,
		pingRequest{Message: "hello"},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response)
	// Output:
	// before
	// after
	// wrapped:hello
}
