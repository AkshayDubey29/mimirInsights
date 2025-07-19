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
}

// Cache holds all cached data
type Cache struct {
	// Discovery data
	DiscoveryResult *discovery.DiscoveryResult `json:"discovery_result"`

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
	return &Manager{
		discoveryEngine:    discoveryEngine,
		metricsClient:      metricsClient,
		limitsAnalyzer:     limitsAnalyzer,
		cache:              &Cache{},
		collectionInterval: 30 * time.Second, // Collect every 30 seconds
		stopChan:           make(chan struct{}),
	}
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

	// Collect metrics for all tenants
	tenantMetrics := make(map[string]*TenantMetricsData)
	tenantNames := make([]string, 0, len(discoveryResult.TenantNamespaces))

	logrus.Infof("📊 [CACHE] Collecting metrics for %d tenants...", len(discoveryResult.TenantNamespaces))
	for _, tenant := range discoveryResult.TenantNamespaces {
		tenantNames = append(tenantNames, tenant.Name)
		logrus.Debugf("📈 [CACHE] Collecting metrics for tenant: %s", tenant.Name)
		metricsData, err := m.collectTenantMetrics(ctx, tenant.Name)
		if err != nil {
			logrus.Warnf("⚠️ [CACHE] Failed to collect metrics for tenant %s: %v", tenant.Name, err)
			metricsData = &TenantMetricsData{
				CollectionErrors: []string{err.Error()},
				LastUpdated:      time.Now(),
			}
		} else {
			logrus.Debugf("✅ [CACHE] Successfully collected metrics for tenant: %s", tenant.Name)
		}
		tenantMetrics[tenant.Name] = metricsData
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

	return map[string]interface{}{
		"last_updated":        m.cache.LastUpdated,
		"last_collection":     m.cache.LastCollection,
		"collection_count":    m.cache.CollectionCount,
		"collection_interval": m.collectionInterval.String(),
		"is_running":          m.isRunning,
		"tenant_count":        len(m.cache.TenantMetrics),
		"mimir_components":    len(m.cache.DiscoveryResult.MimirComponents),
	}
}

// ForceRefresh triggers an immediate data collection
func (m *Manager) ForceRefresh(ctx context.Context) error {
	logrus.Info("Force refreshing cache data")
	return m.collectAllData(ctx)
}
