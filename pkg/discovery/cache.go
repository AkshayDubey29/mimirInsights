package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// DiscoveryCache manages cached discovery results for efficient serving
type DiscoveryCache struct {
	mu sync.RWMutex

	// Tenant discovery cache
	tenantDiscoveryCache   *ComprehensiveTenantDiscoveryResult
	tenantCacheLastUpdated time.Time
	tenantCacheTTL         time.Duration

	// Mimir discovery cache
	mimirDiscoveryCache   *ComprehensiveMimirDiscoveryResult
	mimirCacheLastUpdated time.Time
	mimirCacheTTL         time.Duration

	// Cache configuration
	defaultTTL               time.Duration
	backgroundRefreshEnabled bool
	refreshInterval          time.Duration

	// Discovery engine reference
	engine *Engine
}

// NewDiscoveryCache creates a new discovery cache instance
func NewDiscoveryCache(engine *Engine) *DiscoveryCache {
	cache := &DiscoveryCache{
		tenantCacheTTL:           5 * time.Minute,  // 5 minutes TTL for tenant discovery
		mimirCacheTTL:            10 * time.Minute, // 10 minutes TTL for Mimir discovery
		defaultTTL:               5 * time.Minute,
		backgroundRefreshEnabled: true,
		refreshInterval:          2 * time.Minute, // Refresh every 2 minutes
		engine:                   engine,
	}

	// Start background refresh if enabled
	if cache.backgroundRefreshEnabled {
		go cache.startBackgroundRefresh()
	}

	logrus.Info("🔧 Discovery cache initialized with background refresh enabled")
	return cache
}

// GetTenantDiscovery returns cached tenant discovery results or performs fresh discovery
func (c *DiscoveryCache) GetTenantDiscovery(ctx context.Context) (*ComprehensiveTenantDiscoveryResult, error) {
	c.mu.RLock()

	// Check if cache is valid
	if c.isTenantCacheValid() {
		logrus.Debug("📋 Serving tenant discovery from cache")
		result := c.tenantDiscoveryCache
		c.mu.RUnlock()
		return result, nil
	}

	c.mu.RUnlock()

	// Cache is invalid or expired, perform fresh discovery
	logrus.Info("🔄 Tenant discovery cache expired, performing fresh discovery")
	return c.refreshTenantDiscovery(ctx)
}

// GetMimirDiscovery returns cached Mimir discovery results or performs fresh discovery
func (c *DiscoveryCache) GetMimirDiscovery(ctx context.Context) (*ComprehensiveMimirDiscoveryResult, error) {
	c.mu.RLock()

	// Check if cache is valid
	if c.isMimirCacheValid() {
		logrus.Debug("📋 Serving Mimir discovery from cache")
		result := c.mimirDiscoveryCache
		c.mu.RUnlock()
		return result, nil
	}

	c.mu.RUnlock()

	// Cache is invalid or expired, perform fresh discovery
	logrus.Info("🔄 Mimir discovery cache expired, performing fresh discovery")
	return c.refreshMimirDiscovery(ctx)
}

// RefreshTenantDiscovery forces a refresh of tenant discovery cache
func (c *DiscoveryCache) RefreshTenantDiscovery(ctx context.Context) (*ComprehensiveTenantDiscoveryResult, error) {
	logrus.Info("🔄 Forcing tenant discovery cache refresh")
	return c.refreshTenantDiscovery(ctx)
}

// RefreshMimirDiscovery forces a refresh of Mimir discovery cache
func (c *DiscoveryCache) RefreshMimirDiscovery(ctx context.Context) (*ComprehensiveMimirDiscoveryResult, error) {
	logrus.Info("🔄 Forcing Mimir discovery cache refresh")
	return c.refreshMimirDiscovery(ctx)
}

