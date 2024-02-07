package jetstream

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// SubscriberRequestFunc may take information from a subscriber request and put it into a
// request context. In subscribers, RequestFuncs are executed prior to invoking the
// endpoint.
type SubscriberRequestFunc func(ctx context.Context, msg jetstream.Msg) context.Context

// PublisherRequestFunc may take information from a publisher request and put it into a
// request context. In publishers, RequestFuncs are executed prior to invoking the
// endpoint.
type PublisherRequestFunc func(ctx context.Context, msg *nats.Msg) context.Context

// SubscriberResponseFunc may take information from a request context and use it to
// manipulate a Publisher. SubscriberResponseFuncs are only executed in
// subscribers, after invoking the endpoint but prior to publishing a reply.
type SubscriberResponseFunc func(ctx context.Context, js jetstream.JetStream) context.Context

// PublisherResponseFunc may take information from an NATS request and make the
// response available for consumption. ClientResponseFuncs are only executed in
// clients, after a request has been made, but prior to it being decoded.
type PublisherResponseFunc func(ctx context.Context, ack *jetstream.PubAck) context.Context
