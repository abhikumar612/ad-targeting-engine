package storage

import (
	"sync"
	"sync/atomic"
)

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

type Snapshot[T any] struct{ v atomic.Value }


func (s *Snapshot[T]) Load() (zero T, _ T) {
	v := s.v.Load()
	if v == nil {
		var z T
		return z, z
	}
	return v.(T), v.(T)
}

func (s *Snapshot[T]) Store(v T) {
	s.v.Store(v)
}
