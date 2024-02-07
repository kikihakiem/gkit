package jetstream

import (
	"context"
	"encoding/json"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/transport"
	"github.com/go-kit/log"

	"github.com/nats-io/nats.go/jetstream"
)

// Subscriber wraps an endpoint and provides nats.MsgHandler.
type Subscriber struct {
	e            endpoint.Endpoint
	dec          DecodeRequestFunc
	enc          EncodeResponseFunc
	before       []SubscriberRequestFunc
	after        []SubscriberResponseFunc
	errorEncoder ErrorEncoder
	finalizer    []SubscriberFinalizerFunc
	errorHandler transport.ErrorHandler
}

// NewSubscriber constructs a new subscriber, which provides nats.MsgHandler and wraps
// the provided endpoint.
func NewSubscriber(
	e endpoint.Endpoint,
	dec DecodeRequestFunc,
	enc EncodeResponseFunc,
	options ...SubscriberOption,
) *Subscriber {
	s := &Subscriber{
		e:            e,
		dec:          dec,
		enc:          enc,
		errorEncoder: DefaultErrorEncoder,
		errorHandler: transport.NewLogErrorHandler(log.NewNopLogger()),
	}

	for _, option := range options {
		option(s)
	}

	return s
}

// SubscriberOption sets an optional parameter for subscribers.
type SubscriberOption func(*Subscriber)

// SubscriberBefore functions are executed on the publisher request object before the
// request is decoded.
func SubscriberBefore(before ...SubscriberRequestFunc) SubscriberOption {
	return func(s *Subscriber) { s.before = append(s.before, before...) }
}

// SubscriberAfter functions are executed on the subscriber reply after the
// endpoint is invoked, but before anything is published to the reply.
func SubscriberAfter(after ...SubscriberResponseFunc) SubscriberOption {
	return func(s *Subscriber) { s.after = append(s.after, after...) }
}

// SubscriberErrorEncoder is used to encode errors to the subscriber reply
// whenever they're encountered in the processing of a request. Clients can
// use this to provide custom error formatting. By default,
// errors will be published with the DefaultErrorEncoder.
func SubscriberErrorEncoder(encoder ErrorEncoder) SubscriberOption {
	return func(s *Subscriber) { s.errorEncoder = encoder }
}

// SubscriberErrorLogger is used to log non-terminal errors. By default, no errors
// are logged. This is intended as a diagnostic measure. Finer-grained control
// of error handling, including logging in more detail, should be performed in a
// custom SubscriberErrorEncoder which has access to the context.
// Deprecated: Use SubscriberErrorHandler instead.
func SubscriberErrorLogger(logger log.Logger) SubscriberOption {
	return func(s *Subscriber) { s.errorHandler = transport.NewLogErrorHandler(logger) }
}

// SubscriberErrorHandler is used to handle non-terminal errors. By default, non-terminal errors
// are ignored. This is intended as a diagnostic measure. Finer-grained control
// of error handling, including logging in more detail, should be performed in a
// custom SubscriberErrorEncoder which has access to the context.
func SubscriberErrorHandler(errorHandler transport.ErrorHandler) SubscriberOption {
	return func(s *Subscriber) { s.errorHandler = errorHandler }
}

// SubscriberFinalizer is executed at the end of every request from a publisher through NATS.
// By default, no finalizer is registered.
func SubscriberFinalizer(finalizerFunc ...SubscriberFinalizerFunc) SubscriberOption {
	return func(s *Subscriber) { s.finalizer = finalizerFunc }
}

// ServeMsg provides nats.MsgHandler.
func (s Subscriber) HandleMessage(js jetstream.JetStream) func(jetstream.Msg) {
	return func(msg jetstream.Msg) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var response any
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
type SubscriberFinalizerFunc func(ctx context.Context, msg jetstream.Msg, response any)

// NopRequestDecoder is a DecodeRequestFunc that can be used for requests that do not
// need to be decoded, and simply returns nil, nil.
func NopRequestDecoder(context.Context, jetstream.Msg) (interface{}, error) {
	return nil, nil
}

// NopResponseEncoder is a EncodeResponseFunc that can be used for responses that do not
// need to be decoded, and simply returns nil.
func NopResponseEncoder(context.Context, string, jetstream.JetStream, interface{}) error {
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
	logger := log.NewNopLogger()

	type Response struct {
		Error string `json:"err"`
	}

	response := Response{Error: err.Error()}

	b, err := json.Marshal(response)
	if err != nil {
		logger.Log("err", err)

		return
	}

	if _, err := js.Publish(ctx, replySubject, b); err != nil {
		logger.Log("err", err)
	}
}
