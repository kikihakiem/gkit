package transport

import (
	"net/http"

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

// RegisterEventRoutes registers the event handlers on mux.
func RegisterEventRoutes(mux *http.ServeMux, eventSvc *audit.EventService) {
	mux.Handle("POST /api/v1/events", createEventHTTPHandler(eventSvc))
	mux.Handle("GET /api/v1/events", getEventsHTTPHandler(eventSvc))
}
