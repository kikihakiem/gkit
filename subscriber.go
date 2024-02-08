package jetstream

import (
	"context"
	"encoding/json"

	"github.com/kikihakiem/jetstream-transport/gkit"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Subscriber wraps an endpoint and provides nats.MsgHandler.
type Subscriber[Req, Res any] struct {
	e            gkit.Endpoint[Req, Res]
	dec          gkit.DecodeRequestFunc[jetstream.Msg, Req]
	enc          gkit.EncodeResponseFunc[Res, *nats.Msg]
	before       []gkit.BeforeRequestFunc[Req]
	after        []gkit.AfterResponseFunc[Res]
	errorEncoder gkit.ErrorEncoder[*nats.Msg]
	finalizer    []gkit.FinalizerFunc[jetstream.Msg, *nats.Msg]
	errorHandler gkit.ErrorHandler
}

// NewSubscriber constructs a new subscriber, which provides nats.MsgHandler and wraps
// the provided endpoint.
func NewSubscriber[Req, Res any](
	e gkit.Endpoint[Req, Res],
	dec gkit.DecodeRequestFunc[jetstream.Msg, Req],
	enc gkit.EncodeResponseFunc[Res, *nats.Msg],
	options ...SubscriberOption[Req, Res],
) *Subscriber[Req, Res] {
	s := &Subscriber[Req, Res]{
		e:            e,
		dec:          dec,
		enc:          enc,
		errorEncoder: DefaultErrorEncoder,
		errorHandler: gkit.LogErrorHandler(nil),
	}

	for _, option := range options {
		option(s)
	}

	return s
}

// SubscriberOption sets an optional parameter for subscribers.
type SubscriberOption[Req, Res any] func(*Subscriber[Req, Res])

// SubscriberBefore functions are executed on the publisher request object before the
// request is decoded.
func SubscriberBefore[Req, Res any](before ...gkit.BeforeRequestFunc[Req]) SubscriberOption[Req, Res] {
	return func(s *Subscriber[Req, Res]) { s.before = append(s.before, before...) }
}

// SubscriberAfter functions are executed on the subscriber reply after the
// endpoint is invoked, but before anything is published to the reply.
func SubscriberAfter[Req, Res any](after ...gkit.AfterResponseFunc[Res]) SubscriberOption[Req, Res] {
	return func(s *Subscriber[Req, Res]) { s.after = append(s.after, after...) }
}

// SubscriberErrorEncoder is used to encode errors to the subscriber reply
// whenever they're encountered in the processing of a request. Clients can
// use this to provide custom error formatting. By default,
// errors will be published with the DefaultErrorEncoder.
func SubscriberErrorEncoder[Req, Res any](encoder gkit.ErrorEncoder[*nats.Msg]) SubscriberOption[Req, Res] {
	return func(s *Subscriber[Req, Res]) { s.errorEncoder = encoder }
}

// SubscriberErrorLogger is used to log non-terminal errors. By default, no errors
// are logged. This is intended as a diagnostic measure. Finer-grained control
// of error handling, including logging in more detail, should be performed in a
// custom SubscriberErrorEncoder which has access to the context.
// Deprecated: Use SubscriberErrorHandler instead.
func SubscriberErrorLogger[Req, Res any](logger gkit.LogFunc) SubscriberOption[Req, Res] {
	return func(s *Subscriber[Req, Res]) { s.errorHandler = gkit.LogErrorHandler(logger) }
}

// SubscriberErrorHandler is used to handle non-terminal errors. By default, non-terminal errors
// are ignored. This is intended as a diagnostic measure. Finer-grained control
// of error handling, including logging in more detail, should be performed in a
// custom SubscriberErrorEncoder which has access to the context.
func SubscriberErrorHandler[Req, Res any](errorHandler gkit.ErrorHandler) SubscriberOption[Req, Res] {
	return func(s *Subscriber[Req, Res]) { s.errorHandler = errorHandler }
}

// SubscriberFinalizer is executed at the end of every request from a publisher through NATS.
// By default, no finalizer is registered.
func SubscriberFinalizer[Req, Res any](finalizerFunc ...gkit.FinalizerFunc[jetstream.Msg, *nats.Msg]) SubscriberOption[Req, Res] {
	return func(s *Subscriber[Req, Res]) { s.finalizer = finalizerFunc }
}

// ServeMsg provides nats.MsgHandler.
func (s Subscriber[Req, Res]) HandleMessage(js jetstream.JetStream) func(jetstream.Msg) {
	return func(msg jetstream.Msg) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var (
			response Res
			err      error
			reply    *nats.Msg
		)

		if len(s.finalizer) > 0 {
			defer func() {
				for _, f := range s.finalizer {
					f(ctx, msg, reply, err)
				}
			}()
		}

		request, err := s.dec(ctx, msg)
		if err != nil {
			s.errorHandler.Handle(ctx, err)
			if msg.Reply() != "" {
				reply = s.errorEncoder(ctx, err)
			}

			return
		}

		for _, f := range s.before {
			ctx = f(ctx, request)
		}

		response, err = s.e(ctx, request)
		if err != nil {
			s.errorHandler.Handle(ctx, err)
			if msg.Reply() != "" {
				reply = s.errorEncoder(ctx, err)
			}

			return
		}

		for _, f := range s.after {
			ctx = f(ctx, response, err)
		}

		if msg.Reply() == "" {
			return
		}

		reply, err = s.enc(ctx, response)
		if err != nil {
			s.errorHandler.Handle(ctx, err)
			reply = s.errorEncoder(ctx, err)

			return
		}
	}
}

// EncodeJSONResponse is a EncodeResponseFunc that serializes the response as a
// JSON object to the subscriber reply. Many JSON-over services can use it as
// a sensible default.
func EncodeJSONResponse[In any](ctx context.Context, response In) (*nats.Msg, error) {
	b, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	msg := nats.NewMsg("foo")
	msg.Data = b

	return msg, nil
}

type ErrResponse struct {
	Error string `json:"err"`
}

// DefaultErrorEncoder writes the error to the subscriber reply.
func DefaultErrorEncoder(ctx context.Context, err error) *nats.Msg {
	response := ErrResponse{Error: err.Error()}

	b, err := json.Marshal(response)
	if err != nil {
		return nil
	}

	msg := nats.NewMsg("foo")
	msg.Data = b

	return msg
}
