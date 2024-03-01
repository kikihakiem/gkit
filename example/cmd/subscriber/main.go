package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kikihakiem/gkit/example/internal/audit"
	"github.com/kikihakiem/gkit/example/internal/repository"
	"github.com/kikihakiem/gkit/example/internal/transport"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	natsserver "github.com/nats-io/nats-server/v2/test"
)

func main() {
	eventRepo := repository.NewEventRepositroy()
	eventSvc := audit.NewEventService(eventRepo)

	opts := natsserver.DefaultTestOptions
	opts.Port = -1
	opts.JetStream = true
	srv := natsserver.RunServer(&opts)

	slog.Info("running NATS server... Please publish messages using this URL", slog.String("url", srv.ClientURL()))

	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		slog.Error("failed to connect to NATS server", slog.String("error", err.Error()))

		return
	}

	js, err := jetstream.New(nc)
	if err != nil {
		slog.Error("failed to init Jetstream", slog.String("error", err.Error()))

		return
	}

	ctx := context.Background()

	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     "events:create",
		Subjects: []string{"events.create"},
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to subscribe to tasks stream", slog.String("error", err.Error()))

		return
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable: "auditEvent",
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to create or update consumer", slog.String("error", err.Error()))

		return
	}

	consumeContext, err := consumer.Consume(transport.CreateEventJetstreamHandler(js, eventSvc))
	if err != nil {
		slog.ErrorContext(ctx, "failed to start consumer", slog.String("error", err.Error()))

		return
	}

	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-sigChannel
	slog.InfoContext(ctx, "received OS signal. Exiting...", slog.String("signal", sig.String()))
	consumeContext.Stop()
}
