package storage

import "sync"

type Cache struct {
	mu        sync.RWMutex
	campaigns []CampaignRow
}

func NewCache() *Cache {
	return &Cache{}
}

func (c *Cache) GetCampaigns() []CampaignRow {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]CampaignRow(nil), c.campaigns...)
}

func (c *Cache) UpdateCampaigns(campaigns []CampaignRow) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.campaigns = campaigns
}