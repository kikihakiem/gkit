//go:build unit

package jetstream_test

import (
	"context"
	"testing"
	"time"

	jstransport "github.com/kikihakiem/jetstream-transport"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestPublisher(t *testing.T) {
	js, _, stop := newJetstream(context.Background(), t)
	defer stop()

	publisher := jstransport.NewPublisher(
		js,
		"natstransport.test.1",
		jstransport.NopRequestEncoder,
		jstransport.NopResponseDecoder,
	)

	res, err := publisher.Endpoint()(context.Background(), struct{}{})
	if err != nil {
		t.Fatal(err)
	}

	ack, ok := res.(*jetstream.PubAck)
	if !ok {
		t.Fatal("response should be a *jetstream.PubAck")
	}

	if want, have := "test:stream", ack.Stream; want != have {
		t.Errorf("want %q, have %q", want, have)
	}
}

func TestPublisherBefore(t *testing.T) {
	var (
		testData   = "foobar"
		reqEncoder = func(_ context.Context, msg *nats.Msg, req interface{}) error {
			request := req.(string)
			msg.Data = []byte(request)

			return nil
		}
	)

	js, _, stop := newJetstream(context.Background(), t)
	defer stop()

	publisher := jstransport.NewPublisher(
		js,
		"natstransport.test.1",
		reqEncoder,
		jstransport.NopResponseDecoder,
		jstransport.PublisherBefore(func(ctx context.Context, msg *nats.Msg) context.Context {
			if want, have := testData, string(msg.Data); want != have {
				t.Errorf("want %q, have %q", want, have)
			}
			return ctx
		}),
	)

	res, err := publisher.Endpoint()(context.Background(), testData)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := res.(*jetstream.PubAck)
	if !ok {
		t.Fatal("response should be a *jetstream.PubAck")
	}
}

func TestPublisherAfter(t *testing.T) {
	alteredStreamName := "foo"

	js, _, stop := newJetstream(context.Background(), t)
	defer stop()

	publisher := jstransport.NewPublisher(
		js,
		"natstransport.test.2",
		jstransport.NopRequestEncoder,
		jstransport.NopResponseDecoder,
		jstransport.PublisherAfter(func(ctx context.Context, pa *jetstream.PubAck) context.Context {
			pa.Stream = alteredStreamName // alter the stream name just to check if this function is called
			return ctx
		}),
	)

	res, err := publisher.Endpoint()(context.Background(), struct{}{})
	if err != nil {
		t.Fatal(err)
	}

	response, ok := res.(*jetstream.PubAck)
	if !ok {
		t.Fatal("response should be a *jetstream.PubAck")
	}

	if want, have := alteredStreamName, response.Stream; want != have {
		t.Errorf("want %q, have %q", want, have)
	}
}

func TestPublisherTimeout(t *testing.T) {
	js, _, stop := newJetstream(context.Background(), t)
	defer stop()

	publisher := jstransport.NewPublisher(
		js,
		"natstransport.test.3",
		jstransport.NopRequestEncoder,
		jstransport.NopResponseDecoder,
		jstransport.PublisherTimeout(time.Nanosecond), // set a very low timeout
	)

	_, err := publisher.Endpoint()(context.Background(), struct{}{})
	if err != context.DeadlineExceeded {
		t.Errorf("want %s, have %s", context.DeadlineExceeded, err)
	}
}

func TestPublisherCancellation(t *testing.T) {
	js, _, stop := newJetstream(context.Background(), t)
	defer stop()

	publisher := jstransport.NewPublisher(
		js,
		"natstransport.test.4",
		jstransport.NopRequestEncoder,
		jstransport.NopResponseDecoder,
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := publisher.Endpoint()(ctx, struct{}{})
	if err != context.Canceled {
		t.Errorf("want %s, have %s", context.Canceled, err)
	}
}

func TestEncodeJSONRequest(t *testing.T) {
	encoded := make(chan string, 1)

	js, _, stop := newJetstream(context.Background(), t)
	defer stop()

	publisher := jstransport.NewPublisher(
		js,
		"natstransport.test.5",
		jstransport.EncodeJSONRequest,
		jstransport.NopResponseDecoder,
		jstransport.PublisherBefore(func(ctx context.Context, msg *nats.Msg) context.Context {
			encoded <- string(msg.Data)
			return ctx
		}),
	).Endpoint()

	for _, test := range []struct {
		value interface{}
		body  string
	}{
		{nil, "null"},
		{12, "12"},
		{1.2, "1.2"},
		{true, "true"},
		{"test", "\"test\""},
		{struct {
			Foo string `json:"foo"`
		}{"foo"}, "{\"foo\":\"foo\"}"},
	} {
		if _, err := publisher(context.Background(), test.value); err != nil {
			t.Fatal(err)
			continue
		}

		if data := <-encoded; data != test.body {
			t.Errorf("%v: actual %#v, expected %#v", test.value, data, test.body)
		}
	}
}
