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
	app := strings.TrimSpace(q.Get("app"))
	country := strings.ToUpper(strings.TrimSpace(q.Get("country")))
	osName := strings.ToLower(strings.TrimSpace(q.Get("os")))

	if app == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing app param"})
		return
	}
	if country == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing country param"})
		return
	}
	if osName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing os param"})
		return
	}

	req := engine.MatchRequest{AppID: app, Country: country, OS: osName}
	matches := h.Eng.Match(r.Context(), req)
	if len(matches) == 0 {
		w.WriteHeader(http.StatusNoContent) // 204, no body
		return
	}
	writeJSON(w, http.StatusOK, matches)
}