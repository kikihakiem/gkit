package transport

import (
	gkit "github.com/bobobox-id/gkit/core"
	"github.com/bobobox-id/gkit/example/internal/audit"
	jstransport "github.com/bobobox-id/gkit/transport/jetstream"
	"github.com/nats-io/nats.go/jetstream"
)

func CreateEventJetstreamHandler(js jetstream.JetStream, svc *audit.EventService) jetstream.MessageHandler {
	return jstransport.NewSubscriber(
		svc.CreateEvent,
		jstransport.DecodeJSONRequest,
		gkit.NopResponseEncoder,
	).HandleMessage(js)
}
