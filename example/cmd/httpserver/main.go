package main

import (
	"log/slog"
	"net/http"

	"github.com/kikihakiem/gkit/example/internal/audit"
	"github.com/kikihakiem/gkit/example/internal/repository"
	"github.com/kikihakiem/gkit/example/internal/transport"
)

func main() {
	eventRepo := repository.NewEventRepositroy()
	eventSvc := audit.NewEventService(eventRepo)

	mux := http.NewServeMux()
	transport.RegisterEventRoutes(mux, eventSvc)

	slog.Info("starting HTTP server...")
	if err := http.ListenAndServe(":3000", mux); err != nil {
		slog.Error("unable to start HTTP server")
	}
}
