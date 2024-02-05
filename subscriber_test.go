//go:build unit

package jetstream_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/go-kit/kit/endpoint"
	natstransport "github.com/kikihakiem/jetstream-transport"
)

func TestSubscriberBadDecode(t *testing.T) {
	s, c := newNATSConn(t)
	defer func() { s.Shutdown(); s.WaitForShutdown() }()
	defer c.Close()

	handler := natstransport.NewSubscriber(
		func(context.Context, interface{}) (interface{}, error) { return struct{}{}, nil },
		func(ctx context.Context, m jetstream.Msg) (request interface{}, err error) {
			return struct{}{}, errors.New("dang")
		},
		func(ctx context.Context, s string, js jetstream.JetStream, i interface{}) error { return nil },
	)

	resp := testRequest(t, c, handler)

	if want, have := "dang", resp.Error; want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

func TestSubscriberBadEndpoint(t *testing.T) {
	s, c := newNATSConn(t)
	defer func() { s.Shutdown(); s.WaitForShutdown() }()
	defer c.Close()

	handler := natstransport.NewSubscriber(
		func(context.Context, interface{}) (interface{}, error) { return struct{}{}, errors.New("dang") },
		func(ctx context.Context, m jetstream.Msg) (request interface{}, err error) { return struct{}{}, nil },
		func(ctx context.Context, s string, js jetstream.JetStream, i interface{}) error { return nil },
	)

	resp := testRequest(t, c, handler)

	if want, have := "dang", resp.Error; want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

func TestSubscriberBadEncode(t *testing.T) {
	s, c := newNATSConn(t)
	defer func() { s.Shutdown(); s.WaitForShutdown() }()
	defer c.Close()

	handler := natstransport.NewSubscriber(
		func(context.Context, interface{}) (interface{}, error) { return struct{}{}, nil },
		func(ctx context.Context, m jetstream.Msg) (request interface{}, err error) { return struct{}{}, nil },
		func(ctx context.Context, s string, js jetstream.JetStream, i interface{}) error {
			return errors.New("dang")
		},
	)

	resp := testRequest(t, c, handler)

	if want, have := "dang", resp.Error; want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

func TestSubscriberErrorEncoder(t *testing.T) {
	s, c := newNATSConn(t)
	defer func() { s.Shutdown(); s.WaitForShutdown() }()
	defer c.Close()

	errTeapot := errors.New("teapot")
	code := func(err error) error {
		if errors.Is(err, errTeapot) {
			return err
		}
		return errors.New("dang")
	}
	handler := natstransport.NewSubscriber(
		func(context.Context, interface{}) (interface{}, error) { return struct{}{}, errTeapot },
		func(ctx context.Context, m jetstream.Msg) (request interface{}, err error) { return struct{}{}, nil },
		func(ctx context.Context, s string, js jetstream.JetStream, i interface{}) error { return nil },
		natstransport.SubscriberErrorEncoder(func(ctx context.Context, err error, reply string, js jetstream.JetStream) {
			var r TestResponse
			r.Error = code(err).Error()

			b, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}

			if err := c.Publish(reply, b); err != nil {
				t.Fatal(err)
			}
		}),
	)

	resp := testRequest(t, c, handler)

	if want, have := errTeapot.Error(), resp.Error; want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

func TestSubscriberHappySubject(t *testing.T) {
	step, response := testSubscriber(t)
	step()
	r := <-response

	var resp TestResponse
	err := json.Unmarshal(r.Data, &resp)
	if err != nil {
		t.Fatal(err)
	}

	if want, have := "", resp.Error; want != have {
		t.Errorf("want %s, have %s (%s)", want, have, r.Data)
	}
}

func TestMultipleSubscriberBefore(t *testing.T) {
	var (
		response = struct{ Body string }{"go eat a fly ugly\n"}
		wg       sync.WaitGroup
		done     = make(chan struct{})
	)
	handler := natstransport.NewSubscriber(
		endpoint.Nop,
		func(ctx context.Context, m jetstream.Msg) (request interface{}, err error) { return struct{}{}, nil },
		func(ctx context.Context, reply string, js jetstream.JetStream, i interface{}) error {
			b, err := json.Marshal(response)
			if err != nil {
				return err
			}

			_, err = js.Publish(ctx, reply, b)
			return err
		},
		natstransport.SubscriberBefore(func(ctx context.Context, m jetstream.Msg) context.Context {
			ctx = context.WithValue(ctx, "one", 1)

			return ctx
		}),
		natstransport.SubscriberBefore(func(ctx context.Context, m jetstream.Msg) context.Context {
			if _, ok := ctx.Value("one").(int); !ok {
				t.Error("Value was not set properly when multiple ServerBefores are used")
			}

			close(done)
			return ctx
		}),
	)

	js, stop := newConsumer(t, handler)
	defer stop()

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := js.Publish(context.Background(), "testSubject", []byte("test data"))
		if err != nil {
			t.Fatal(err)
		}
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for finalizer")
	}

	wg.Wait()
}

func TestMultipleSubscriberAfter(t *testing.T) {
	var (
		response = struct{ Body string }{"go eat a fly ugly\n"}
		wg       sync.WaitGroup
		done     = make(chan struct{})
	)
	handler := natstransport.NewSubscriber(
		endpoint.Nop,
		func(ctx context.Context, m jetstream.Msg) (request interface{}, err error) {
			return struct{}{}, nil
		},
		func(ctx context.Context, reply string, js jetstream.JetStream, i interface{}) error {
			b, err := json.Marshal(response)
			if err != nil {
				return err
			}

			_, err = js.Publish(ctx, reply, b)
			return err
		},
		natstransport.SubscriberAfter(func(ctx context.Context, js jetstream.JetStream) context.Context {
			return context.WithValue(ctx, "one", 1)
		}),
		natstransport.SubscriberAfter(func(ctx context.Context, js jetstream.JetStream) context.Context {
			if _, ok := ctx.Value("one").(int); !ok {
				t.Error("Value was not set properly when multiple ServerAfters are used")
			}
			close(done)
			return ctx
		}),
	)

	js, stop := newConsumer(t, handler)
	defer stop()

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := js.Publish(context.Background(), "testSubject", []byte("test data"))
		if err != nil {
			t.Fatal(err)
		}
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for finalizer")
	}

	wg.Wait()
}

func TestSubscriberFinalizerFunc(t *testing.T) {
	var (
		response = struct{ Body string }{"go eat a fly ugly\n"}
		wg       sync.WaitGroup
		done     = make(chan struct{})
	)
	handler := natstransport.NewSubscriber(
		endpoint.Nop,
		func(ctx context.Context, m jetstream.Msg) (request interface{}, err error) {
			return struct{}{}, nil
		},
		func(ctx context.Context, reply string, js jetstream.JetStream, i interface{}) error {
			b, err := json.Marshal(response)
			if err != nil {
				return err
			}

			_, err = js.Publish(ctx, reply, b)
			return err
		},
		natstransport.SubscriberFinalizer(func(ctx context.Context, msg jetstream.Msg) {
			close(done)
		}),
	)

	js, stop := newConsumer(t, handler)
	defer stop()

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := js.Publish(context.Background(), "testSubject", []byte("test data"))
		if err != nil {
			t.Fatal(err)
		}
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for finalizer")
	}

	wg.Wait()
}

