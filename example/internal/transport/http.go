package transport

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	gkit "github.com/kikihakiem/gkit/core"
	"github.com/kikihakiem/gkit/example/internal/audit"
	httptransport "github.com/kikihakiem/gkit/transport/http"
)

func createEventHTTPHandler(eventSvc *audit.EventService) http.Handler {
	return httptransport.NewServer(
		eventSvc.CreateEvent,
		httptransport.DecodeJSONRequest,
		httptransport.EncodeJSONResponse,
	)
}

func getEventsHTTPHandler(eventSvc *audit.EventService) http.Handler {
	return httptransport.NewServer(
		eventSvc.GetList,
		gkit.NopEncoderDecoder,
		httptransport.EncodeJSONResponse,
	)
}

func EventRoutes(eventSvc *audit.EventService) func(r chi.Router) {
	return func(r chi.Router) {
		r.Method(http.MethodPost, "/", createEventHTTPHandler(eventSvc))
		r.Method(http.MethodGet, "/", getEventsHTTPHandler(eventSvc))
	}
}
