package engine

// API-facing campaign
type Campaign struct {
	ID    string `json:"cid"`
	Image string `json:"img"`
	CTA   string `json:"cta"`
}

// Generic rule for one dimension
// Dimension: "AppID" | "Country" | "OS" (extensible)
type Rule struct {
	Dimension   string
	IsInclusion bool
	Values      []string // canonicalized at snapshot time
}

type CampaignWithRules struct {
	ID     string
	Name   string
	Image  string
	CTA    string
	Status string // "ACTIVE" | "INACTIVE"
	Rules  []Rule
}

type MatchRequest struct {
	AppID   string // lower-cased at handler
	Country string // upper-cased at handler
	OS      string // lower-cased at handler
}