func TestEncodeJSONResponse(t *testing.T) {
	handler := natstransport.NewSubscriber(
		func(context.Context, interface{}) (interface{}, error) {
			return struct {
				Foo string `json:"foo"`
			}{"bar"}, nil
		},
		func(ctx context.Context, m jetstream.Msg) (request interface{}, err error) { return struct{}{}, nil },
		natstransport.EncodeJSONResponse,
	)

	js, stop := newConsumer(t, handler)
	defer stop()

	_, err := js.Publish(context.Background(), "testSubject", []byte("test data"))
	if err != nil {
		t.Fatal(err)
	}

	// if want, have := `{"foo":"bar"}`, strings.TrimSpace(string(r.Data)); want != have {
	// 	t.Errorf("Body: want %s, have %s", want, have)
	// }
}

type responseError struct {
	msg string
}

func (m responseError) Error() string {
	return m.msg
}

func TestErrorEncoder(t *testing.T) {
	errResp := struct {
		Error string `json:"err"`
	}{"oh no"}
	handler := natstransport.NewSubscriber(
		func(context.Context, interface{}) (interface{}, error) {
			return nil, responseError{msg: errResp.Error}
		},
		func(ctx context.Context, m jetstream.Msg) (request interface{}, err error) { return struct{}{}, nil },
		natstransport.EncodeJSONResponse,
	)

	js, stop := newConsumer(t, handler)
	defer stop()

	_, err := js.Publish(context.Background(), "testSubject", []byte("test data"))
	if err != nil {
		t.Fatal(err)
	}

	// b, err := json.Marshal(errResp)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// if string(b) != string(r.Data) {
	// 	t.Errorf("ErrorEncoder: got: %q, expected: %q", r.Data, b)
	// }
}

