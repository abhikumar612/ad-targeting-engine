package api

import (
	"ad-targeting-engine/internal/observability"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func Router(h *DeliveryHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(observability.Measure)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(2 * time.Second))

	r.Get("/v1/delivery", h.Delivery)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Handle("/metrics", observability.MetricsHandler())
	return r
}