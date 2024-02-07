//go:build unit

package jetstream_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/transport"
	jstransport "github.com/kikihakiem/jetstream-transport"
)

func TestSubscriberBadDecode(t *testing.T) {
	var (
		reqDecoder = func(ctx context.Context, m jetstream.Msg) (request interface{}, err error) {
			return struct{}{}, errors.New("dang")
		}
		errChan = make(chan error, 1)
	)

	handler := jstransport.NewSubscriber(
		endpoint.Nop,
		reqDecoder,
		jstransport.NopResponseEncoder,
		jstransport.SubscriberErrorHandler(transport.ErrorHandlerFunc(func(ctx context.Context, err error) {
			errChan <- err
		})),
	)

	js, stop := newConsumer(t, handler)
	defer stop()

	publish(t, js, "test data")

	if want, have := "dang", (<-errChan).Error(); want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

func TestSubscriberBadEndpoint(t *testing.T) {
	var (
		endpt   = func(context.Context, interface{}) (interface{}, error) { return struct{}{}, errors.New("dang") }
		errChan = make(chan error, 1)
	)

	handler := jstransport.NewSubscriber(
		endpt,
		jstransport.NopRequestDecoder,
		jstransport.NopResponseEncoder,
		jstransport.SubscriberErrorHandler(transport.ErrorHandlerFunc(func(ctx context.Context, err error) {
			errChan <- err
		})),
	)

	js, stop := newConsumer(t, handler)
	defer stop()

	publish(t, js, "test data")

	if want, have := "dang", (<-errChan).Error(); want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

func TestSubscriberBadEncode(t *testing.T) {
	var (
		respEncoder = func(ctx context.Context, s string, js jetstream.JetStream, i interface{}) error {
			return errors.New("dang")
		}
		errChan = make(chan error, 1)
	)

	handler := jstransport.NewSubscriber(
		endpoint.Nop,
		jstransport.NopRequestDecoder,
		respEncoder,
		jstransport.SubscriberErrorHandler(transport.ErrorHandlerFunc(func(ctx context.Context, err error) {
			errChan <- err
		})),
	)

	js, stop := newConsumer(t, handler)
	defer stop()

	publish(t, js, "test data")

	if want, have := "dang", (<-errChan).Error(); want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

func TestSubscriberErrorEncoder(t *testing.T) {
	var (
		respEncoder = func(ctx context.Context, s string, js jetstream.JetStream, i interface{}) error {
			return errors.New("dang")
		}
		resChan = make(chan string, 1)
	)

	handler := jstransport.NewSubscriber(
		endpoint.Nop,
		jstransport.NopRequestDecoder,
		respEncoder,
		jstransport.SubscriberErrorEncoder(func(ctx context.Context, err error, reply string, js jetstream.JetStream) {
			future, _ := js.PublishAsync(reply, []byte(err.Error()))
			resChan <- string(future.Msg().Data)
		}),
	)

	js, stop := newConsumer(t, handler)
	defer stop()

	publish(t, js, "test data")

	if want, have := "dang", (<-resChan); want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

func TestSubscriberHappyPath(t *testing.T) {
	var (
		endpt = func(_ context.Context, req interface{}) (interface{}, error) {
			request, ok := req.(string)
			if want, have := "test data foo", request; !ok || want != have {
				t.Errorf("want %s, have %s", want, have)
			}

			return request + " bar", nil
		}
		reqDecoder = func(ctx context.Context, m jetstream.Msg) (request interface{}, err error) {
			return string(m.Data()) + " foo", nil
		}
		respEncoder = func(ctx context.Context, s string, js jetstream.JetStream, resp interface{}) error {
			response, ok := resp.(string)
			if want, have := "test data foo bar", response; !ok || want != have {
				t.Errorf("want %s, have %s", want, have)
			}

			return nil
		}
		errChan = make(chan error, 1)
	)

	handler := jstransport.NewSubscriber(
		endpt,
		reqDecoder,
		respEncoder,
		jstransport.SubscriberErrorHandler(transport.ErrorHandlerFunc(func(ctx context.Context, err error) {
			errChan <- err
		})),
	)

	js, stop := newConsumer(t, handler)
	defer stop()

	publish(t, js, "test data")

	select {
	case err := <-errChan:
		t.Errorf("unexpected error: %v", err)
	case <-time.After(time.Second): // expect no error
	}
}

func TestMultipleSubscriberBefore(t *testing.T) {
	errChan := make(chan error, 1)

	handler := jstransport.NewSubscriber(
		endpoint.Nop,
		jstransport.NopRequestDecoder,
		jstransport.NopResponseEncoder,
		jstransport.SubscriberBefore(func(ctx context.Context, m jetstream.Msg) context.Context {
			ctx = context.WithValue(ctx, "one", 1)

			return ctx
		}),
		jstransport.SubscriberBefore(func(ctx context.Context, m jetstream.Msg) context.Context {
			if _, ok := ctx.Value("one").(int); !ok {
				errChan <- errors.New("value was not set properly when multiple SubscriberBefore are used")
			} else {
				errChan <- nil
			}

			return ctx
		}),
	)

	js, stop := newConsumer(t, handler)
	defer stop()

	publish(t, js, "test data")

	if err := <-errChan; err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMultipleSubscriberAfter(t *testing.T) {
	var (
		respEncoder = func(ctx context.Context, s string, js jetstream.JetStream, i interface{}) error {
			return errors.New("dang")
		}
		errChan = make(chan error, 1)
	)

	handler := jstransport.NewSubscriber(
		endpoint.Nop,
		jstransport.NopRequestDecoder,
		respEncoder,
		jstransport.SubscriberAfter(func(ctx context.Context, js jetstream.JetStream) context.Context {
			return context.WithValue(ctx, "one", 1)
		}),
		jstransport.SubscriberAfter(func(ctx context.Context, js jetstream.JetStream) context.Context {
			if _, ok := ctx.Value("one").(int); !ok {
				errChan <- errors.New("value was not set properly when multiple SubscriberAfter are used")
			} else {
				errChan <- nil
			}

			return ctx
		}),
	)

	js, stop := newConsumer(t, handler)
	defer stop()

	publish(t, js, "test data")

	if err := <-errChan; err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSubscriberFinalizerFunc(t *testing.T) {
	errChan := make(chan error, 1)

	handler := jstransport.NewSubscriber(
		endpoint.Nop,
		jstransport.NopRequestDecoder,
		jstransport.NopResponseEncoder,
		jstransport.SubscriberAfter(func(ctx context.Context, js jetstream.JetStream) context.Context {
			return context.WithValue(ctx, "one", 1)
		}),
		jstransport.SubscriberFinalizer(func(ctx context.Context, msg jetstream.Msg, resp any) {
			if _, ok := ctx.Value("one").(int); !ok {
				errChan <- errors.New("value was not set properly when multiple SubscriberAfter are used")
			} else {
				errChan <- nil
			}
		}),
	)

	js, stop := newConsumer(t, handler)
	defer stop()

	publish(t, js, "test data")

	if err := <-errChan; err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEncodeJSONResponse(t *testing.T) {
	dataChan := make(chan string, 1)

	handler := jstransport.NewSubscriber(
		func(context.Context, interface{}) (interface{}, error) {
			return struct {
				Foo string `json:"foo"`
			}{"bar"}, nil
		},
		jstransport.NopRequestDecoder,
		jstransport.EncodeJSONResponse,
	)

	handler.HandleMessage(&jetstreamMock{dataChan: dataChan})(&jetstreamMsg{replySubject: "foo"})

	if want, have := `{"foo":"bar"}`, strings.TrimSpace(<-dataChan); want != have {
		t.Errorf("Body: want %s, have %s", want, have)
	}
}

func TestDefaultErrorEncoder(t *testing.T) {
	dataChan := make(chan string, 1)

	handler := jstransport.NewSubscriber(
		func(context.Context, interface{}) (interface{}, error) {
			return nil, errors.New("dang")
		},
		jstransport.NopRequestDecoder,
		jstransport.NopResponseEncoder,
		jstransport.SubscriberErrorEncoder(jstransport.DefaultErrorEncoder),
	)

	handler.HandleMessage(&jetstreamMock{dataChan: dataChan})(&jetstreamMsg{replySubject: "foo"})

	if want, have := `{"err":"dang"}`, strings.TrimSpace(<-dataChan); want != have {
		t.Errorf("Body: want %s, have %s", want, have)
	}
}

func TestNoOpRequestDecoder(t *testing.T) {
	errChan := make(chan error, 1)

	handler := jstransport.NewSubscriber(
		func(ctx context.Context, request interface{}) (interface{}, error) {
			if request != nil {
				errChan <- errors.New("expected nil request in endpoint when using NopRequestDecoder")
			} else {
				errChan <- nil
			}

			return nil, nil
		},
		jstransport.NopRequestDecoder,
		jstransport.EncodeJSONResponse,
	)

	js, stop := newConsumer(t, handler)
	defer stop()

	publish(t, js, "test data")

	if err := <-errChan; err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
