package jetstream

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Publisher wraps a URL and provides a method that implements endpoint.Endpoint.
type Publisher[Req, Res any] struct {
	publisher jetstream.JetStream
	subject   string
	enc       EncodeRequestFunc[Req]
	dec       DecodeResponseFunc[Res]
	before    []PublisherRequestFunc
	after     []PublisherResponseFunc
	timeout   time.Duration
}

// NewPublisher constructs a usable Publisher for a single remote method.
func NewPublisher[Req, Res any](
	publisher jetstream.JetStream,
	subject string,
	enc EncodeRequestFunc[Req],
	dec DecodeResponseFunc[Res],
	options ...PublisherOption[Req, Res],
) *Publisher[Req, Res] {
	p := &Publisher[Req, Res]{
		publisher: publisher,
		subject:   subject,
		enc:       enc,
		dec:       dec,
		timeout:   10 * time.Second,
	}

	for _, option := range options {
		option(p)
	}

	return p
}

// PublisherOption sets an optional parameter for clients.
type PublisherOption[Req, Res any] func(*Publisher[Req, Res])

// PublisherBefore sets the PublisherRequestFuncs that are applied to the outgoing NATS
// request before it's invoked.
func PublisherBefore[Req, Res any](before ...PublisherRequestFunc) PublisherOption[Req, Res] {
	return func(p *Publisher[Req, Res]) { p.before = append(p.before, before...) }
}

// PublisherAfter sets the ClientResponseFuncs applied to the incoming NATS
// request prior to it being decoded. This is useful for obtaining anything off
// of the response and adding onto the context prior to decoding.
func PublisherAfter[Req, Res any](after ...PublisherResponseFunc) PublisherOption[Req, Res] {
	return func(p *Publisher[Req, Res]) { p.after = append(p.after, after...) }
}

// PublisherTimeout sets the available timeout for NATS request.
func PublisherTimeout[Req, Res any](timeout time.Duration) PublisherOption[Req, Res] {
	return func(p *Publisher[Req, Res]) { p.timeout = timeout }
}

// Endpoint returns a usable endpoint that invokes the remote endpoint.
func (p Publisher[Req, Res]) Endpoint() Endpoint[Req, Res] {
	return func(ctx context.Context, request Req) (Res, error) {
		ctx, cancel := context.WithTimeout(ctx, p.timeout)
		defer cancel()

		var (
			msg      = nats.NewMsg(p.subject)
			response Res
		)

		if err := p.enc(ctx, msg, request); err != nil {
			return response, err
		}

		for _, f := range p.before {
			ctx = f(ctx, msg)
		}

		resp, err := p.publisher.PublishMsg(ctx, msg)
		if err != nil {
			return response, err
		}

		for _, f := range p.after {
			ctx = f(ctx, resp)
		}

		response, err = p.dec(ctx, resp)
		if err != nil {
			return response, err
		}

		return response, nil
	}
}

// EncodeJSONRequest is an EncodeRequestFunc that serializes the request as a
// JSON object to the Data of the Msg. Many JSON-over-NATS services can use it as
// a sensible default.
func EncodeJSONRequest[Req any](_ context.Context, msg *nats.Msg, request Req) error {
	b, err := json.Marshal(request)
	if err != nil {
		return err
	}

	msg.Data = b

	return nil
}

// NopRequestEncoder is a EncodeRequestFunc that can be used for requests that do not
// need to be decoded, and returns nil.
func NopRequestEncoder[Req any](context.Context, *nats.Msg, Req) error {
	return nil
}

// NopResponseDecoder is a DecodeResponseFunc that can be used for responses that do not
// need to be decoded, and simply returns ack, nil.
func NopResponseDecoder[Res any](_ context.Context, ack Res) (Res, error) {
	return ack, nil
}
