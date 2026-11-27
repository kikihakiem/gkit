//go:build unit

package jetstream_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	gkit "github.com/kikihakiem/gkit/core"
	jstransport "github.com/kikihakiem/gkit/transport/jetstream"
)

type emptyStruct struct{}

func TestSubscriberBadDecode(t *testing.T) {
	var (
		reqDecoder = func(context.Context, jetstream.Msg) (emptyStruct, error) {
			return emptyStruct{}, errors.New("dang")
		}
		errChan = make(chan error, 1)
	)

	handler := jstransport.NewSubscriber(
		gkit.NopEndpoint[emptyStruct, emptyStruct],
		reqDecoder,
		gkit.NopResponseEncoder,
		jstransport.SubscriberErrorHandler[emptyStruct, emptyStruct](gkit.ErrorHandlerFunc(func(ctx context.Context, err error) {
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
		endpt   = func(context.Context, emptyStruct) (emptyStruct, error) { return emptyStruct{}, errors.New("dang") }
		errChan = make(chan error, 1)
	)

	handler := jstransport.NewSubscriber(
		endpt,
		gkit.NopEncoderDecoder,
		gkit.NopResponseEncoder,
		jstransport.SubscriberErrorHandler[emptyStruct, emptyStruct](gkit.ErrorHandlerFunc(func(ctx context.Context, err error) {
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
		respEncoder = func(context.Context, jetstream.JetStream, emptyStruct) error {
			return errors.New("dang")
		}
		errChan = make(chan error, 1)
	)

	handler := jstransport.NewSubscriber(
		gkit.NopEndpoint[emptyStruct, emptyStruct],
		gkit.NopEncoderDecoder,
		respEncoder,
		jstransport.SubscriberErrorHandler[emptyStruct, emptyStruct](gkit.ErrorHandlerFunc(func(ctx context.Context, err error) {
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
		respEncoder = func(context.Context, jetstream.JetStream, emptyStruct) error {
			return errors.New("dang")
		}
		resChan = make(chan string, 1)
	)

	handler := jstransport.NewSubscriber(
		gkit.NopEndpoint[emptyStruct, emptyStruct],
		gkit.NopEncoderDecoder,
		respEncoder,
		jstransport.SubscriberErrorEncoder[emptyStruct, emptyStruct](func(ctx context.Context, js jetstream.JetStream, err error) {
			resChan <- err.Error()
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
		endpt = func(_ context.Context, req string) (string, error) {
			if want, have := "test data foo", req; want != have {
				t.Errorf("want %s, have %s", want, have)
			}

			return req + " bar", nil
		}
		reqDecoder = func(ctx context.Context, m jetstream.Msg) (string, error) {
			return string(m.Data()) + " foo", nil
		}
		respEncoder = func(ctx context.Context, js jetstream.JetStream, resp string) error {
			if want, have := "test data foo bar", resp; want != have {
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
		jstransport.SubscriberErrorHandler[string, string](gkit.ErrorHandlerFunc(func(ctx context.Context, err error) {
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
		gkit.NopEndpoint[emptyStruct, emptyStruct],
		gkit.NopEncoderDecoder,
		gkit.NopResponseEncoder,
		jstransport.SubscriberBefore[emptyStruct, emptyStruct](func(ctx context.Context, _ jetstream.Msg) context.Context {
			ctx = context.WithValue(ctx, "one", 1)

			return ctx
		}),
		jstransport.SubscriberBefore[emptyStruct, emptyStruct](func(ctx context.Context, _ jetstream.Msg) context.Context {
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
		respEncoder = func(context.Context, jetstream.JetStream, emptyStruct) error {
			return errors.New("dang")
		}
		errChan = make(chan error, 1)
	)

	handler := jstransport.NewSubscriber(
		gkit.NopEndpoint[emptyStruct, emptyStruct],
		gkit.NopEncoderDecoder,
		respEncoder,
		jstransport.SubscriberAfter[emptyStruct](func(ctx context.Context, resp emptyStruct) context.Context {
			return context.WithValue(ctx, "one", 1)
		}),
		jstransport.SubscriberAfter[emptyStruct](func(ctx context.Context, resp emptyStruct) context.Context {
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
		gkit.NopEndpoint[emptyStruct, emptyStruct],
		gkit.NopEncoderDecoder,
		gkit.NopResponseEncoder,
		jstransport.SubscriberAfter[emptyStruct](func(ctx context.Context, resp emptyStruct) context.Context {
			return context.WithValue(ctx, "one", 1)
		}),
		jstransport.SubscriberFinalizer[emptyStruct, emptyStruct](func(ctx context.Context, req jetstream.Msg, err error) {
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
	var (
		dataChan    = make(chan string, 1)
		subjectChan = make(chan string, 1)
		subject     = "jstransport.test"
	)

	type foo struct {
		Foo string `json:"foo"`
	}

	ctx := context.WithValue(context.Background(), jstransport.ContextKeyResponseSubject, subject)

	err := jstransport.EncodeJSONResponse(ctx, &jetstreamMock{dataChan: dataChan, subjectChan: subjectChan}, foo{Foo: "bar"})
	if err != nil {
		t.Fatal(err)
	}

	if want, have := subject, <-subjectChan; want != have {
		t.Errorf("Subject: want %s, have %s", want, have)
	}

	if want, have := `{"foo":"bar"}`, strings.TrimSpace(<-dataChan); want != have {
		t.Errorf("Body: want %s, have %s", want, have)
	}
}

func TestEncodeJSONResponseNoSubject(t *testing.T) {
	err := jstransport.EncodeJSONResponse(context.Background(), &jetstreamMock{dataChan: make(chan string, 1)}, struct{}{})
	if !errors.Is(err, jstransport.ErrNoResponseSubject) {
		t.Errorf("want %v, have %v", jstransport.ErrNoResponseSubject, err)
	}
}

func TestEncodeJSONError(t *testing.T) {
	var (
		dataChan    = make(chan string, 1)
		subjectChan = make(chan string, 1)
		subject     = "jstransport.test"
	)

	ctx := context.WithValue(context.Background(), jstransport.ContextKeyResponseSubject, subject)

	jstransport.EncodeJSONError(ctx, &jetstreamMock{dataChan: dataChan, subjectChan: subjectChan}, errors.New("dang"))

	if want, have := subject, <-subjectChan; want != have {
		t.Errorf("Subject: want %s, have %s", want, have)
	}

	if want, have := `{"err":"dang"}`, strings.TrimSpace(<-dataChan); want != have {
		t.Errorf("Body: want %s, have %s", want, have)
	}
}

func TestSubscriberResponseSubject(t *testing.T) {
	const subject = "jstransport.test"

	subjectChan := make(chan string, 1)

	handler := jstransport.NewSubscriber(
		gkit.NopEndpoint[emptyStruct, emptyStruct],
		gkit.NopEncoderDecoder,
		gkit.NopResponseEncoder,
		jstransport.SubscriberResponseSubject[emptyStruct, emptyStruct](subject),
		jstransport.SubscriberFinalizer[emptyStruct, emptyStruct](func(ctx context.Context, _ jetstream.Msg, _ error) {
			got, _ := ctx.Value(jstransport.ContextKeyResponseSubject).(string)
			subjectChan <- got
		}),
	)

	js, stop := newConsumer(t, handler)
	defer stop()

	publish(t, js, "test data")

	if want, have := subject, <-subjectChan; want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

func TestErrorLogger(t *testing.T) {
	errChan := make(chan error, 1)

	handler := jstransport.NewSubscriber(
		func(context.Context, emptyStruct) (emptyStruct, error) {
			return emptyStruct{}, errors.New("dang")
		},
		gkit.NopEncoderDecoder,
		gkit.NopResponseEncoder,
		jstransport.SubscriberErrorLogger[emptyStruct, emptyStruct](func(ctx context.Context, err error) {
			errChan <- err
		}),
	)

	js, stop := newConsumer(t, handler)
	defer stop()

	publish(t, js, "test data")

	if want, have := "dang", (<-errChan).Error(); want != have {
		t.Errorf("Body: want %s, have %s", want, have)
	}
}

func TestMultipleSubscriberFinalizers(t *testing.T) {
	finChan := make(chan struct{}, 2)

	finalizer := jstransport.SubscriberFinalizer[emptyStruct, emptyStruct](func(context.Context, jetstream.Msg, error) {
		finChan <- struct{}{}
	})

	handler := jstransport.NewSubscriber(
		gkit.NopEndpoint[emptyStruct, emptyStruct],
		gkit.NopEncoderDecoder,
		gkit.NopResponseEncoder,
		finalizer,
		finalizer,
	)

	js, stop := newConsumer(t, handler)
	defer stop()

	publish(t, js, "test data")

	for i := 1; i <= 2; i++ {
		select {
		case <-finChan:
		case <-time.After(time.Second):
			t.Fatalf("finalizer %d did not run", i)
		}
	}
}

func TestNoOpRequestDecoder(t *testing.T) {
	errChan := make(chan error, 1)

	handler := jstransport.NewSubscriber(
		func(ctx context.Context, request any) (any, error) {
			if request != nil {
				errChan <- errors.New("expected nil request in endpoint when using NopRequestDecoder")
			} else {
				errChan <- nil
			}

			return nil, nil
		},
		gkit.NopEncoderDecoder,
		jstransport.EncodeJSONResponse,
		jstransport.SubscriberResponseSubject[any, any]("jstransport.test"),
	)

	js, stop := newConsumer(t, handler)
	defer stop()

	publish(t, js, "test data")

	if err := <-errChan; err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
