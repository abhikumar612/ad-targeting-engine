package engine

import (
	"context"
	"slices"
	"strings"

	"ad-targeting-engine/internal/cache"
	"ad-targeting-engine/internal/storage"
)

// Indexes for fast candidate narrowing
type indexes struct {
	Campaigns []CampaignWithRules // backing array; indexes reference this

	IncApp     map[string][]int
	ExcApp     map[string][]int
	IncOS      map[string][]int
	ExcOS      map[string][]int
	IncCountry map[string][]int
	ExcCountry map[string][]int

	AgnosticApp     []int
	AgnosticOS      []int
	AgnosticCountry []int
}

type snapshot struct { idx indexes }

// DeliveryEngine exposes read-only, lock-free match operations.
type DeliveryEngine struct { snap cache.Snapshot[snapshot] }

func NewEngine() *DeliveryEngine { return &DeliveryEngine{} }

// BuildSnapshot loads active campaigns+rules and builds inverted indexes.
func (e *DeliveryEngine) BuildSnapshot(ctx context.Context, st *storage.Store) error {
	rows, err := st.LoadActiveCampaigns(ctx)
	if err != nil { return err }
	// Normalize & build rules
	var cs []CampaignWithRules
	for _, r := range rows {
		c := CampaignWithRules{ID: r.ID, Name: r.Name, Image: r.ImageURL, CTA: r.CTA, Status: r.Status}
		for _, rr := range r.Rules {
			vals := make([]string, len(rr.Values))
			for i, v := range rr.Values {
				switch rr.Dimension {
				case "Country": vals[i] = strings.ToUpper(strings.TrimSpace(v))
				default: vals[i] = strings.ToLower(strings.TrimSpace(v)) // AppID, OS
				}
			}
			c.Rules = append(c.Rules, Rule{Dimension: rr.Dimension, IsInclusion: rr.IsInclusion, Values: vals})
		}
		cs = append(cs, c)
	}

	idx := buildIndexes(cs)
	e.snap.Store(snapshot{idx: idx})
	return nil
}

func buildIndexes(cs []CampaignWithRules) indexes {
	ix := indexes{
		Campaigns:       cs,
		IncApp:          map[string][]int{},
		ExcApp:          map[string][]int{},
		IncOS:           map[string][]int{},
		ExcOS:           map[string][]int{},
		IncCountry:      map[string][]int{},
		ExcCountry:      map[string][]int{},
		AgnosticApp:     []int{},
		AgnosticOS:      []int{},
		AgnosticCountry: []int{},
	}
	for i, c := range cs {
		var hasApp, hasOS, hasCountry bool
		for _, r := range c.Rules {
			switch r.Dimension {
			case "AppID":
				hasApp = true
				for _, v := range r.Values {
					m := ix.IncApp
					if !r.IsInclusion { m = ix.ExcApp }
					m[v] = append(m[v], i)
				}
			case "OS":
				hasOS = true
				for _, v := range r.Values {
					m := ix.IncOS
					if !r.IsInclusion { m = ix.ExcOS }
					m[v] = append(m[v], i)
				}
			case "Country":
				hasCountry = true
				for _, v := range r.Values {
					m := ix.IncCountry
					if !r.IsInclusion { m = ix.ExcCountry }
					m[v] = append(m[v], i)
				}
			}
		}
		if !hasApp { ix.AgnosticApp = append(ix.AgnosticApp, i) }
		if !hasOS { ix.AgnosticOS = append(ix.AgnosticOS, i) }
		if !hasCountry { ix.AgnosticCountry = append(ix.AgnosticCountry, i) }
	}
	return ix
}

// Match returns API campaigns for the given request.
func (e *DeliveryEngine) Match(_ context.Context, req MatchRequest) []Campaign {
	// load snapshot atomically
	s, _ := e.snap.Load()
	ix := s.idx

	// candidates set (use int indexes)
	cand := set{}
	// app candidates: inclusion for app + agnostic
	cand.addAll(ix.IncApp[req.AppID])
	cand.addAll(ix.AgnosticApp)
	// os inclusion + agnostic
	cand = cand.intersect(newSet(ix.IncOS[req.OS], ix.AgnosticOS))
	// country inclusion + agnostic
	cand = cand.intersect(newSet(ix.IncCountry[req.Country], ix.AgnosticCountry))
	// Exclusions
	cand = cand.subtract(ix.ExcApp[req.AppID])
	cand = cand.subtract(ix.ExcOS[req.OS])
	cand = cand.subtract(ix.ExcCountry[req.Country])

	// Final verification (cheap; limited set) and build response
	var out []Campaign
	for _, i := range cand.list() {
		c := ix.Campaigns[i]
		if c.Status != "ACTIVE" { continue }
		if matchesAll(c.Rules, req) {
			out = append(out, Campaign{ID: c.ID, Image: c.Image, CTA: c.CTA})
		}
	}
	// deterministic order (optional)
	slices.SortFunc(out, func(a, b Campaign) int { return strings.Compare(a.ID, b.ID) })
	return out
}

func matchesAll(rules []Rule, req MatchRequest) bool {
	for _, r := range rules {
		switch r.Dimension {
		case "AppID":
			if !applyRule(r, req.AppID, false) { return false }
		case "OS":
			if !applyRule(r, req.OS, false) { return false }
		case "Country":
			if !applyRule(r, req.Country, true) { return false }
		}
	}
	return true
}

func applyRule(r Rule, val string, upper bool) bool {
	if len(r.Values) == 0 { return true }
	if upper { val = strings.ToUpper(val) } else { val = strings.ToLower(val) }
	found := false
	for _, v := range r.Values {
		if v == val { found = true; break }
	}
	if r.IsInclusion { return found }
	return !found
}

// --- small set helper ---

type set map[int]struct{}

func newSet(slices ...[]int) set {
	s := set{}
	for _, sl := range slices {
		for _, v := range sl { s[v] = struct{}{} }
	}
	return s
}
func (s set) addAll(sl []int) { for _, v := range sl { s[v] = struct{}{} } }
func (s set) intersect(other set) set {
	if len(s) == 0 { return other }
	if len(other) == 0 { return s }
	res := set{}
	for k := range s { if _, ok := other[k]; ok { res[k] = struct{}{} } }
	return res
}
func (s set) subtract(sl []int) set {
	for _, v := range sl { delete(s, v) }
	return s
}
func (s set) list() []int { out := make([]int, 0, len(s)); for k := range s { out = append(out, k) }; return out }