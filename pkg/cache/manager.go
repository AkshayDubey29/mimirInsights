package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/akshaydubey29/mimirInsights/pkg/discovery"
	"github.com/akshaydubey29/mimirInsights/pkg/limits"
	"github.com/akshaydubey29/mimirInsights/pkg/metrics"
	"github.com/sirupsen/logrus"
)

// Manager handles data collection, caching, and serving
type Manager struct {
	discoveryEngine *discovery.Engine
	metricsClient   *metrics.Client
	limitsAnalyzer  *limits.Analyzer

	// Cache storage
	cache     *Cache
	cacheLock sync.RWMutex

	// Background collection
	collectionInterval time.Duration
	stopChan           chan struct{}
	isRunning          bool
	runningLock        sync.Mutex

	// Discovery cache TTL settings
	tenantCacheTTL time.Duration
	mimirCacheTTL  time.Duration

	// Memory management
	memoryManager *MemoryManager
}

// Cache holds all cached data
type Cache struct {
	// Discovery data
	DiscoveryResult *discovery.DiscoveryResult `json:"discovery_result"`

	// Comprehensive discovery data (new multi-strategy)
	TenantDiscoveryCache   *discovery.ComprehensiveTenantDiscoveryResult `json:"tenant_discovery_cache"`
	MimirDiscoveryCache    *discovery.ComprehensiveMimirDiscoveryResult  `json:"mimir_discovery_cache"`
	TenantCacheLastUpdated time.Time                                     `json:"tenant_cache_last_updated"`
	MimirCacheLastUpdated  time.Time                                     `json:"mimir_cache_last_updated"`

	// Metrics data
	TenantMetrics map[string]*TenantMetricsData `json:"tenant_metrics"`

	// Limits analysis
	LimitsSummary map[string]*limits.TenantLimits `json:"limits_summary"`

	// Auto-discovered limits
	AutoDiscoveredLimits *limits.DiscoveredLimits `json:"auto_discovered_limits"`

	// Environment info
	Environment map[string]interface{} `json:"environment"`

	// Metadata
	LastUpdated     time.Time `json:"last_updated"`
	LastCollection  time.Time `json:"last_collection"`
	CollectionCount int64     `json:"collection_count"`
}

// TenantMetricsData holds metrics for a specific tenant
type TenantMetricsData struct {
	IngestionRate    map[string]float64 `json:"ingestion_rate"`
	ActiveSeries     map[string]float64 `json:"active_series"`
	RejectionRate    map[string]float64 `json:"rejection_rate"`
	QueryLatency     map[string]float64 `json:"query_latency"`
	LastUpdated      time.Time          `json:"last_updated"`
	CollectionErrors []string           `json:"collection_errors"`
}

// NewManager creates a new cache manager
func NewManager(discoveryEngine *discovery.Engine, metricsClient *metrics.Client, limitsAnalyzer *limits.Analyzer) *Manager {
	manager := &Manager{
		discoveryEngine:    discoveryEngine,
		metricsClient:      metricsClient,
		limitsAnalyzer:     limitsAnalyzer,
		cache:              &Cache{},
		collectionInterval: 30 * time.Second, // Collect every 30 seconds
		stopChan:           make(chan struct{}),
		tenantCacheTTL:     5 * time.Minute,  // 5 minutes TTL for tenant discovery
		mimirCacheTTL:      10 * time.Minute, // 10 minutes TTL for Mimir discovery
	}

	// Initialize memory manager
	manager.memoryManager = NewMemoryManager()

	// Start memory monitoring
	manager.memoryManager.StartMemoryMonitoring()

	return manager
}

// Start begins the background data collection
func (m *Manager) Start(ctx context.Context) error {
	m.runningLock.Lock()
	defer m.runningLock.Unlock()

	if m.isRunning {
		return fmt.Errorf("cache manager is already running")
	}

	m.isRunning = true
	logrus.Info("Starting cache manager with 30-second collection interval")

	// Perform initial collection
	if err := m.collectAllData(ctx); err != nil {
		logrus.Errorf("Initial data collection failed: %v", err)
	}

	// Start background collection
	go m.runBackgroundCollection(ctx)

	return nil
}

// Stop stops the background data collection
func (m *Manager) Stop() {
	m.runningLock.Lock()
	defer m.runningLock.Unlock()

	if !m.isRunning {
		return
	}

	close(m.stopChan)
	m.isRunning = false
	logrus.Info("Cache manager stopped")
}

