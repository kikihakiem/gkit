package jetstream

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// DecodeRequestFunc extracts a user-domain request object from a publisher
// request object. It's designed to be used in NATS subscribers, for subscriber-side
// endpoints. One straightforward DecodeRequestFunc could be something that
// JSON decodes from the request body to the concrete response type.
type DecodeRequestFunc[Req any] func(ctx context.Context, msg jetstream.Msg) (request Req, err error)

// EncodeRequestFunc encodes the passed request object into the NATS request
// object. It's designed to be used in NATS publishers, for publisher-side
// endpoints. One straightforward EncodeRequestFunc could something that JSON
// encodes the object directly to the request payload.
type EncodeRequestFunc[Req any] func(ctx context.Context, msg *nats.Msg, request Req) error

// EncodeResponseFunc encodes the passed response object to the subscriber reply.
// It's designed to be used in NATS subscribers, for subscriber-side
// endpoints. One straightforward EncodeResponseFunc could be something that
// JSON encodes the object directly to the response body.
type EncodeResponseFunc[Res any] func(ctx context.Context, replySubject string, js jetstream.JetStream, response Res) error

// DecodeResponseFunc extracts a user-domain response object from an NATS
// response object. It's designed to be used in NATS publisher, for publisher-side
// endpoints. One straightforward DecodeResponseFunc could be something that
// JSON decodes from the response payload to the concrete response type.
type DecodeResponseFunc[Res any] func(ctx context.Context, ack *jetstream.PubAck) (response Res, err error)
