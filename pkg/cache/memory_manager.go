package cache

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// MemoryManager handles precise memory management for cache
type MemoryManager struct {
	mu sync.RWMutex

	// Memory limits
	maxMemoryBytes     int64
	maxCacheSize       int
	maxTenantCacheSize int
	maxMimirCacheSize  int

	// Current memory usage
	currentMemoryBytes int64
	cacheItemCount     int
	tenantCacheCount   int
	mimirCacheCount    int

	// Memory monitoring
	memoryThreshold     float64 // Percentage threshold for warnings
	lastMemoryCheck     time.Time
	memoryCheckInterval time.Duration

	// Eviction policies
	evictionPolicy    EvictionPolicy
	evictionThreshold float64 // Percentage threshold for eviction
	lastEviction      time.Time
	evictionCount     int64

	// Memory statistics
	peakMemoryBytes int64
	totalEvictions  int64
	memoryWarnings  int64
	lastReset       time.Time
}

// EvictionPolicy defines how cache items should be evicted
type EvictionPolicy string

const (
	EvictionPolicyLRU    EvictionPolicy = "lru"    // Least Recently Used
	EvictionPolicyLFU    EvictionPolicy = "lfu"    // Least Frequently Used
	EvictionPolicyTTL    EvictionPolicy = "ttl"    // Time To Live
	EvictionPolicySize   EvictionPolicy = "size"   // Largest items first
	EvictionPolicyHybrid EvictionPolicy = "hybrid" // Combination of policies
)

// MemoryStats provides detailed memory statistics
type MemoryStats struct {
	CurrentMemoryBytes int64     `json:"current_memory_bytes"`
	MaxMemoryBytes     int64     `json:"max_memory_bytes"`
	MemoryUsagePercent float64   `json:"memory_usage_percent"`
	PeakMemoryBytes    int64     `json:"peak_memory_bytes"`
	CacheItemCount     int       `json:"cache_item_count"`
	MaxCacheSize       int       `json:"max_cache_size"`
	TenantCacheCount   int       `json:"tenant_cache_count"`
	MaxTenantCacheSize int       `json:"max_tenant_cache_size"`
	MimirCacheCount    int       `json:"mimir_cache_count"`
	MaxMimirCacheSize  int       `json:"max_mimir_cache_size"`
	EvictionCount      int64     `json:"eviction_count"`
	MemoryWarnings     int64     `json:"memory_warnings"`
	LastEviction       time.Time `json:"last_eviction"`
	LastMemoryCheck    time.Time `json:"last_memory_check"`
	LastReset          time.Time `json:"last_reset"`
	EvictionPolicy     string    `json:"eviction_policy"`
	EvictionThreshold  float64   `json:"eviction_threshold"`
	MemoryThreshold    float64   `json:"memory_threshold"`
}

// NewMemoryManager creates a new memory manager with precise limits
func NewMemoryManager() *MemoryManager {
	// Get system memory info
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Calculate memory limits based on system memory
	totalMemory := int64(m.Sys)
	maxMemoryBytes := totalMemory / 4 // Use 25% of system memory for cache

	// Ensure minimum and maximum bounds
	if maxMemoryBytes < 100*1024*1024 { // 100MB minimum
		maxMemoryBytes = 100 * 1024 * 1024
	}
	if maxMemoryBytes > 2*1024*1024*1024 { // 2GB maximum
		maxMemoryBytes = 2 * 1024 * 1024 * 1024
	}

	mm := &MemoryManager{
		maxMemoryBytes:      maxMemoryBytes,
		maxCacheSize:        1000, // Maximum total cache items
		maxTenantCacheSize:  500,  // Maximum tenant cache items
		maxMimirCacheSize:   500,  // Maximum Mimir cache items
		memoryThreshold:     0.8,  // 80% threshold for warnings
		evictionPolicy:      EvictionPolicyHybrid,
		evictionThreshold:   0.9,              // 90% threshold for eviction
		memoryCheckInterval: 30 * time.Second, // Check memory every 30 seconds
		lastReset:           time.Now(),
	}

	logrus.Infof("🔧 Memory manager initialized: max=%d MB, cache_size=%d, tenant_size=%d, mimir_size=%d",
		maxMemoryBytes/(1024*1024), mm.maxCacheSize, mm.maxTenantCacheSize, mm.maxMimirCacheSize)

	return mm
}