// RefreshAllDiscovery forces a refresh of both tenant and Mimir discovery caches
func (c *DiscoveryCache) RefreshAllDiscovery(ctx context.Context) error {
	logrus.Info("🔄 Forcing complete discovery cache refresh")

	// Perform both discoveries in parallel
	var tenantErr, mimirErr error

	// Use a WaitGroup to wait for both discoveries to complete
	var wg sync.WaitGroup
	wg.Add(2)

	// Tenant discovery
	go func() {
		defer wg.Done()
		_, tenantErr = c.refreshTenantDiscovery(ctx)
	}()

	// Mimir discovery
	go func() {
		defer wg.Done()
		_, mimirErr = c.refreshMimirDiscovery(ctx)
	}()

	// Wait for both to complete
	wg.Wait()

	// Check for errors
	if tenantErr != nil {
		logrus.Errorf("❌ Tenant discovery refresh failed: %v", tenantErr)
	}
	if mimirErr != nil {
		logrus.Errorf("❌ Mimir discovery refresh failed: %v", mimirErr)
	}

	if tenantErr != nil || mimirErr != nil {
		return fmt.Errorf("discovery refresh failed: tenant=%v, mimir=%v", tenantErr, mimirErr)
	}

	logrus.Info("✅ Complete discovery cache refresh completed successfully")
	return nil
}

// GetCacheStatus returns the current status of all caches
func (c *DiscoveryCache) GetCacheStatus() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"tenant_discovery": map[string]interface{}{
			"cached":          c.tenantDiscoveryCache != nil,
			"last_updated":    c.tenantCacheLastUpdated,
			"ttl":             c.tenantCacheTTL,
			"is_valid":        c.isTenantCacheValid(),
			"cache_age":       time.Since(c.tenantCacheLastUpdated),
			"component_count": c.getTenantComponentCount(),
		},
		"mimir_discovery": map[string]interface{}{
			"cached":          c.mimirDiscoveryCache != nil,
			"last_updated":    c.mimirCacheLastUpdated,
			"ttl":             c.mimirCacheTTL,
			"is_valid":        c.isMimirCacheValid(),
			"cache_age":       time.Since(c.mimirCacheLastUpdated),
			"component_count": c.getMimirComponentCount(),
		},
		"cache_config": map[string]interface{}{
			"background_refresh_enabled": c.backgroundRefreshEnabled,
			"refresh_interval":           c.refreshInterval,
			"default_ttl":                c.defaultTTL,
		},
	}
}

// SetCacheTTL sets the TTL for specific cache types
func (c *DiscoveryCache) SetCacheTTL(cacheType string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch cacheType {
	case "tenant":
		c.tenantCacheTTL = ttl
		logrus.Infof("🔧 Tenant discovery cache TTL set to %v", ttl)
	case "mimir":
		c.mimirCacheTTL = ttl
		logrus.Infof("🔧 Mimir discovery cache TTL set to %v", ttl)
	default:
		logrus.Warnf("⚠️ Unknown cache type: %s", cacheType)
	}
}

// EnableBackgroundRefresh enables or disables background cache refresh
func (c *DiscoveryCache) EnableBackgroundRefresh(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.backgroundRefreshEnabled = enabled
	if enabled {
		go c.startBackgroundRefresh()
		logrus.Info("🔧 Background cache refresh enabled")
	} else {
		logrus.Info("🔧 Background cache refresh disabled")
	}
}

// ClearCache clears all cached data
func (c *DiscoveryCache) ClearCache() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tenantDiscoveryCache = nil
	c.mimirDiscoveryCache = nil
	c.tenantCacheLastUpdated = time.Time{}
	c.mimirCacheLastUpdated = time.Time{}

	logrus.Info("🧹 Discovery cache cleared")
}

// Private methods

func (c *DiscoveryCache) refreshTenantDiscovery(ctx context.Context) (*ComprehensiveTenantDiscoveryResult, error) {
	// Perform fresh tenant discovery
	result, err := c.engine.DiscoverTenantsComprehensive(ctx)
	if err != nil {
		logrus.Errorf("❌ Tenant discovery failed: %v", err)
		return nil, err
	}

	// Update cache
	c.mu.Lock()
	c.tenantDiscoveryCache = result
	c.tenantCacheLastUpdated = time.Now()
	c.mu.Unlock()

	logrus.Infof("✅ Tenant discovery cache updated with %d tenants", len(result.ConsolidatedTenants))
	return result, nil
}

