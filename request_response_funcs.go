package jetstream

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// SubscriberRequestFunc may take information from a subscriber request and put it into a
// request context. In subscribers, RequestFuncs are executed prior to invoking the
// endpoint.
type SubscriberRequestFunc func(context.Context, jetstream.Msg) context.Context

// PublisherRequestFunc may take information from a publisher request and put it into a
// request context. In publishers, RequestFuncs are executed prior to invoking the
// endpoint.
type PublisherRequestFunc func(context.Context, *nats.Msg) context.Context

// SubscriberResponseFunc may take information from a request context and use it to
// manipulate a Publisher. SubscriberResponseFuncs are only executed in
// subscribers, after invoking the endpoint but prior to publishing a reply.
type SubscriberResponseFunc func(context.Context, jetstream.JetStream) context.Context

// PublisherResponseFunc may take information from an NATS request and make the
// response available for consumption. ClientResponseFuncs are only executed in
// clients, after a request has been made, but prior to it being decoded.
type PublisherResponseFunc func(context.Context, *jetstream.PubAck) context.Context