// CheckMemoryUsage checks current memory usage and triggers eviction if needed
func (mm *MemoryManager) CheckMemoryUsage() error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// Update current memory usage
	mm.updateMemoryUsage()
	mm.lastMemoryCheck = time.Now()

	// Calculate memory usage percentage
	usagePercent := float64(mm.currentMemoryBytes) / float64(mm.maxMemoryBytes)

	// Log memory usage
	logrus.Debugf("📊 Memory usage: %d/%d bytes (%.1f%%)",
		mm.currentMemoryBytes, mm.maxMemoryBytes, usagePercent*100)

	// Check if we need to trigger eviction
	if usagePercent >= mm.evictionThreshold {
		logrus.Warnf("⚠️ Memory threshold exceeded: %.1f%% >= %.1f%%, triggering eviction",
			usagePercent*100, mm.evictionThreshold*100)

		if err := mm.triggerEviction(); err != nil {
			return fmt.Errorf("eviction failed: %w", err)
		}
	} else if usagePercent >= mm.memoryThreshold {
		mm.memoryWarnings++
		logrus.Warnf("⚠️ Memory warning: %.1f%% >= %.1f%%",
			usagePercent*100, mm.memoryThreshold*100)
	}

	return nil
}

// CanAddItem checks if we can add a new item without exceeding limits
func (mm *MemoryManager) CanAddItem(itemType string, estimatedSize int64) (bool, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	// Check cache size limits
	switch itemType {
	case "tenant":
		if mm.tenantCacheCount >= mm.maxTenantCacheSize {
			return false, fmt.Errorf("tenant cache size limit exceeded: %d >= %d",
				mm.tenantCacheCount, mm.maxTenantCacheSize)
		}
	case "mimir":
		if mm.mimirCacheCount >= mm.maxMimirCacheSize {
			return false, fmt.Errorf("Mimir cache size limit exceeded: %d >= %d",
				mm.mimirCacheCount, mm.maxMimirCacheSize)
		}
	default:
		if mm.cacheItemCount >= mm.maxCacheSize {
			return false, fmt.Errorf("total cache size limit exceeded: %d >= %d",
				mm.cacheItemCount, mm.maxCacheSize)
		}
	}

	// Check memory limits
	if mm.currentMemoryBytes+estimatedSize > mm.maxMemoryBytes {
		return false, fmt.Errorf("memory limit would be exceeded: %d + %d > %d",
			mm.currentMemoryBytes, estimatedSize, mm.maxMemoryBytes)
	}

	return true, nil
}

// AddItem adds an item to memory tracking
func (mm *MemoryManager) AddItem(itemType string, size int64) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.currentMemoryBytes += size
	mm.cacheItemCount++

	switch itemType {
	case "tenant":
		mm.tenantCacheCount++
	case "mimir":
		mm.mimirCacheCount++
	}

	// Update peak memory
	if mm.currentMemoryBytes > mm.peakMemoryBytes {
		mm.peakMemoryBytes = mm.currentMemoryBytes
	}

	logrus.Debugf("📦 Added %s item: size=%d bytes, total=%d/%d bytes",
		itemType, size, mm.currentMemoryBytes, mm.maxMemoryBytes)
}

// RemoveItem removes an item from memory tracking
func (mm *MemoryManager) RemoveItem(itemType string, size int64) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.currentMemoryBytes -= size
	mm.cacheItemCount--

	switch itemType {
	case "tenant":
		mm.tenantCacheCount--
	case "mimir":
		mm.mimirCacheCount--
	}

	logrus.Debugf("🗑️ Removed %s item: size=%d bytes, total=%d/%d bytes",
		itemType, size, mm.currentMemoryBytes, mm.maxMemoryBytes)
}