// runBackgroundCollection runs the periodic data collection
func (m *Manager) runBackgroundCollection(ctx context.Context) {
	ticker := time.NewTicker(m.collectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := m.collectAllData(ctx); err != nil {
				logrus.Errorf("Background data collection failed: %v", err)
			}
		case <-m.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// collectAllData performs a complete data collection cycle
func (m *Manager) collectAllData(ctx context.Context) error {
	start := time.Now()
	logrus.Infof("🔄 [CACHE] Starting data collection cycle")

	// Collect discovery data
	logrus.Infof("🔍 [CACHE] Collecting discovery data...")
	discoveryResult, err := m.discoveryEngine.DiscoverAll(ctx)
	if err != nil {
		logrus.Errorf("❌ [CACHE] Discovery collection failed: %v", err)
		return fmt.Errorf("discovery collection failed: %w", err)
	}
	logrus.Infof("✅ [CACHE] Discovery completed: %d tenants, %d Mimir components",
		len(discoveryResult.TenantNamespaces), len(discoveryResult.MimirComponents))

	// Collect metrics for all tenants (including detected tenants)
	tenantMetrics := make(map[string]*TenantMetricsData)
	tenantNames := make([]string, 0)

	// Add discovered tenant namespaces
	for _, tenant := range discoveryResult.TenantNamespaces {
		tenantNames = append(tenantNames, tenant.Name)
	}

	// Add detected tenants from environment
	if discoveryResult.Environment != nil && discoveryResult.Environment.DetectedTenants != nil {
		logrus.Infof("🔍 [CACHE] Found %d detected tenants in environment", len(discoveryResult.Environment.DetectedTenants))
		for _, detectedTenant := range discoveryResult.Environment.DetectedTenants {
			// Check if this tenant is already in the list
			exists := false
			for _, existingTenant := range tenantNames {
				if existingTenant == detectedTenant.Name {
					exists = true
					break
				}
			}
			if !exists {
				tenantNames = append(tenantNames, detectedTenant.Name)
				logrus.Debugf("📋 [CACHE] Added detected tenant: %s", detectedTenant.Name)
			}
		}
	}

	logrus.Infof("📊 [CACHE] Collecting metrics for %d total tenants...", len(tenantNames))
	for _, tenantName := range tenantNames {
		logrus.Debugf("📈 [CACHE] Collecting metrics for tenant: %s", tenantName)
		metricsData, err := m.collectTenantMetrics(ctx, tenantName)
		if err != nil {
			logrus.Warnf("⚠️ [CACHE] Failed to collect metrics for tenant %s: %v", tenantName, err)
			metricsData = &TenantMetricsData{
				CollectionErrors: []string{err.Error()},
				LastUpdated:      time.Now(),
			}
		} else {
			logrus.Debugf("✅ [CACHE] Successfully collected metrics for tenant: %s", tenantName)
		}
		tenantMetrics[tenantName] = metricsData
	}

	// Collect limits analysis
	logrus.Infof("⚖️ [CACHE] Collecting limits analysis for %d tenants...", len(tenantNames))
	limitsSummary, err := m.limitsAnalyzer.GetTenantLimitsSummary(ctx, tenantNames)
	if err != nil {
		logrus.Warnf("⚠️ [CACHE] Failed to collect limits analysis: %v", err)
		limitsSummary = make(map[string]*limits.TenantLimits)
	} else {
		logrus.Infof("✅ [CACHE] Successfully collected limits for %d tenants", len(limitsSummary))
	}

	// Collect auto-discovered limits
	logrus.Infof("🔍 [CACHE] Collecting auto-discovered limits...")
	autoDiscovery := limits.NewAutoDiscovery(m.discoveryEngine.GetK8sClient())
	discoveredLimits, err := autoDiscovery.DiscoverAllLimits(ctx, discoveryResult.Environment.MimirNamespace)
	if err != nil {
		logrus.Warnf("⚠️ [CACHE] Failed to get auto-discovered limits: %v", err)
		discoveredLimits = &limits.DiscoveredLimits{
			GlobalLimits:  make(map[string]interface{}),
			TenantLimits:  make(map[string]limits.TenantLimit),
			ConfigSources: []limits.ConfigSource{},
			LastUpdated:   time.Now(),
		}
	} else {
		logrus.Infof("✅ [CACHE] Auto-discovery completed: %d global limits, %d tenant limits from %d sources",
			len(discoveredLimits.GlobalLimits), len(discoveredLimits.TenantLimits), len(discoveredLimits.ConfigSources))
	}

	// Build environment info
	environment := map[string]interface{}{
		"cluster_info":         discoveryResult.Environment,
		"auto_discovered":      discoveredLimits,
		"mimir_components":     discoveryResult.MimirComponents,
		"detected_tenants":     discoveryResult.Environment.DetectedTenants,
		"data_source_status":   discoveryResult.Environment.DataSource,
		"is_production":        discoveryResult.Environment.IsProduction,
		"total_config_sources": len(discoveredLimits.ConfigSources),
		"last_updated":         discoveryResult.LastUpdated,
	}

	// Update cache
	m.cacheLock.Lock()
	m.cache.DiscoveryResult = discoveryResult
	m.cache.TenantMetrics = tenantMetrics
	m.cache.LimitsSummary = limitsSummary
	m.cache.AutoDiscoveredLimits = discoveredLimits
	m.cache.Environment = environment
	m.cache.LastUpdated = time.Now()
	m.cache.LastCollection = time.Now()
	m.cache.CollectionCount++
	m.cacheLock.Unlock()

	logrus.Infof("✅ [CACHE] Data collection completed in %v. Found %d tenants, %d Mimir components",
		time.Since(start), len(tenantNames), len(discoveryResult.MimirComponents))

	return nil
}

// collectTenantMetrics collects metrics for a specific tenant
func (m *Manager) collectTenantMetrics(ctx context.Context, tenantName string) (*TenantMetricsData, error) {
	metricsData := &TenantMetricsData{
		IngestionRate: make(map[string]float64),
		ActiveSeries:  make(map[string]float64),
		RejectionRate: make(map[string]float64),
		QueryLatency:  make(map[string]float64),
		LastUpdated:   time.Now(),
	}

	// Collect metrics for different time ranges
	timeRanges := map[string]time.Duration{
		"1h":  1 * time.Hour,
		"6h":  6 * time.Hour,
		"24h": 24 * time.Hour,
		"7d":  7 * 24 * time.Hour,
	}

	for rangeName, duration := range timeRanges {
		timeRange := metrics.CreateTimeRange(duration, "1m")

		// Ingestion rate
		if rateSeries, err := m.metricsClient.GetIngestionRate(ctx, tenantName, timeRange); err == nil && len(rateSeries) > 0 {
			// Get the latest value
			if len(rateSeries[0].Values) > 0 {
				metricsData.IngestionRate[rangeName] = rateSeries[0].Values[len(rateSeries[0].Values)-1].Value
			}
		}

		// Active series
		if seriesData, err := m.metricsClient.GetActiveSeries(ctx, tenantName, timeRange); err == nil && len(seriesData) > 0 {
			// Get the latest value
			if len(seriesData[0].Values) > 0 {
				metricsData.ActiveSeries[rangeName] = seriesData[0].Values[len(seriesData[0].Values)-1].Value
			}
		}

		// Rejected samples (using available method)
		if rejectedSeries, err := m.metricsClient.GetRejectedSamples(ctx, tenantName, timeRange); err == nil && len(rejectedSeries) > 0 {
			// Get the latest value
			if len(rejectedSeries[0].Values) > 0 {
				metricsData.RejectionRate[rangeName] = rejectedSeries[0].Values[len(rejectedSeries[0].Values)-1].Value
			}
		}

		// Memory usage as a proxy for query latency (since query latency method doesn't exist)
		if memorySeries, err := m.metricsClient.GetMemoryUsage(ctx, tenantName, timeRange); err == nil && len(memorySeries) > 0 {
			// Get the latest value
			if len(memorySeries[0].Values) > 0 {
				metricsData.QueryLatency[rangeName] = memorySeries[0].Values[len(memorySeries[0].Values)-1].Value
			}
		}
	}

	return metricsData, nil
}

// GetDiscoveryResult returns cached discovery data
func (m *Manager) GetDiscoveryResult() *discovery.DiscoveryResult {
	m.cacheLock.RLock()
	defer m.cacheLock.RUnlock()
	return m.cache.DiscoveryResult
}

// GetTenantMetrics returns cached metrics for a tenant
func (m *Manager) GetTenantMetrics(tenantName string) *TenantMetricsData {
	m.cacheLock.RLock()
	defer m.cacheLock.RUnlock()
	return m.cache.TenantMetrics[tenantName]
}

// GetAllTenantMetrics returns all cached tenant metrics
func (m *Manager) GetAllTenantMetrics() map[string]*TenantMetricsData {
	m.cacheLock.RLock()
	defer m.cacheLock.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*TenantMetricsData)
	for k, v := range m.cache.TenantMetrics {
		result[k] = v
	}
	return result
}

// GetLimitsSummary returns cached limits analysis
func (m *Manager) GetLimitsSummary() map[string]*limits.TenantLimits {
	m.cacheLock.RLock()
	defer m.cacheLock.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*limits.TenantLimits)
	for k, v := range m.cache.LimitsSummary {
		result[k] = v
	}
	return result
}

