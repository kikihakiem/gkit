package jetstream

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	gkit "github.com/kikihakiem/gkit/core"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// ErrNoResponseSubject is returned by EncodeJSONResponse when the context has
// no response subject.
var ErrNoResponseSubject = errors.New("jetstream: no response subject")

type contextKey int

// ContextKeyResponseSubject holds the subject set by SubscriberResponseSubject.
const ContextKeyResponseSubject contextKey = iota

// Subscriber wraps an endpoint and provides nats.MsgHandler.
type Subscriber[Req, Res any] struct {
	e               gkit.Endpoint[Req, Res]
	dec             gkit.EncodeDecodeFunc[jetstream.Msg, Req]
	enc             gkit.ResponseEncoder[jetstream.JetStream, Res]
	before          []gkit.BeforeRequestFunc[jetstream.Msg]
	after           []gkit.AfterResponseFunc[Res]
	errorEncoder    gkit.ErrorEncoder[jetstream.JetStream]
	finalizer       []gkit.FinalizerFunc[jetstream.Msg]
	errorHandler    gkit.ErrorHandler
	responseSubject string
}

// NewSubscriber constructs a new subscriber, which provides nats.MsgHandler and wraps
// the provided endpoint.
func NewSubscriber[Req, Res any](
	e gkit.Endpoint[Req, Res],
	dec gkit.EncodeDecodeFunc[jetstream.Msg, Req],
	enc gkit.ResponseEncoder[jetstream.JetStream, Res],
	options ...gkit.Option[*Subscriber[Req, Res]],
) *Subscriber[Req, Res] {
	s := &Subscriber[Req, Res]{
		e:            e,
		dec:          dec,
		enc:          enc,
		errorEncoder: EncodeJSONError,
		errorHandler: gkit.LogErrorHandler(nil),
	}

	for _, option := range options {
		option(s)
	}

	return s
}

// NewMessageHandler constructs a new jetstream.MessageHandler and wraps the provided endpoint.
func NewMessageHandler[Req, Res any](
	js jetstream.JetStream,
	e gkit.Endpoint[Req, Res],
	dec gkit.EncodeDecodeFunc[jetstream.Msg, Req],
	enc gkit.ResponseEncoder[jetstream.JetStream, Res],
	options ...gkit.Option[*Subscriber[Req, Res]],
) jetstream.MessageHandler {
	return NewSubscriber(e, dec, enc, options...).HandleMessage(js)
}

// SubscriberBefore functions are executed on the publisher request object before the
// request is decoded.
func SubscriberBefore[Req, Res any](before ...gkit.BeforeRequestFunc[jetstream.Msg]) gkit.Option[*Subscriber[Req, Res]] {
	return func(s *Subscriber[Req, Res]) { s.before = append(s.before, before...) }
}

// SubscriberAfter functions are executed on the subscriber reply after the
// endpoint is invoked, but before anything is published to the reply.
func SubscriberAfter[Req, Res any](after ...gkit.AfterResponseFunc[Res]) gkit.Option[*Subscriber[Req, Res]] {
	return func(s *Subscriber[Req, Res]) { s.after = append(s.after, after...) }
}

// SubscriberErrorEncoder is used to encode errors whenever they're encountered
// in the processing of a request. Clients can use this to provide custom error
// formatting. By default, errors are published with EncodeJSONError to the
// subject set by SubscriberResponseSubject.
func SubscriberErrorEncoder[Req, Res any](encoder gkit.ErrorEncoder[jetstream.JetStream]) gkit.Option[*Subscriber[Req, Res]] {
	return func(s *Subscriber[Req, Res]) { s.errorEncoder = encoder }
}

// SubscriberErrorLogger is used to log non-terminal errors. By default, no errors
// are logged. This is intended as a diagnostic measure. Finer-grained control
// of error handling, including logging in more detail, should be performed in a
// custom SubscriberErrorEncoder which has access to the context.
// Deprecated: Use SubscriberErrorHandler instead.
func SubscriberErrorLogger[Req, Res any](logger gkit.LogFunc) gkit.Option[*Subscriber[Req, Res]] {
	return func(s *Subscriber[Req, Res]) { s.errorHandler = gkit.LogErrorHandler(logger) }
}

