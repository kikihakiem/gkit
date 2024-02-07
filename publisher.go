package jetstream

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Publisher wraps a URL and provides a method that implements endpoint.Endpoint.
type Publisher struct {
	publisher jetstream.JetStream
	subject   string
	enc       EncodeRequestFunc
	dec       DecodeResponseFunc
	before    []PublisherRequestFunc
	after     []PublisherResponseFunc
	timeout   time.Duration
}

// NewPublisher constructs a usable Publisher for a single remote method.
func NewPublisher(
	publisher jetstream.JetStream,
	subject string,
	enc EncodeRequestFunc,
	dec DecodeResponseFunc,
	options ...PublisherOption,
) *Publisher {
	p := &Publisher{
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
type PublisherOption func(*Publisher)

// PublisherBefore sets the PublisherRequestFuncs that are applied to the outgoing NATS
// request before it's invoked.
func PublisherBefore(before ...PublisherRequestFunc) PublisherOption {
	return func(p *Publisher) { p.before = append(p.before, before...) }
}

// PublisherAfter sets the ClientResponseFuncs applied to the incoming NATS
// request prior to it being decoded. This is useful for obtaining anything off
// of the response and adding onto the context prior to decoding.
func PublisherAfter(after ...PublisherResponseFunc) PublisherOption {
	return func(p *Publisher) { p.after = append(p.after, after...) }
}

// PublisherTimeout sets the available timeout for NATS request.
func PublisherTimeout(timeout time.Duration) PublisherOption {
	return func(p *Publisher) { p.timeout = timeout }
}

// Endpoint returns a usable endpoint that invokes the remote endpoint.
func (p Publisher) Endpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		ctx, cancel := context.WithTimeout(ctx, p.timeout)
		defer cancel()

		msg := nats.NewMsg(p.subject)

		if err := p.enc(ctx, msg, request); err != nil {
			return nil, err
		}

		for _, f := range p.before {
			ctx = f(ctx, msg)
		}

		resp, err := p.publisher.PublishMsg(ctx, msg)
		if err != nil {
			return nil, err
		}

		for _, f := range p.after {
			ctx = f(ctx, resp)
		}

		response, err := p.dec(ctx, resp)
		if err != nil {
			return nil, err
		}

		return response, nil
	}
}

// EncodeJSONRequest is an EncodeRequestFunc that serializes the request as a
// JSON object to the Data of the Msg. Many JSON-over-NATS services can use it as
// a sensible default.
func EncodeJSONRequest(_ context.Context, msg *nats.Msg, request interface{}) error {
	b, err := json.Marshal(request)
	if err != nil {
		return err
	}

	msg.Data = b

	return nil
}

// NopRequestEncoder is a EncodeRequestFunc that can be used for requests that do not
// need to be decoded, and returns nil.
func NopRequestEncoder(context.Context, *nats.Msg, interface{}) error {
	return nil
}

// NopResponseDecoder is a DecodeResponseFunc that can be used for responses that do not
// need to be decoded, and simply returns ack, nil.
func NopResponseDecoder(_ context.Context, ack *jetstream.PubAck) (interface{}, error) {
	return ack, nil
}