// GetTenantLimitsSummary returns cached limits for a specific tenant
func (m *Manager) GetTenantLimitsSummary(tenantName string) *limits.TenantLimits {
	m.cacheLock.RLock()
	defer m.cacheLock.RUnlock()
	return m.cache.LimitsSummary[tenantName]
}

// GetAutoDiscoveredLimits returns cached auto-discovered limits
func (m *Manager) GetAutoDiscoveredLimits() *limits.DiscoveredLimits {
	m.cacheLock.RLock()
	defer m.cacheLock.RUnlock()
	return m.cache.AutoDiscoveredLimits
}

// GetEnvironment returns cached environment data
func (m *Manager) GetEnvironment() map[string]interface{} {
	m.cacheLock.RLock()
	defer m.cacheLock.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]interface{})
	for k, v := range m.cache.Environment {
		result[k] = v
	}
	return result
}

// GetCacheStatus returns cache status information
func (m *Manager) GetCacheStatus() map[string]interface{} {
	m.cacheLock.RLock()
	defer m.cacheLock.RUnlock()

	// Get memory statistics
	memoryStats := m.memoryManager.GetMemoryStats()

	return map[string]interface{}{
		"general_cache": map[string]interface{}{
			"last_updated":        m.cache.LastUpdated,
			"last_collection":     m.cache.LastCollection,
			"collection_count":    m.cache.CollectionCount,
			"collection_interval": m.collectionInterval.String(),
			"is_running":          m.isRunning,
			"tenant_count":        len(m.cache.TenantMetrics),
			"mimir_components":    len(m.cache.DiscoveryResult.MimirComponents),
		},
		"tenant_discovery_cache": map[string]interface{}{
			"cached":          m.cache.TenantDiscoveryCache != nil,
			"last_updated":    m.cache.TenantCacheLastUpdated,
			"ttl":             m.tenantCacheTTL,
			"is_valid":        m.isTenantCacheValid(),
			"cache_age":       time.Since(m.cache.TenantCacheLastUpdated),
			"component_count": m.getTenantDiscoveryCount(),
		},
		"mimir_discovery_cache": map[string]interface{}{
			"cached":          m.cache.MimirDiscoveryCache != nil,
			"last_updated":    m.cache.MimirCacheLastUpdated,
			"ttl":             m.mimirCacheTTL,
			"is_valid":        m.isMimirCacheValid(),
			"cache_age":       time.Since(m.cache.MimirCacheLastUpdated),
			"component_count": m.getMimirDiscoveryCount(),
		},
		"memory_management": map[string]interface{}{
			"current_memory_bytes":  memoryStats.CurrentMemoryBytes,
			"max_memory_bytes":      memoryStats.MaxMemoryBytes,
			"memory_usage_percent":  memoryStats.MemoryUsagePercent,
			"peak_memory_bytes":     memoryStats.PeakMemoryBytes,
			"cache_item_count":      memoryStats.CacheItemCount,
			"max_cache_size":        memoryStats.MaxCacheSize,
			"tenant_cache_count":    memoryStats.TenantCacheCount,
			"max_tenant_cache_size": memoryStats.MaxTenantCacheSize,
			"mimir_cache_count":     memoryStats.MimirCacheCount,
			"max_mimir_cache_size":  memoryStats.MaxMimirCacheSize,
			"eviction_count":        memoryStats.EvictionCount,
			"memory_warnings":       memoryStats.MemoryWarnings,
			"last_eviction":         memoryStats.LastEviction,
			"last_memory_check":     memoryStats.LastMemoryCheck,
			"eviction_policy":       memoryStats.EvictionPolicy,
			"eviction_threshold":    memoryStats.EvictionThreshold,
			"memory_threshold":      memoryStats.MemoryThreshold,
		},
	}
}

