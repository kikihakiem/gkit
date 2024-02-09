//go:build unit

package jetstream_test

import (
	"context"
	"testing"
	"time"

	jstransport "github.com/kikihakiem/jetstream-transport"
	"github.com/kikihakiem/jetstream-transport/gkit"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestPublisher(t *testing.T) {
	js, _, stop := newJetstream(context.Background(), t)
	defer stop()

	publisher := jstransport.NewPublisher[struct{}, *jetstream.PubAck](
		js,
		jstransport.EncodeJSONRequest,
		gkit.PassThroughResponseDecoder,
	)

	res, err := publisher.Endpoint()(context.Background(), struct{}{})
	if err != nil {
		t.Fatal(err)
	}

	if want, have := "test:stream", res.Stream; want != have {
		t.Errorf("want %q, have %q", want, have)
	}
}

func TestPublisherBefore(t *testing.T) {
	var (
		testData   = "foobar"
		reqEncoder = func(_ context.Context, req string) (*nats.Msg, error) {
			msg := nats.NewMsg("natstransport.response")
			msg.Data = []byte(req)

			return msg, nil
		}
	)

	js, _, stop := newJetstream(context.Background(), t)
	defer stop()

	publisher := jstransport.NewPublisher[string, *jetstream.PubAck](
		js,
		reqEncoder,
		gkit.NopResponseDecoder,
		jstransport.PublisherBefore[string, *jetstream.PubAck](func(ctx context.Context, msg *nats.Msg) context.Context {
			if want, have := testData, string(msg.Data); want != have {
				t.Errorf("want %q, have %q", want, have)
			}
			return ctx
		}),
	)

	_, err := publisher.Endpoint()(context.Background(), testData)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPublisherAfter(t *testing.T) {
	alteredStreamName := "foo"

	js, _, stop := newJetstream(context.Background(), t)
	defer stop()

	publisher := jstransport.NewPublisher[struct{}, *jetstream.PubAck](
		js,
		jstransport.EncodeJSONRequest,
		gkit.PassThroughResponseDecoder,
		jstransport.PublisherAfter[struct{}, *jetstream.PubAck](func(ctx context.Context, pa *jetstream.PubAck, err error) context.Context {
			pa.Stream = alteredStreamName // alter the stream name just to check if this function is called
			return ctx
		}),
	)

	res, err := publisher.Endpoint()(context.Background(), struct{}{})
	if err != nil {
		t.Fatal(err)
	}

	if want, have := alteredStreamName, res.Stream; want != have {
		t.Errorf("want %q, have %q", want, have)
	}
}

func TestPublisherTimeout(t *testing.T) {
	js, _, stop := newJetstream(context.Background(), t)
	defer stop()

	publisher := jstransport.NewPublisher[struct{}, *jetstream.PubAck](
		js,
		jstransport.EncodeJSONRequest,
		gkit.NopResponseDecoder,
		jstransport.PublisherTimeout[struct{}, *jetstream.PubAck](time.Nanosecond), // set a very low timeout
	)

	_, err := publisher.Endpoint()(context.Background(), struct{}{})
	if err != context.DeadlineExceeded {
		t.Errorf("want %s, have %s", context.DeadlineExceeded, err)
	}
}

func TestPublisherCancellation(t *testing.T) {
	js, _, stop := newJetstream(context.Background(), t)
	defer stop()

	publisher := jstransport.NewPublisher[struct{}, *jetstream.PubAck](
		js,
		jstransport.EncodeJSONRequest,
		gkit.NopResponseDecoder,
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

	publisher := jstransport.NewPublisher[any, *jetstream.PubAck](
		js,
		jstransport.EncodeJSONRequest,
		gkit.NopResponseDecoder,
		jstransport.PublisherBefore[any, *jetstream.PubAck](func(ctx context.Context, msg *nats.Msg) context.Context {
			encoded <- string(msg.Data)
			return ctx
		}),
	).Endpoint()

	for _, test := range []struct {
		value any
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
