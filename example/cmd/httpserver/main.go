package main

import (
	"log/slog"
	"net/http"

	"github.com/bobobox-id/gkit/example/internal/audit"
	"github.com/bobobox-id/gkit/example/internal/repository"
	"github.com/bobobox-id/gkit/example/internal/transport"
	"github.com/go-chi/chi/v5"
)

func main() {
	eventRepo := repository.NewEventRepositroy()
	eventSvc := audit.NewEventService(eventRepo)

	r := chi.NewRouter()
	r.Route("/api/v1/events", transport.EventRoutes(eventSvc))

	slog.Info("starting HTTP server...")
	if err := http.ListenAndServe(":3000", r); err != nil {
		slog.Error("unable to start HTTP server")
	}
}
