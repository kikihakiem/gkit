package jetstream_test

import (
	"context"
	"os"
	"testing"
	"time"

	natstransport "github.com/kikihakiem/jetstream-transport"
	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type TestResponse struct {
	String string `json:"str"`
	Error  string `json:"err"`
}

func newNATSConn(t *testing.T) (*server.Server, *nats.Conn) {
	t.Helper()

	opts := natsserver.DefaultTestOptions
	opts.Port = -1
	opts.JetStream = true

	srv := natsserver.RunServer(&opts)

	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	return srv, nc
}

func newJetstream(t *testing.T) (jetstream.JetStream, jetstream.Stream, func()) {
	t.Helper()

	srv, nc := newNATSConn(t)

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "testStream",
		Subjects: []string{"testSubject"},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	return js, stream, func() {
		shutdownJSServerAndRemoveStorage(t, srv)
	}
}

func newConsumer(t *testing.T, handler *natstransport.Subscriber) (jetstream.JetStream, func()) {
	t.Helper()

	srv, nc := newNATSConn(t)

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "testStream",
		Subjects: []string{"testSubject"},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable: "testConsumer",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	consumeCtx, err := consumer.Consume(handler.HandleMessage(js))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	return js, func() {
		consumeCtx.Stop()
		shutdownJSServerAndRemoveStorage(t, srv)
	}
}

func shutdownJSServerAndRemoveStorage(t *testing.T, s *server.Server) {
	t.Helper()

	var sd string
	if config := s.JetStreamConfig(); config != nil {
		sd = config.StoreDir
	}
	s.Shutdown()
	if sd != "" {
		if err := os.RemoveAll(sd); err != nil {
			t.Fatalf("Unable to remove storage %q: %v", sd, err)
		}
	}
	s.WaitForShutdown()
}