// ForceRefresh triggers an immediate data collection including discovery caches
func (m *Manager) ForceRefresh(ctx context.Context) error {
	logrus.Info("Force refreshing all cache data")

	// Refresh discovery caches in parallel with general data collection
	var discoveryErr error
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		discoveryErr = m.RefreshAllDiscovery(ctx)
	}()

	// Perform general data collection
	generalErr := m.collectAllData(ctx)

	// Wait for discovery refresh to complete
	wg.Wait()

	// Return error if either failed
	if generalErr != nil {
		return fmt.Errorf("general cache refresh failed: %w", generalErr)
	}
	if discoveryErr != nil {
		return fmt.Errorf("discovery cache refresh failed: %w", discoveryErr)
	}

	return nil
}

// GetTenantDiscovery returns cached tenant discovery results or performs fresh discovery
func (m *Manager) GetTenantDiscovery(ctx context.Context) (*discovery.ComprehensiveTenantDiscoveryResult, error) {
	m.cacheLock.RLock()

	// Check if cache is valid
	if m.isTenantCacheValid() {
		logrus.Debug("📋 Serving tenant discovery from cache")
		result := m.cache.TenantDiscoveryCache
		m.cacheLock.RUnlock()
		return result, nil
	}

	m.cacheLock.RUnlock()

	// Cache is invalid or expired, perform fresh discovery
	logrus.Info("🔄 Tenant discovery cache expired, performing fresh discovery")
	return m.refreshTenantDiscovery(ctx)
}

