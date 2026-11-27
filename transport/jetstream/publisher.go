package jetstream

import (
	"context"
	"encoding/json"
	"time"

	gkit "github.com/kikihakiem/gkit/core"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Publisher wraps a URL and provides a method that implements endpoint.Endpoint.
type Publisher[Req, Res any] struct {
	publisher jetstream.JetStream
	enc       gkit.EncodeDecodeFunc[Req, *nats.Msg]
	dec       gkit.EncodeDecodeFunc[*jetstream.PubAck, Res]
	publisherOptions
}

// publisherOptions holds the settings that do not depend on the request or
// response type, so that PublisherOption does not need type parameters.
type publisherOptions struct {
	before  []gkit.BeforeRequestFunc[*nats.Msg]
	after   []gkit.AfterResponseFunc[*jetstream.PubAck]
	timeout time.Duration
	subject string
}

// PublisherOption sets an optional parameter for publishers. The same option
// value works for publishers of any request and response type.
type PublisherOption func(*publisherOptions)

// NewPublisher constructs a usable Publisher for a single remote method.
func NewPublisher[Req, Res any](
	publisher jetstream.JetStream,
	enc gkit.EncodeDecodeFunc[Req, *nats.Msg],
	dec gkit.EncodeDecodeFunc[*jetstream.PubAck, Res],
	options ...PublisherOption,
) *Publisher[Req, Res] {
	p := &Publisher[Req, Res]{
		publisher:        publisher,
		enc:              enc,
		dec:              dec,
		publisherOptions: publisherOptions{timeout: 10 * time.Second},
	}

	for _, option := range options {
		option(&p.publisherOptions)
	}

	return p
}

// PublisherBefore sets the PublisherRequestFuncs that are applied to the outgoing NATS
// request before it's invoked.
func PublisherBefore(before ...gkit.BeforeRequestFunc[*nats.Msg]) PublisherOption {
	return func(p *publisherOptions) { p.before = append(p.before, before...) }
}

// PublisherAfter sets the ClientResponseFuncs applied to the incoming NATS
// request prior to it being decoded. This is useful for obtaining anything off
// of the response and adding onto the context prior to decoding.
func PublisherAfter(after ...gkit.AfterResponseFunc[*jetstream.PubAck]) PublisherOption {
	return func(p *publisherOptions) { p.after = append(p.after, after...) }
}

// PublisherTimeout sets the available timeout for NATS request.
func PublisherTimeout(timeout time.Duration) PublisherOption {
	return func(p *publisherOptions) { p.timeout = timeout }
}

// PublisherSubject sets the subject of the published message. PublisherBefore
// functions run after this value is applied, so they can replace it.
func PublisherSubject(subject string) PublisherOption {
	return func(p *publisherOptions) { p.subject = subject }
}

// Endpoint returns a usable endpoint that invokes the remote endpoint.
func (p Publisher[Req, Res]) Endpoint() gkit.Endpoint[Req, Res] {
	return func(ctx context.Context, request Req) (Res, error) {
		ctx, cancel := context.WithTimeout(ctx, p.timeout)
		defer cancel()

		var response Res

		msg, err := p.enc(ctx, request)
		if err != nil {
			return response, err
		}

		if p.subject != "" {
			msg.Subject = p.subject
		}

		for _, f := range p.before {
			ctx = f(ctx, msg)
		}

		resp, err := p.publisher.PublishMsg(ctx, msg)
		if err != nil {
			return response, err
		}

		for _, f := range p.after {
			ctx = f(ctx, resp, err)
		}

		response, err = p.dec(ctx, resp)
		if err != nil {
			return response, err
		}

		return response, nil
	}
}

// EncodeJSONRequest is an EncodeRequestFunc that serializes the request as a
// JSON object to the Data of the Msg. The message has no subject; set one with
// PublisherSubject or in a PublisherBefore function.
func EncodeJSONRequest[In any](_ context.Context, request In) (*nats.Msg, error) {
	b, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	msg := nats.NewMsg("")
	msg.Data = b

	return msg, nil
}
