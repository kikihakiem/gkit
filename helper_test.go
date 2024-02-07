//go:build unit

package jetstream_test

import (
	"context"
	"testing"
	"time"

	natstransport "github.com/kikihakiem/jetstream-transport"
	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

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

func newJetstream(ctx context.Context, t *testing.T) (jetstream.JetStream, jetstream.Stream, func()) {
	t.Helper()

	srv, nc := newNATSConn(t)

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	stream, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "test:stream",
		Subjects: []string{"natstransport.>"},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	return js, stream, func() {
		nc.Close()
		shutdownJSServer(t, srv)
	}
}

func newConsumer(t *testing.T, handler *natstransport.Subscriber) (jetstream.JetStream, func()) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	js, stream, stopServer := newJetstream(ctx, t)
	err := stream.Purge(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	consumeCtx, err := consumer.Consume(handler.HandleMessage(js))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	return js, func() {
		consumeCtx.Stop()
		stopServer()
	}
}

func shutdownJSServer(t *testing.T, s *server.Server) {
	t.Helper()

	s.Shutdown()
	s.WaitForShutdown()
}

func publish(t *testing.T, js jetstream.JetStream, message string) {
	_, err := js.Publish(context.Background(), "natstransport.test.99", []byte(message))
	if err != nil {
		t.Fatal(err)
	}
}

type jetstreamMock struct {
	jetstream.JetStream
	dataChan chan string
}

func (jm *jetstreamMock) Publish(ctx context.Context, subject string, data []byte, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
	go func() {
		jm.dataChan <- string(data)
	}()

	return nil, nil
}

type jetstreamMsg struct {
	data         []byte
	replySubject string
}

func (jm *jetstreamMsg) Metadata() (*jetstream.MsgMetadata, error) {
	return nil, nil
}

func (jm *jetstreamMsg) Data() []byte {
	return jm.data
}

func (jm *jetstreamMsg) Headers() nats.Header {
	return nil
}

func (jm *jetstreamMsg) Subject() string {
	return ""
}

func (jm *jetstreamMsg) Reply() string {
	return jm.replySubject
}

func (jm *jetstreamMsg) Ack() error {
	return nil
}

func (jm *jetstreamMsg) DoubleAck(context.Context) error {
	return nil
}

func (jm *jetstreamMsg) Nak() error {
	return nil
}

func (jm *jetstreamMsg) NakWithDelay(delay time.Duration) error {
	return nil
}

func (jm *jetstreamMsg) InProgress() error {
	return nil
}

func (jm *jetstreamMsg) Term() error {
	return nil
}
