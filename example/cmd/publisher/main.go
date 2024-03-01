package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	if len(os.Args) < 3 {
		slog.Error("missing required NATS server URL and/or payload")

		return
	}

	natsURL := os.Args[1]
	payload := os.Args[2]

	slog.Info("publishing a message to NATS server", slog.String("url", natsURL), slog.String("payload", payload))

	nc, err := nats.Connect(natsURL)
	if err != nil {
		slog.Error("failed to connect to NATS server", slog.String("error", err.Error()))

		return
	}

	js, err := jetstream.New(nc)
	if err != nil {
		slog.Error("failed to init Jetstream", slog.String("error", err.Error()))

		return
	}

	js.Publish(context.Background(), "events.create", []byte(payload))
}
