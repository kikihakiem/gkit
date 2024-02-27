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
	before    []gkit.BeforeRequestFunc[*nats.Msg]
	after     []gkit.AfterResponseFunc[*jetstream.PubAck]
	timeout   time.Duration
}

// NewPublisher constructs a usable Publisher for a single remote method.
func NewPublisher[Req, Res any](
	publisher jetstream.JetStream,
	enc gkit.EncodeDecodeFunc[Req, *nats.Msg],
	dec gkit.EncodeDecodeFunc[*jetstream.PubAck, Res],
	options ...gkit.Option[*Publisher[Req, Res]],
) *Publisher[Req, Res] {
	p := &Publisher[Req, Res]{
		publisher: publisher,
		enc:       enc,
		dec:       dec,
		timeout:   10 * time.Second,
	}

	for _, option := range options {
		option(p)
	}

	return p
}

// PublisherBefore sets the PublisherRequestFuncs that are applied to the outgoing NATS
// request before it's invoked.
func PublisherBefore[Req, Res any](before ...gkit.BeforeRequestFunc[*nats.Msg]) gkit.Option[*Publisher[Req, Res]] {
	return func(p *Publisher[Req, Res]) { p.before = append(p.before, before...) }
}

// PublisherAfter sets the ClientResponseFuncs applied to the incoming NATS
// request prior to it being decoded. This is useful for obtaining anything off
// of the response and adding onto the context prior to decoding.
func PublisherAfter[Req, Res any](after ...gkit.AfterResponseFunc[*jetstream.PubAck]) gkit.Option[*Publisher[Req, Res]] {
	return func(p *Publisher[Req, Res]) { p.after = append(p.after, after...) }
}

// PublisherTimeout sets the available timeout for NATS request.
func PublisherTimeout[Req, Res any](timeout time.Duration) gkit.Option[*Publisher[Req, Res]] {
	return func(p *Publisher[Req, Res]) { p.timeout = timeout }
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
// JSON object to the Data of the Msg. Many JSON-over-NATS services can use it as
// a sensible default.
func EncodeJSONRequest[In any](_ context.Context, request In) (*nats.Msg, error) {
	b, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	msg := nats.NewMsg("jstransport.response")
	msg.Data = b

	return msg, nil
}
