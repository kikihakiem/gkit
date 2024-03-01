package transport

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	gkit "github.com/kikihakiem/gkit/core"
	"github.com/kikihakiem/gkit/example/internal/audit"
	"github.com/kikihakiem/gkit/example/internal/repository"
	httptransport "github.com/kikihakiem/gkit/transport/http"
)

func createEventHTTPHandler(eventRepo *repository.EventRepository) http.Handler {
	svc := audit.NewEventService(eventRepo)

	return httptransport.NewServer(
		svc.CreateEvent,
		httptransport.DecodeJSONRequest,
		httptransport.EncodeJSONResponse,
	)
}

func getEventsHTTPHandler(eventRepo *repository.EventRepository) http.Handler {
	svc := audit.NewEventService(eventRepo)

	return httptransport.NewServer(
		svc.GetList,
		gkit.NopEncoderDecoder,
		httptransport.EncodeJSONResponse,
	)
}

func EventRoutes(eventRepo *repository.EventRepository) func(r chi.Router) {
	return func(r chi.Router) {
		r.Method(http.MethodPost, "/", createEventHTTPHandler(eventRepo))
		r.Method(http.MethodGet, "/", getEventsHTTPHandler(eventRepo))
	}
}
