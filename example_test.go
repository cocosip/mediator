package mediator_test

import (
	"context"
	"errors"
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
			func(_ context.Context, request pingRequest) (string, error) {
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
		pingRequest{Message: testHello},
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
			func(_ context.Context, notification userRegisteredNotification) error {
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
		userRegisteredNotification{ID: testUserID},
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
			func(_ context.Context, request pingRequest) (string, error) {
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
			func(ctx context.Context, _ pingRequest, next mediator.RequestHandlerDelegate[string]) (string, error) {
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
		pingRequest{Message: testHello},
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

func ExampleRecoverBehavior() {
	m := mediator.New()

	err := mediator.RegisterRequestHandler(
		m,
		mediator.RequestHandlerFunc[pingRequest, string](
			func(_ context.Context, _ pingRequest) (string, error) {
				panic("service unavailable")
			},
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = mediator.RegisterPipelineBehavior(
		m,
		mediator.RecoverBehavior[pingRequest, string](
			func(_ context.Context, _ pingRequest, recovered any) error {
				return fmt.Errorf("ping failed: %v", recovered)
			},
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = mediator.Send[pingRequest, string](
		context.Background(),
		m,
		pingRequest{Message: testHello},
	)
	fmt.Println(err)
	// Output:
	// ping failed: service unavailable
}

func ExamplePreProcessor() {
	m := mediator.New()

	err := mediator.RegisterRequestHandler(
		m,
		mediator.RequestHandlerFunc[pingRequest, string](
			func(_ context.Context, request pingRequest) (string, error) {
				return "pong:" + request.Message, nil
			},
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = mediator.RegisterPipelineBehavior(
		m,
		mediator.PreProcessor[pingRequest, string](
			func(_ context.Context, request pingRequest) error {
				if request.Message == "" {
					return errors.New("message is required")
				}

				return nil
			},
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = mediator.Send[pingRequest, string](
		context.Background(),
		m,
		pingRequest{},
	)
	fmt.Println(err)
	// Output:
	// message is required
}

func ExamplePostProcessor() {
	m := mediator.New()

	err := mediator.RegisterRequestHandler(
		m,
		mediator.RequestHandlerFunc[pingRequest, string](
			func(_ context.Context, request pingRequest) (string, error) {
				return "pong:" + request.Message, nil
			},
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = mediator.RegisterPipelineBehavior(
		m,
		mediator.PostProcessor[pingRequest, string](
			func(_ context.Context, request pingRequest, response string) error {
				fmt.Println("handled", request.Message, "as", response)
				return nil
			},
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	response, err := mediator.Send[pingRequest, string](
		context.Background(),
		m,
		pingRequest{Message: testHello},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response)
	// Output:
	// handled hello as pong:hello
	// pong:hello
}

func ExampleStream() {
	m := mediator.New()

	err := mediator.RegisterStreamHandler(
		m,
		mediator.StreamHandlerFunc[pingRequest, string](
			func(ctx context.Context, request pingRequest, yield mediator.StreamYield[string]) error {
				for _, suffix := range []string{"one", "two"} {
					if err := yield(ctx, request.Message+":"+suffix); err != nil {
						return err
					}
				}

				return nil
			},
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = mediator.Stream(
		context.Background(),
		m,
		pingRequest{Message: "item"},
		func(_ context.Context, item string) error {
			fmt.Println(item)
			return nil
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// item:one
	// item:two
}
