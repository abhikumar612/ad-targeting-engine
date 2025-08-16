package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"ad-targeting-engine/internal/storage"
)

type MockStore struct {
	campaigns []storage.CampaignRow
	err       error
}

func (m *MockStore) LoadActiveCampaigns(ctx context.Context) ([]storage.CampaignRow, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.campaigns, nil
}

func TestDelivery_Scenarios(t *testing.T) {
	tests := []struct {
		name       string
		campaigns  []storage.CampaignRow
		url        string
		wantStatus int
		wantIDs    []string
	}{
		{"missing app", nil, "/v1/delivery?country=us&os=android", http.StatusBadRequest, nil},
		{"missing country", nil, "/v1/delivery?app=com.any&os=android", http.StatusBadRequest, nil},
		{"missing os", nil, "/v1/delivery?app=com.any&country=us", http.StatusBadRequest, nil},
		{
			name: "no match",
			campaigns: []storage.CampaignRow{
				{
					ID:       "1",
					Name:     "Test",
					ImageURL: "img",
					CTA:      "Download",
					Status:   "ACTIVE",
					Rules: []storage.RuleRow{
						{Dimension: "country", IsInclusion: true, Values: []string{"us"}},
					},
				},
			},
			url:        "/v1/delivery?app=com.any&country=ca&os=android",
			wantStatus: http.StatusNoContent,
			wantIDs:    nil,
		},
		{
			name: "single match",
			campaigns: []storage.CampaignRow{
				{
					ID:       "1",
					Name:     "Spotify",
					ImageURL: "img1",
					CTA:      "Download",
					Status:   "ACTIVE",
					Rules: []storage.RuleRow{
						{Dimension: "appid", IsInclusion: true, Values: []string{"com.any"}},
						{Dimension: "country", IsInclusion: true, Values: []string{"us"}},
						{Dimension: "os", IsInclusion: true, Values: []string{"android"}},
					},
				},
			},
			url:        "/v1/delivery?app=com.any&country=us&os=android",
			wantStatus: http.StatusOK,
			wantIDs:    []string{"1"},
		},
		{
			name: "multiple matches",
			campaigns: []storage.CampaignRow{
				{
					ID:       "1",
					Name:     "Spotify",
					ImageURL: "img1",
					CTA:      "Download",
					Status:   "ACTIVE",
					Rules: []storage.RuleRow{
						{Dimension: "country", IsInclusion: true, Values: []string{"us"}},
						{Dimension: "os", IsInclusion: true, Values: []string{"android"}},
					},
				},
				{
					ID:       "2",
					Name:     "Duolingo",
					ImageURL: "img2",
					CTA:      "Install",
					Status:   "ACTIVE",
					Rules: []storage.RuleRow{
						{Dimension: "os", IsInclusion: true, Values: []string{"android"}},
						{Dimension: "country", IsInclusion: false, Values: []string{"ca"}}, // exclude Canada
					},
				},
			},
			url:        "/v1/delivery?app=com.any&country=us&os=android",
			wantStatus: http.StatusOK,
			wantIDs:    []string{"1", "2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestServer(tt.campaigns)

			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()
			srv.handleDelivery(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("got status %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var campaigns []storage.CampaignRow
				if err := json.Unmarshal(w.Body.Bytes(), &campaigns); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if len(campaigns) != len(tt.wantIDs) {
					t.Fatalf("expected %d campaigns, got %d", len(tt.wantIDs), len(campaigns))
				}

				for i, id := range tt.wantIDs {
					if campaigns[i].ID != id {
						t.Errorf("expected campaign %s, got %s", id, campaigns[i].ID)
					}
				}
			}
		})
	}
}

func TestServer_Start_HTTPHandler(t *testing.T) {
	tests := []struct {
		name       string
		campaigns  []storage.CampaignRow
		url        string
		wantStatus int
	}{
		{
			name: "match returns 200",
			campaigns: []storage.CampaignRow{
				{
					ID:     "1",
					Name:   "Test",
					Status: "ACTIVE",
					Rules:  []storage.RuleRow{{Dimension: "country", IsInclusion: true, Values: []string{"us"}}},
				},
			},
			url:        "/v1/delivery?app=x&country=us&os=android",
			wantStatus: http.StatusOK,
		},
		{
			name: "no match returns 204",
			campaigns: []storage.CampaignRow{
				{
					ID:     "1",
					Name:   "Test",
					Status: "ACTIVE",
					Rules:  []storage.RuleRow{{Dimension: "country", IsInclusion: true, Values: []string{"us"}}},
				},
			},
			url:        "/v1/delivery?app=x&country=ca&os=android",
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := storage.NewCache()
			cache.UpdateCampaigns(tt.campaigns)
			srv := &Server{store: nil, cache: cache}

			ts := httptest.NewServer(http.HandlerFunc(srv.handleDelivery))
			defer ts.Close()

			resp, err := http.Get(ts.URL + tt.url)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestServer_StartCacheRefresher(t *testing.T) {
	tests := []struct {
		name         string
		mockStore    *MockStore
		wantCampaign int
	}{
		{
			name: "successful refresh updates cache",
			mockStore: &MockStore{
				campaigns: []storage.CampaignRow{
					{ID: "1", Name: "Spotify", Status: "ACTIVE"},
				},
			},
			wantCampaign: 1,
		},
		{
			name: "error refresh leaves cache empty",
			mockStore: &MockStore{
				err: context.DeadlineExceeded,
			},
			wantCampaign: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cache := storage.NewCache()
			srv := New(tt.mockStore, cache)

			srv.StartCacheRefresher(ctx)
			time.Sleep(100 * time.Millisecond)

			got := cache.GetCampaigns()
			if len(got) != tt.wantCampaign {
				t.Fatalf("expected %d campaigns in cache, got %d", tt.wantCampaign, len(got))
			}
		})
	}
}

func newTestServer(campaigns []storage.CampaignRow) *Server {
	store := &storage.Store{}
	cache := storage.NewCache()
	cache.UpdateCampaigns(campaigns)
	return New(store, cache)
}