func (c *DiscoveryCache) refreshMimirDiscovery(ctx context.Context) (*ComprehensiveMimirDiscoveryResult, error) {
	// Perform fresh Mimir discovery
	result, err := c.engine.DiscoverMimirComprehensive(ctx)
	if err != nil {
		logrus.Errorf("❌ Mimir discovery failed: %v", err)
		return nil, err
	}

	// Update cache
	c.mu.Lock()
	c.mimirDiscoveryCache = result
	c.mimirCacheLastUpdated = time.Now()
	c.mu.Unlock()

	logrus.Infof("✅ Mimir discovery cache updated with %d components", len(result.ConsolidatedComponents))
	return result, nil
}

func (c *DiscoveryCache) isTenantCacheValid() bool {
	if c.tenantDiscoveryCache == nil {
		return false
	}
	return time.Since(c.tenantCacheLastUpdated) < c.tenantCacheTTL
}

func (c *DiscoveryCache) isMimirCacheValid() bool {
	if c.mimirDiscoveryCache == nil {
		return false
	}
	return time.Since(c.mimirCacheLastUpdated) < c.mimirCacheTTL
}

func (c *DiscoveryCache) getTenantComponentCount() int {
	if c.tenantDiscoveryCache == nil {
		return 0
	}
	return len(c.tenantDiscoveryCache.ConsolidatedTenants)
}

func (c *DiscoveryCache) getMimirComponentCount() int {
	if c.mimirDiscoveryCache == nil {
		return 0
	}
	return len(c.mimirDiscoveryCache.ConsolidatedComponents)
}

func (c *DiscoveryCache) startBackgroundRefresh() {
	ticker := time.NewTicker(c.refreshInterval)
	defer ticker.Stop()

	logrus.Infof("🔄 Starting background cache refresh every %v", c.refreshInterval)

	for {
		select {
		case <-ticker.C:
			ctx := context.Background()

			// Check if cache needs refresh
			c.mu.RLock()
			tenantNeedsRefresh := !c.isTenantCacheValid()
			mimirNeedsRefresh := !c.isMimirCacheValid()
			c.mu.RUnlock()

			if tenantNeedsRefresh || mimirNeedsRefresh {
				logrus.Info("🔄 Background refresh: Cache expired, performing discovery")

				if err := c.RefreshAllDiscovery(ctx); err != nil {
					logrus.Errorf("❌ Background refresh failed: %v", err)
				} else {
					logrus.Info("✅ Background refresh completed successfully")
				}
			} else {
				logrus.Debug("🔄 Background refresh: Cache still valid, skipping refresh")
			}
		}
	}
}

// InitialDiscovery performs initial discovery on startup
func (c *DiscoveryCache) InitialDiscovery(ctx context.Context) error {
	logrus.Info("🚀 Performing initial discovery on startup")

	// Perform both discoveries in parallel for faster startup
	var tenantErr, mimirErr error
	var wg sync.WaitGroup
	wg.Add(2)

	// Tenant discovery
	go func() {
		defer wg.Done()
		_, tenantErr = c.refreshTenantDiscovery(ctx)
	}()

	// Mimir discovery
	go func() {
		defer wg.Done()
		_, mimirErr = c.refreshMimirDiscovery(ctx)
	}()

	// Wait for both to complete
	wg.Wait()

	// Check for errors
	if tenantErr != nil {
		logrus.Errorf("❌ Initial tenant discovery failed: %v", tenantErr)
	}
	if mimirErr != nil {
		logrus.Errorf("❌ Initial Mimir discovery failed: %v", mimirErr)
	}

	if tenantErr != nil || mimirErr != nil {
		return fmt.Errorf("initial discovery failed: tenant=%v, mimir=%v", tenantErr, mimirErr)
	}

	logrus.Info("✅ Initial discovery completed successfully")
	return nil
}
