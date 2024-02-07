package jetstream

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/nats-io/nats.go/jetstream"
)

// Subscriber wraps an endpoint and provides nats.MsgHandler.
type Subscriber[Req, Res any] struct {
	e            Endpoint[Req, Res]
	dec          DecodeRequestFunc[Req]
	enc          EncodeResponseFunc[Res]
	before       []SubscriberRequestFunc
	after        []SubscriberResponseFunc
	errorEncoder ErrorEncoder
	finalizer    []SubscriberFinalizerFunc[Res]
	errorHandler ErrorHandler
}

// NewSubscriber constructs a new subscriber, which provides nats.MsgHandler and wraps
// the provided endpoint.
func NewSubscriber[Req, Res any](
	e Endpoint[Req, Res],
	dec DecodeRequestFunc[Req],
	enc EncodeResponseFunc[Res],
	options ...SubscriberOption[Req, Res],
) *Subscriber[Req, Res] {
	s := &Subscriber[Req, Res]{
		e:            e,
		dec:          dec,
		enc:          enc,
		errorEncoder: DefaultErrorEncoder,
		errorHandler: LogErrorHandler(nil),
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
func SubscriberBefore[Req, Res any](before ...SubscriberRequestFunc) SubscriberOption[Req, Res] {
	return func(s *Subscriber[Req, Res]) { s.before = append(s.before, before...) }
}

// SubscriberAfter functions are executed on the subscriber reply after the
// endpoint is invoked, but before anything is published to the reply.
func SubscriberAfter[Req, Res any](after ...SubscriberResponseFunc) SubscriberOption[Req, Res] {
	return func(s *Subscriber[Req, Res]) { s.after = append(s.after, after...) }
}

// SubscriberErrorEncoder is used to encode errors to the subscriber reply
// whenever they're encountered in the processing of a request. Clients can
// use this to provide custom error formatting. By default,
// errors will be published with the DefaultErrorEncoder.
func SubscriberErrorEncoder[Req, Res any](encoder ErrorEncoder) SubscriberOption[Req, Res] {
	return func(s *Subscriber[Req, Res]) { s.errorEncoder = encoder }
}

// SubscriberErrorLogger is used to log non-terminal errors. By default, no errors
// are logged. This is intended as a diagnostic measure. Finer-grained control
// of error handling, including logging in more detail, should be performed in a
// custom SubscriberErrorEncoder which has access to the context.
// Deprecated: Use SubscriberErrorHandler instead.
func SubscriberErrorLogger[Req, Res any](logger LogFunc) SubscriberOption[Req, Res] {
	return func(s *Subscriber[Req, Res]) { s.errorHandler = LogErrorHandler(logger) }
}

// SubscriberErrorHandler is used to handle non-terminal errors. By default, non-terminal errors
// are ignored. This is intended as a diagnostic measure. Finer-grained control
// of error handling, including logging in more detail, should be performed in a
// custom SubscriberErrorEncoder which has access to the context.
func SubscriberErrorHandler[Req, Res any](errorHandler ErrorHandler) SubscriberOption[Req, Res] {
	return func(s *Subscriber[Req, Res]) { s.errorHandler = errorHandler }
}

// SubscriberFinalizer is executed at the end of every request from a publisher through NATS.
// By default, no finalizer is registered.
func SubscriberFinalizer[Req, Res any](finalizerFunc ...SubscriberFinalizerFunc[Res]) SubscriberOption[Req, Res] {
	return func(s *Subscriber[Req, Res]) { s.finalizer = finalizerFunc }
}

// ServeMsg provides nats.MsgHandler.
func (s Subscriber[Req, Res]) HandleMessage(js jetstream.JetStream) func(jetstream.Msg) {
	return func(msg jetstream.Msg) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var response Res
		if len(s.finalizer) > 0 {
			defer func() {
				for _, f := range s.finalizer {
					f(ctx, msg, response)
				}
			}()
		}

		for _, f := range s.before {
			ctx = f(ctx, msg)
		}

		request, err := s.dec(ctx, msg)
		if err != nil {
			s.errorHandler.Handle(ctx, err)
			if msg.Reply() != "" {
				s.errorEncoder(ctx, err, msg.Reply(), js)
			}

			return
		}

		response, err = s.e(ctx, request)
		if err != nil {
			s.errorHandler.Handle(ctx, err)
			if msg.Reply() != "" {
				s.errorEncoder(ctx, err, msg.Reply(), js)
			}

			return
		}

		for _, f := range s.after {
			ctx = f(ctx, js)
		}

		if msg.Reply() == "" {
			return
		}

		if err := s.enc(ctx, msg.Reply(), js, response); err != nil {
			s.errorHandler.Handle(ctx, err)
			s.errorEncoder(ctx, err, msg.Reply(), js)

			return
		}
	}
}

// ErrorEncoder is responsible for encoding an error to the subscriber reply.
// Users are encouraged to use custom ErrorEncoders to encode errors to
// their replies, and will likely want to pass and check for their own error
// types.
type ErrorEncoder func(ctx context.Context, err error, replySubject string, js jetstream.JetStream)

// SubscriberFinalizerFunc can be used to perform work at the end of an request
// from a publisher, after the response has been written to the publisher. The principal
// intended use is for request logging.
type SubscriberFinalizerFunc[Res any] func(ctx context.Context, msg jetstream.Msg, response Res)

// NopRequestDecoder is a DecodeRequestFunc that can be used for requests that do not
// need to be decoded, and simply returns nil, nil.
func NopRequestDecoder[Req any](context.Context, jetstream.Msg) (Req, error) {
	var req Req
	return req, nil
}

// NopResponseEncoder is a EncodeResponseFunc that can be used for responses that do not
// need to be decoded, and simply returns nil.
func NopResponseEncoder[Res any](context.Context, string, jetstream.JetStream, Res) error {
	return nil
}

// EncodeJSONResponse is a EncodeResponseFunc that serializes the response as a
// JSON object to the subscriber reply. Many JSON-over services can use it as
// a sensible default.
func EncodeJSONResponse(ctx context.Context, replySubject string, js jetstream.JetStream, response interface{}) error {
	b, err := json.Marshal(response)
	if err != nil {
		return err
	}

	_, err = js.Publish(ctx, replySubject, b)

	return err
}

// DefaultErrorEncoder writes the error to the subscriber reply.
func DefaultErrorEncoder(ctx context.Context, err error, replySubject string, js jetstream.JetStream) {
	type Response struct {
		Error string `json:"err"`
	}

	response := Response{Error: err.Error()}

	b, err := json.Marshal(response)
	if err != nil {
		return
	}

	js.Publish(ctx, replySubject, b)
}

// ErrorHandler receives a transport error to be processed for diagnostic purposes.
// Usually this means logging the error.
type ErrorHandler interface {
	Handle(ctx context.Context, err error)
}

// The ErrorHandlerFunc type is an adapter to allow the use of
// ordinary function as ErrorHandler. If f is a function
// with the appropriate signature, ErrorHandlerFunc(f) is a
// ErrorHandler that calls f.
type ErrorHandlerFunc func(ctx context.Context, err error)

// Handle calls f(ctx, err).
func (f ErrorHandlerFunc) Handle(ctx context.Context, err error) {
	f(ctx, err)
}

type LogFunc func(context.Context, error)

var LogErrorHandler = func(logFunc LogFunc) ErrorHandlerFunc {
	return func(ctx context.Context, err error) {
		if logFunc == nil {
			slog.ErrorContext(ctx, err.Error())
		} else {
			logFunc(ctx, err)
		}
	}
}