// SubscriberErrorHandler is used to handle non-terminal errors. By default, non-terminal errors
// are ignored. This is intended as a diagnostic measure. Finer-grained control
// of error handling, including logging in more detail, should be performed in a
// custom SubscriberErrorEncoder which has access to the context.
func SubscriberErrorHandler[Req, Res any](errorHandler gkit.ErrorHandler) gkit.Option[*Subscriber[Req, Res]] {
	return func(s *Subscriber[Req, Res]) { s.errorHandler = errorHandler }
}

// SubscriberFinalizer is executed at the end of every request from a publisher through NATS.
// By default, no finalizer is registered.
func SubscriberFinalizer[Req, Res any](finalizerFunc ...gkit.FinalizerFunc[jetstream.Msg]) gkit.Option[*Subscriber[Req, Res]] {
	return func(s *Subscriber[Req, Res]) { s.finalizer = append(s.finalizer, finalizerFunc...) }
}

// SubscriberResponseSubject sets the subject used by EncodeJSONResponse and
// EncodeJSONError. Without it, EncodeJSONResponse returns ErrNoResponseSubject
// and EncodeJSONError publishes nothing.
func SubscriberResponseSubject[Req, Res any](subject string) gkit.Option[*Subscriber[Req, Res]] {
	return func(s *Subscriber[Req, Res]) { s.responseSubject = subject }
}

// ServeMsg provides nats.MsgHandler.
func (s Subscriber[Req, Res]) HandleMessage(js jetstream.JetStream) func(jetstream.Msg) {
	return func(msg jetstream.Msg) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ctx = context.WithValue(ctx, ContextKeyResponseSubject, s.responseSubject)

		var (
			response Res
			err      error
		)

		defer func() {
			for _, f := range s.finalizer {
				f(ctx, msg, err)
			}

			if err != nil {
				msg.Nak() //nolint:errcheck
			} else {
				msg.Ack() //nolint:errcheck
			}
		}()

		for _, f := range s.before {
			ctx = f(ctx, msg)
		}

		request, err := s.dec(ctx, msg)
		if err != nil {
			s.errorHandler.Handle(ctx, err)
			s.errorEncoder(ctx, js, err)

			return
		}

		response, err = s.e(ctx, request)
		if err != nil {
			s.errorHandler.Handle(ctx, err)
			s.errorEncoder(ctx, js, err)

			return
		}

		for _, f := range s.after {
			ctx = f(ctx, response)
		}

		err = s.enc(ctx, js, response)
		if err != nil {
			s.errorHandler.Handle(ctx, err)
			s.errorEncoder(ctx, js, err)

			return
		}
	}
}

// DecodeJSONRequest is a DecodeRequestFunc that deserialize JSON to domain object.
func DecodeJSONRequest[Req any](_ context.Context, msg jetstream.Msg) (Req, error) {
	var req Req

	err := json.NewDecoder(bytes.NewReader(msg.Data())).Decode(&req)
	if err != nil {
		return req, err
	}

	return req, nil
}

// EncodeJSONResponse is a EncodeResponseFunc that serializes the response as a
// JSON object and publishes it to the subject set by SubscriberResponseSubject.
func EncodeJSONResponse[Res any](ctx context.Context, js jetstream.JetStream, response Res) error {
	subject, _ := ctx.Value(ContextKeyResponseSubject).(string)
	if subject == "" {
		return ErrNoResponseSubject
	}

	b, err := json.Marshal(response)
	if err != nil {
		return err
	}

	msg := nats.NewMsg(subject)
	msg.Data = b

	_, err = js.PublishMsg(ctx, msg)

	return err
}

type ErrResponse struct {
	Error string `json:"err"`
}

// EncodeJSONError writes the error to the subject set by
// SubscriberResponseSubject. It publishes nothing when the subject is empty.
func EncodeJSONError(ctx context.Context, js jetstream.JetStream, err error) {
	subject, _ := ctx.Value(ContextKeyResponseSubject).(string)
	if subject == "" {
		return
	}

	response := ErrResponse{Error: err.Error()}

	b, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		return
	}

	msg := nats.NewMsg(subject)
	msg.Data = b

	js.PublishMsg(ctx, msg) //nolint:errcheck
}