type noContentResponse struct{}

func TestEncodeNoContent(t *testing.T) {
	handler := natstransport.NewSubscriber(
		func(context.Context, interface{}) (interface{}, error) { return noContentResponse{}, nil },
		func(ctx context.Context, m jetstream.Msg) (request interface{}, err error) { return struct{}{}, nil },
		natstransport.EncodeJSONResponse,
	)

	js, stop := newConsumer(t, handler)
	defer stop()

	_, err := js.Publish(context.Background(), "testSubject", []byte("test data"))
	if err != nil {
		t.Fatal(err)
	}

	// if want, have := `{}`, strings.TrimSpace(string(r.Data)); want != have {
	// 	t.Errorf("Body: want %s, have %s", want, have)
	// }
}

func TestNoOpRequestDecoder(t *testing.T) {
	handler := natstransport.NewSubscriber(
		func(ctx context.Context, request interface{}) (interface{}, error) {
			if request != nil {
				t.Error("Expected nil request in endpoint when using NopRequestDecoder")
			}
			return nil, nil
		},
		natstransport.NopRequestDecoder,
		natstransport.EncodeJSONResponse,
	)

	js, stop := newConsumer(t, handler)
	defer stop()

	_, err := js.Publish(context.Background(), "testSubject", []byte("test data"))
	if err != nil {
		t.Fatal(err)
	}

	// if want, have := `null`, strings.TrimSpace(string(r.Data)); want != have {
	// 	t.Errorf("Body: want %s, have %s", want, have)
	// }
}

func testSubscriber(t *testing.T) (step func(), resp <-chan *nats.Msg) {
	var (
		stepch   = make(chan bool)
		endpoint = func(context.Context, interface{}) (interface{}, error) {
			<-stepch
			return struct{}{}, nil
		}
		response = make(chan *nats.Msg)
		handler  = natstransport.NewSubscriber(
			endpoint,
			func(ctx context.Context, m jetstream.Msg) (request interface{}, err error) { return struct{}{}, nil },
			natstransport.EncodeJSONResponse,
			natstransport.SubscriberBefore(func(ctx context.Context, m jetstream.Msg) context.Context { return ctx }),
			natstransport.SubscriberAfter(func(ctx context.Context, js jetstream.JetStream) context.Context { return ctx }),
		)
	)

	go func() {
		js, stop := newConsumer(t, handler)
		defer stop()

		_, err := js.Publish(context.Background(), "testSubject", []byte("test data"))
		if err != nil {
			t.Fatal(err)
		}

		// response <- r
	}()

	return func() { stepch <- true }, response
}

func testRequest(t *testing.T, c *nats.Conn, handler *natstransport.Subscriber) TestResponse {
	js, stop := newConsumer(t, handler)
	defer stop()

	_, err := js.Publish(context.Background(), "testSubject", []byte("test data"))
	if err != nil {
		t.Fatal(err)
	}

	// var resp TestResponse
	// err = json.Unmarshal(r.Data, &resp)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// return resp

	return TestResponse{}
}
