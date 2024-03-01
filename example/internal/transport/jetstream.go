package transport

import (
	gkit "github.com/kikihakiem/gkit/core"
	"github.com/kikihakiem/gkit/example/internal/audit"
	jstransport "github.com/kikihakiem/gkit/transport/jetstream"
	"github.com/nats-io/nats.go/jetstream"
)

func CreateEventJetstreamHandler(js jetstream.JetStream, svc *audit.EventService) jetstream.MessageHandler {
	return jstransport.NewSubscriber(
		svc.CreateEvent,
		jstransport.DecodeJSONRequest,
		gkit.NopResponseEncoder,
	).HandleMessage(js)
}
