package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"ad-targeting-engine/internal/engine"
)

type DeliveryHandler struct {
	Eng *engine.DeliveryEngine
}

func NewDeliveryHandler(eng *engine.DeliveryEngine) *DeliveryHandler {
	return &DeliveryHandler{Eng: eng}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *DeliveryHandler) Delivery(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	req := engine.MatchRequest{
		AppID:   strings.ToLower(q.Get("app")),
		OS:      strings.ToLower(q.Get("os")),
		Country: strings.ToUpper(q.Get("country")),
	}

	ctx := r.Context()
	campaigns := h.Eng.Match(ctx, req)

	if len(campaigns) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(campaigns)
}