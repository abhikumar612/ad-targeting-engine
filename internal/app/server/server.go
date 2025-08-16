package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ad-targeting-engine/internal/storage"
)

type Server struct {
	store *storage.Store
	cache *storage.Cache
}

func New(store *storage.Store, cache *storage.Cache) *Server {
	return &Server{store: store, cache: cache}
}

func (s *Server) Start(addr string) {
	http.HandleFunc("/v1/delivery", s.handleDelivery)
	fmt.Println("http server starting on " + addr)
	http.ListenAndServe(addr, nil)
}

// cache refresher loop
func (s *Server) StartCacheRefresher(ctx context.Context) {
	go func() {
		for {
			campaigns, err := s.store.LoadActiveCampaigns(ctx)
			if err != nil {
				fmt.Println("error refreshing campaigns:", err)
				time.Sleep(10 * time.Second)
				continue
			}
			s.cache.UpdateCampaigns(campaigns)
			time.Sleep(30 * time.Second)
		}
	}()
}

// delivery handler
func (s *Server) handleDelivery(w http.ResponseWriter, r *http.Request) {
	app := r.URL.Query().Get("app")
	country := r.URL.Query().Get("country")
	os := r.URL.Query().Get("os")

	// required params check
	if app == "" || country == "" || os == "" {
		http.Error(w, `{"error":"missing required param"}`, http.StatusBadRequest)
		return
	}

	req := map[string]string{
		"appid":   strings.ToLower(r.URL.Query().Get("app")),
		"country": strings.ToLower(r.URL.Query().Get("country")),
		"os":      strings.ToLower(r.URL.Query().Get("os")),
	}

	campaigns := s.cache.GetCampaigns()
	var matched []storage.CampaignRow
	for _, c := range campaigns {
		if matches(c, req) {
			matched = append(matched, c)
		}
	}

	if len(matched) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matched)
}

func matches(c storage.CampaignRow, req map[string]string) bool {
	for _, r := range c.Rules {
		dim := strings.ToLower(r.Dimension)
		val, ok := req[dim]
		if !ok {
			return false
		}

		inSet := contains(r.Values, val)
		if r.IsInclusion && !inSet {
			return false
		}
		if !r.IsInclusion && inSet {
			return false
		}
	}
	return true
}

func contains(list []string, val string) bool {
	for _, v := range list {
		if v == val {
			return true
		}
	}
	return false
}