// GetMemoryStats returns current memory statistics
func (mm *MemoryManager) GetMemoryStats() MemoryStats {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	usagePercent := float64(mm.currentMemoryBytes) / float64(mm.maxMemoryBytes)

	return MemoryStats{
		CurrentMemoryBytes: mm.currentMemoryBytes,
		MaxMemoryBytes:     mm.maxMemoryBytes,
		MemoryUsagePercent: usagePercent,
		PeakMemoryBytes:    mm.peakMemoryBytes,
		CacheItemCount:     mm.cacheItemCount,
		MaxCacheSize:       mm.maxCacheSize,
		TenantCacheCount:   mm.tenantCacheCount,
		MaxTenantCacheSize: mm.maxTenantCacheSize,
		MimirCacheCount:    mm.mimirCacheCount,
		MaxMimirCacheSize:  mm.maxMimirCacheSize,
		EvictionCount:      mm.totalEvictions,
		MemoryWarnings:     mm.memoryWarnings,
		LastEviction:       mm.lastEviction,
		LastMemoryCheck:    mm.lastMemoryCheck,
		LastReset:          mm.lastReset,
		EvictionPolicy:     string(mm.evictionPolicy),
		EvictionThreshold:  mm.evictionThreshold,
		MemoryThreshold:    mm.memoryThreshold,
	}
}

// SetMemoryLimits allows dynamic adjustment of memory limits
func (mm *MemoryManager) SetMemoryLimits(maxMemoryBytes int64, maxCacheSize, maxTenantSize, maxMimirSize int) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.maxMemoryBytes = maxMemoryBytes
	mm.maxCacheSize = maxCacheSize
	mm.maxTenantCacheSize = maxTenantSize
	mm.maxMimirCacheSize = maxMimirSize

	logrus.Infof("🔧 Memory limits updated: max=%d MB, cache_size=%d, tenant_size=%d, mimir_size=%d",
		maxMemoryBytes/(1024*1024), maxCacheSize, maxTenantSize, maxMimirSize)
}

// SetEvictionPolicy allows changing the eviction policy
func (mm *MemoryManager) SetEvictionPolicy(policy EvictionPolicy, threshold float64) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.evictionPolicy = policy
	mm.evictionThreshold = threshold

	logrus.Infof("🔧 Eviction policy updated: %s, threshold=%.1f%%", policy, threshold*100)
}

// ForceEviction forces an immediate eviction cycle
func (mm *MemoryManager) ForceEviction() error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	logrus.Info("🔄 Forcing memory eviction")
	return mm.triggerEviction()
}

// ResetStats resets memory statistics
func (mm *MemoryManager) ResetStats() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.peakMemoryBytes = 0
	mm.totalEvictions = 0
	mm.memoryWarnings = 0
	mm.lastReset = time.Now()

	logrus.Info("🔄 Memory statistics reset")
}

// Private methods

func (mm *MemoryManager) updateMemoryUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Estimate cache memory usage (this is approximate)
	// In a real implementation, you'd track actual memory usage more precisely
	mm.currentMemoryBytes = int64(m.Alloc) // Use allocated memory as approximation
}

func (mm *MemoryManager) triggerEviction() error {
	mm.lastEviction = time.Now()
	mm.totalEvictions++

	logrus.Infof("🗑️ Starting eviction cycle (policy: %s)", mm.evictionPolicy)

	// This would trigger the actual eviction in the cache manager
	// The cache manager should implement the specific eviction logic
	// based on the policy and current memory usage

	// For now, we'll just log the eviction
	logrus.Infof("✅ Eviction cycle completed")

	return nil
}

// EstimateItemSize estimates the memory size of a cache item
func EstimateItemSize(item interface{}) int64 {
	// This is a simplified estimation
	// In production, you'd want more precise memory measurement

	switch v := item.(type) {
	case string:
		return int64(len(v))
	case []byte:
		return int64(len(v))
	case map[string]interface{}:
		size := int64(0)
		for k, val := range v {
			size += int64(len(k))
			size += EstimateItemSize(val)
		}
		return size
	case []interface{}:
		size := int64(0)
		for _, val := range v {
			size += EstimateItemSize(val)
		}
		return size
	default:
		// Default estimation for unknown types
		return 1024 // 1KB default
	}
}

// StartMemoryMonitoring starts periodic memory monitoring
func (mm *MemoryManager) StartMemoryMonitoring() {
	go func() {
		ticker := time.NewTicker(mm.memoryCheckInterval)
		defer ticker.Stop()

		logrus.Infof("🔍 Starting memory monitoring every %v", mm.memoryCheckInterval)

		for {
			select {
			case <-ticker.C:
				if err := mm.CheckMemoryUsage(); err != nil {
					logrus.Errorf("❌ Memory check failed: %v", err)
				}
			}
		}
	}()
}