// GetMimirDiscovery returns cached Mimir discovery results or performs fresh discovery
func (m *Manager) GetMimirDiscovery(ctx context.Context) (*discovery.ComprehensiveMimirDiscoveryResult, error) {
	m.cacheLock.RLock()

	// Check if cache is valid
	if m.isMimirCacheValid() {
		logrus.Debug("📋 Serving Mimir discovery from cache")
		result := m.cache.MimirDiscoveryCache
		m.cacheLock.RUnlock()
		return result, nil
	}

	m.cacheLock.RUnlock()

	// Cache is invalid or expired, perform fresh discovery
	logrus.Info("🔄 Mimir discovery cache expired, performing fresh discovery")
	return m.refreshMimirDiscovery(ctx)
}

// RefreshTenantDiscovery forces a refresh of tenant discovery cache
func (m *Manager) RefreshTenantDiscovery(ctx context.Context) (*discovery.ComprehensiveTenantDiscoveryResult, error) {
	logrus.Info("🔄 Forcing tenant discovery cache refresh")
	return m.refreshTenantDiscovery(ctx)
}

// RefreshMimirDiscovery forces a refresh of Mimir discovery cache
func (m *Manager) RefreshMimirDiscovery(ctx context.Context) (*discovery.ComprehensiveMimirDiscoveryResult, error) {
	logrus.Info("🔄 Forcing Mimir discovery cache refresh")
	return m.refreshMimirDiscovery(ctx)
}

