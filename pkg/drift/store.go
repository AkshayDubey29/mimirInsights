package drift

import (
	"sync"
	"time"
)

// BaselineStore manages configuration baselines
type BaselineStore struct {
	baselines map[string]*BaselineConfig
	mutex     sync.RWMutex
}

// NewBaselineStore creates a new baseline store
func NewBaselineStore() *BaselineStore {
	return &BaselineStore{
		baselines: make(map[string]*BaselineConfig),
	}
}

// StoreBaseline stores a baseline configuration
func (s *BaselineStore) StoreBaseline(key string, baseline *BaselineConfig) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.baselines[key] = baseline
}

// GetBaseline retrieves a baseline configuration
func (s *BaselineStore) GetBaseline(key string) *BaselineConfig {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.baselines[key]
}

// GetBaselinesForNamespace retrieves all baselines for a namespace
func (s *BaselineStore) GetBaselinesForNamespace(namespace string) map[string]*BaselineConfig {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make(map[string]*BaselineConfig)
	for key, baseline := range s.baselines {
		if baseline.Namespace == namespace {
			result[key] = baseline
		}
	}
	return result
}

// DeleteBaseline deletes a baseline configuration
func (s *BaselineStore) DeleteBaseline(key string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.baselines, key)
}

// GetAllBaselines returns all baseline configurations
func (s *BaselineStore) GetAllBaselines() map[string]*BaselineConfig {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make(map[string]*BaselineConfig)
	for key, baseline := range s.baselines {
		result[key] = baseline
	}
	return result
}

// CleanupOldBaselines removes baselines older than the specified duration
func (s *BaselineStore) CleanupOldBaselines(maxAge time.Duration) int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	cutoff := time.Now().Add(-maxAge)
	cleaned := 0

	for key, baseline := range s.baselines {
		if baseline.LastModified.Before(cutoff) {
			delete(s.baselines, key)
			cleaned++
		}
	}

	return cleaned
}
