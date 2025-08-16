package engine

import (
	"context"
	"fmt"
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

type snapshot struct{ idx indexes }

// DeliveryEngine exposes read-only, lock-free match operations.
type DeliveryEngine struct{ snap cache.Snapshot[snapshot] }

func NewEngine() *DeliveryEngine { return &DeliveryEngine{} }

// BuildSnapshot loads active campaigns+rules and builds inverted indexes.
func (e *DeliveryEngine) BuildSnapshot(ctx context.Context, st *storage.Store) error {
	rows, err := st.LoadActiveCampaigns(ctx)
	fmt.Printf("Loaded %d campaigns from DB\n", len(rows))
	for _, r := range rows {
		fmt.Printf("Campaign %s (%s), rules=%v\n", r.ID, r.Name, r.Rules)
	}

	if err != nil {
		return err
	}
	// Normalize & build rules
	var cs []CampaignWithRules
	for _, r := range rows {
		c := CampaignWithRules{ID: r.ID, Name: r.Name, Image: r.ImageURL, CTA: r.CTA, Status: r.Status}
		for _, rr := range r.Rules {
			vals := make([]string, len(rr.Values))
			for i, v := range rr.Values {
				switch strings.ToLower(rr.Dimension) {
				case "country":
					vals[i] = strings.ToUpper(strings.TrimSpace(v))
				default: // appid, os
					vals[i] = strings.ToLower(strings.TrimSpace(v))
				}
			}
			c.Rules = append(c.Rules, Rule{
				Dimension:   strings.ToLower(rr.Dimension),
				IsInclusion: rr.IsInclusion,
				Values:      vals,
			})
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
		var hasApp, hasOS, hasCountry, hasIncCountry bool
		for _, r := range c.Rules {
			switch r.Dimension {
			case "appid":
				hasApp = true
				for _, v := range r.Values {
					m := ix.IncApp
					if !r.IsInclusion {
						m = ix.ExcApp
					}
					m[strings.ToLower(v)] = append(m[strings.ToLower(v)], i)
				}
			case "os":
				hasOS = true
				for _, v := range r.Values {
					m := ix.IncOS
					if !r.IsInclusion {
						m = ix.ExcOS
					}
					m[strings.ToLower(v)] = append(m[strings.ToLower(v)], i)
				}
			case "country":
				hasCountry = true
				if r.IsInclusion {
					hasIncCountry = true
				}
				for _, v := range r.Values {
					m := ix.IncCountry
					if !r.IsInclusion {
						m = ix.ExcCountry
					}
					m[strings.ToUpper(v)] = append(m[strings.ToUpper(v)], i)
				}
			}
		}
		if !hasApp {
			ix.AgnosticApp = append(ix.AgnosticApp, i)
		}
		if !hasOS {
			ix.AgnosticOS = append(ix.AgnosticOS, i)
		}
		if hasCountry && !hasIncCountry {
			ix.AgnosticCountry = append(ix.AgnosticCountry, i)
		} else if !hasCountry {
			ix.AgnosticCountry = append(ix.AgnosticCountry, i)
		}
	}
	return ix
}

// Match returns API campaigns for the given request.
func (e *DeliveryEngine) Match(_ context.Context, req MatchRequest) []Campaign {
	// normalize request
	req.AppID = strings.ToLower(strings.TrimSpace(req.AppID))
	req.OS = strings.ToLower(strings.TrimSpace(req.OS))
	req.Country = strings.ToUpper(strings.TrimSpace(req.Country))

	// load snapshot
	s, _ := e.snap.Load()
	ix := s.idx

	// start with ALL campaigns, then narrow down
	cand := newSet(rangeIndices(len(ix.Campaigns))) // start with all campaigns
	fmt.Printf("Initial candidates: %v\n", cand.list())

	// Apply AppID rules
	cand = cand.intersect(newSet(ix.IncApp[req.AppID], ix.AgnosticApp))
	fmt.Printf("After app: %v\n", cand.list())

	// Apply OS rules
	cand = cand.intersect(newSet(ix.IncOS[req.OS], ix.AgnosticOS))
	fmt.Printf("After os: %v\n", cand.list())

	// Apply Country rules
	cand = cand.intersect(newSet(ix.IncCountry[req.Country], ix.AgnosticCountry))
	fmt.Printf("After country: %v\n", cand.list())

	// Apply exclusions
	cand = cand.subtract(ix.ExcApp[req.AppID])
	cand = cand.subtract(ix.ExcOS[req.OS])
	cand = cand.subtract(ix.ExcCountry[req.Country])
	fmt.Printf("After exclusions: %v\n", cand.list())

	// final verification
	var out []Campaign
	for _, i := range cand.list() {
		c := ix.Campaigns[i]
		if c.Status != "ACTIVE" {
			continue
		}
		if matchesAll(c.Rules, req) {
			out = append(out, Campaign{ID: c.ID, Image: c.Image, CTA: c.CTA})
		}
	}

	// deterministic order
	slices.SortFunc(out, func(a, b Campaign) int { return strings.Compare(a.ID, b.ID) })

	fmt.Printf("Final matches: %v\n", out)
	return out
}

func matchesAll(rules []Rule, req MatchRequest) bool {
	for _, r := range rules {
		switch strings.ToLower(r.Dimension) {
		case "appid":
			if !applyRule(r, req.AppID, false) {
				return false
			}
		case "os":
			if !applyRule(r, req.OS, false) {
				return false
			}
		case "country":
			if !applyRule(r, req.Country, true) {
				return false
			}
		}
	}
	return true
}

func rangeIndices(n int) []int {
	out := make([]int, n)
	for i := 0; i < n; i++ {
		out[i] = i
	}
	return out
}

func applyRule(r Rule, val string, upper bool) bool {
	if len(r.Values) == 0 {
		return true
	}
	if upper {
		val = strings.ToUpper(val)
	} else {
		val = strings.ToLower(val)
	}
	found := false
	for _, v := range r.Values {
		if v == val {
			found = true
			break
		}
	}
	if r.IsInclusion {
		return found
	}
	return !found
}

type set map[int]struct{}

func newSet(slices ...[]int) set {
	s := set{}
	for _, sl := range slices {
		for _, v := range sl {
			s[v] = struct{}{}
		}
	}
	return s
}

func (s set) intersect(other set) set {
	if len(s) == 0 {
		return other
	}
	if len(other) == 0 {
		return s
	}
	res := set{}
	for k := range s {
		if _, ok := other[k]; ok {
			res[k] = struct{}{}
		}
	}
	return res
}
func (s set) subtract(sl []int) set {
	for _, v := range sl {
		delete(s, v)
	}
	return s
}
func (s set) list() []int {
	out := make([]int, 0, len(s))
	for k := range s {
		out = append(out, k)
	}
	return out
}

func allIdx(n int) []int {
	out := make([]int, n)
	for i := 0; i < n; i++ {
		out[i] = i
	}
	return out
}
