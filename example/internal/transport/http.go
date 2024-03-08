package transport

import (
	"net/http"

	gkit "github.com/bobobox-id/gkit/core"
	"github.com/bobobox-id/gkit/example/internal/audit"
	httptransport "github.com/bobobox-id/gkit/transport/http"
	"github.com/go-chi/chi/v5"
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
