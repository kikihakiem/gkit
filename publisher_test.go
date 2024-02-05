//go:build unit

package jetstream_test

import (
	"context"
	"testing"
	"time"

	natstransport "github.com/kikihakiem/jetstream-transport"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestPublisher(t *testing.T) {
	var (
		encode = func(context.Context, *nats.Msg, interface{}) error { return nil }
		decode = func(_ context.Context, pa *jetstream.PubAck) (interface{}, error) {
			return pa, nil
		}
	)

	js, _, stop := newJetstream(context.Background(), t)
	defer stop()

	publisher := natstransport.NewPublisher(
		js,
		"natstransport.test.1",
		encode,
		decode,
	)

	res, err := publisher.Endpoint()(context.Background(), struct{}{})
	if err != nil {
		t.Fatal(err)
	}

	response, ok := res.(*jetstream.PubAck)
	if !ok {
		t.Fatal("response should be a *jetstream.PubAck")
	}
	if want, have := "test:stream", response.Stream; want != have {
		t.Errorf("want %q, have %q", want, have)
	}
}

func TestPublisherBefore(t *testing.T) {
	var (
		testData = "foobar"
		encode   = func(_ context.Context, msg *nats.Msg, req interface{}) error {
			request := req.(string)
			msg.Data = []byte(request)

			return nil
		}
		decode = func(_ context.Context, pa *jetstream.PubAck) (interface{}, error) {
			return pa, nil
		}
	)

	js, _, stop := newJetstream(context.Background(), t)
	defer stop()

	publisher := natstransport.NewPublisher(
		js,
		"natstransport.test.1",
		encode,
		decode,
		natstransport.PublisherBefore(func(ctx context.Context, msg *nats.Msg) context.Context {
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
	var (
		alteredStreamName = "foo"
		encode            = func(context.Context, *nats.Msg, interface{}) error { return nil }
		decode            = func(_ context.Context, pa *jetstream.PubAck) (interface{}, error) {
			return pa, nil
		}
	)

	js, _, stop := newJetstream(context.Background(), t)
	defer stop()

	publisher := natstransport.NewPublisher(
		js,
		"natstransport.test.2",
		encode,
		decode,
		natstransport.PublisherAfter(func(ctx context.Context, pa *jetstream.PubAck) context.Context {
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
	var (
		encode = func(context.Context, *nats.Msg, interface{}) error { return nil }
		decode = func(_ context.Context, pa *jetstream.PubAck) (interface{}, error) {
			return pa, nil
		}
	)

	js, _, stop := newJetstream(context.Background(), t)
	defer stop()

	publisher := natstransport.NewPublisher(
		js,
		"natstransport.test.3",
		encode,
		decode,
		natstransport.PublisherTimeout(time.Nanosecond), // set a very low timeout
	)

	_, err := publisher.Endpoint()(context.Background(), struct{}{})
	if err != context.DeadlineExceeded {
		t.Errorf("want %s, have %s", context.DeadlineExceeded, err)
	}
}

func TestPublisherCancellation(t *testing.T) {
	var (
		encode = func(context.Context, *nats.Msg, interface{}) error { return nil }
		decode = func(_ context.Context, pa *jetstream.PubAck) (interface{}, error) {
			return pa, nil
		}
	)

	js, _, stop := newJetstream(context.Background(), t)
	defer stop()

	publisher := natstransport.NewPublisher(
		js,
		"natstransport.test.4",
		encode,
		decode,
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

	publisher := natstransport.NewPublisher(
		js,
		"natstransport.test.5",
		natstransport.EncodeJSONRequest,
		func(ctx context.Context, pa *jetstream.PubAck) (response interface{}, err error) { return nil, nil },
		natstransport.PublisherBefore(func(ctx context.Context, msg *nats.Msg) context.Context {
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