// RefreshAllDiscovery forces a refresh of both tenant and Mimir discovery caches
func (m *Manager) RefreshAllDiscovery(ctx context.Context) error {
	logrus.Info("🔄 Forcing complete discovery cache refresh")

	// Perform both discoveries in parallel
	var tenantErr, mimirErr error
	var wg sync.WaitGroup
	wg.Add(2)

	// Tenant discovery
	go func() {
		defer wg.Done()
		_, tenantErr = m.refreshTenantDiscovery(ctx)
	}()

	// Mimir discovery
	go func() {
		defer wg.Done()
		_, mimirErr = m.refreshMimirDiscovery(ctx)
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

// Private methods for discovery cache management

func (m *Manager) refreshTenantDiscovery(ctx context.Context) (*discovery.ComprehensiveTenantDiscoveryResult, error) {
	// Perform fresh tenant discovery
	result, err := m.discoveryEngine.DiscoverTenantsComprehensive(ctx)
	if err != nil {
		logrus.Errorf("❌ Tenant discovery failed: %v", err)
		return nil, err
	}

	// Estimate memory size
	estimatedSize := EstimateItemSize(result)

	// Check if we can add this item
	if canAdd, err := m.memoryManager.CanAddItem("tenant", estimatedSize); !canAdd {
		logrus.Warnf("⚠️ Cannot add tenant discovery to cache: %v", err)
		// Still return the result, but don't cache it
		return result, nil
	}

	// Update cache
	m.cacheLock.Lock()

	// Remove old item if exists
	if m.cache.TenantDiscoveryCache != nil {
		oldSize := EstimateItemSize(m.cache.TenantDiscoveryCache)
		m.memoryManager.RemoveItem("tenant", oldSize)
	}

	m.cache.TenantDiscoveryCache = result
	m.cache.TenantCacheLastUpdated = time.Now()

	// Add new item to memory tracking
	m.memoryManager.AddItem("tenant", estimatedSize)

	m.cacheLock.Unlock()

	logrus.Infof("✅ Tenant discovery cache updated with %d tenants (size: %d bytes)",
		len(result.ConsolidatedTenants), estimatedSize)
	return result, nil
}

func (m *Manager) refreshMimirDiscovery(ctx context.Context) (*discovery.ComprehensiveMimirDiscoveryResult, error) {
	// Perform fresh Mimir discovery
	result, err := m.discoveryEngine.DiscoverMimirComprehensive(ctx)
	if err != nil {
		logrus.Errorf("❌ Mimir discovery failed: %v", err)
		return nil, err
	}

	// Estimate memory size
	estimatedSize := EstimateItemSize(result)

	// Check if we can add this item
	if canAdd, err := m.memoryManager.CanAddItem("mimir", estimatedSize); !canAdd {
		logrus.Warnf("⚠️ Cannot add Mimir discovery to cache: %v", err)
		// Still return the result, but don't cache it
		return result, nil
	}

	// Update cache
	m.cacheLock.Lock()

	// Remove old item if exists
	if m.cache.MimirDiscoveryCache != nil {
		oldSize := EstimateItemSize(m.cache.MimirDiscoveryCache)
		m.memoryManager.RemoveItem("mimir", oldSize)
	}

	m.cache.MimirDiscoveryCache = result
	m.cache.MimirCacheLastUpdated = time.Now()

	// Add new item to memory tracking
	m.memoryManager.AddItem("mimir", estimatedSize)

	m.cacheLock.Unlock()

	logrus.Infof("✅ Mimir discovery cache updated with %d components (size: %d bytes)",
		len(result.ConsolidatedComponents), estimatedSize)
	return result, nil
}

func (m *Manager) isTenantCacheValid() bool {
	if m.cache.TenantDiscoveryCache == nil {
		return false
	}
	return time.Since(m.cache.TenantCacheLastUpdated) < m.tenantCacheTTL
}

func (m *Manager) isMimirCacheValid() bool {
	if m.cache.MimirDiscoveryCache == nil {
		return false
	}
	return time.Since(m.cache.MimirCacheLastUpdated) < m.mimirCacheTTL
}

func (m *Manager) getTenantDiscoveryCount() int {
	if m.cache.TenantDiscoveryCache == nil {
		return 0
	}
	return len(m.cache.TenantDiscoveryCache.ConsolidatedTenants)
}

func (m *Manager) getMimirDiscoveryCount() int {
	if m.cache.MimirDiscoveryCache == nil {
		return 0
	}
	return len(m.cache.MimirDiscoveryCache.ConsolidatedComponents)
}

// Memory management methods

// GetMemoryStats returns detailed memory statistics
func (m *Manager) GetMemoryStats() MemoryStats {
	return m.memoryManager.GetMemoryStats()
}

// SetMemoryLimits allows dynamic adjustment of memory limits
func (m *Manager) SetMemoryLimits(maxMemoryBytes int64, maxCacheSize, maxTenantSize, maxMimirSize int) {
	m.memoryManager.SetMemoryLimits(maxMemoryBytes, maxCacheSize, maxTenantSize, maxMimirSize)
}

// SetEvictionPolicy allows changing the eviction policy
func (m *Manager) SetEvictionPolicy(policy EvictionPolicy, threshold float64) {
	m.memoryManager.SetEvictionPolicy(policy, threshold)
}

// ForceMemoryEviction forces an immediate memory eviction cycle
func (m *Manager) ForceMemoryEviction() error {
	return m.memoryManager.ForceEviction()
}

// ResetMemoryStats resets memory statistics
func (m *Manager) ResetMemoryStats() {
	m.memoryManager.ResetStats()
}
