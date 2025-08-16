package cache

import "sync/atomic"

// Snapshot is a lock-free, read-optimized container
// holding any immutable structure.
type Snapshot[T any] struct { v atomic.Value }

// Load returns the stored value. If none is stored yet, the zero value is returned twice.
func (s *Snapshot[T]) Load() (zero T, _ T) {
	v := s.v.Load()
	if v == nil {
		var z T
		return z, z
	}
	return v.(T), v.(T)
}

// Store atomically swaps in the new value.
func (s *Snapshot[T]) Store(v T) {
	s.v.Store(v)
}