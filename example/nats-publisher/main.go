package main

import (
	"context"
	"log"

	"github.com/onbyzerollc/pubsub"
	"github.com/onbyzerollc/pubsub/middleware/defaults"
	"github.com/onbyzerollc/pubsub/providers/nats"
)

const HelloTopic = "hello.topic"

type HelloMsg struct {
	Greeting string
	Name     string
}

func main() {
	n, err := nats.NewNats("test-cluster")
	if err != nil {
		log.Fatal(err)
	}

	pubsub.SetClient(&pubsub.Client{
		ServiceName: "my-new-service",
		Provider:    n,
		Middleware:  defaults.Middleware,
	})

	r := pubsub.PublishJSON(context.Background(), HelloTopic, &HelloMsg{Greeting: "Hello", Name: "Alex"})
	if r.Err != nil {
		log.Panic(r.Err)
	}

	<-r.Ready
}
