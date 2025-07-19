package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/akshaydubey29/mimirInsights/pkg/cache"
	"github.com/akshaydubey29/mimirInsights/pkg/capacity"
	"github.com/akshaydubey29/mimirInsights/pkg/discovery"
	"github.com/akshaydubey29/mimirInsights/pkg/drift"
	"github.com/akshaydubey29/mimirInsights/pkg/limits"
	"github.com/akshaydubey29/mimirInsights/pkg/llm"
	"github.com/akshaydubey29/mimirInsights/pkg/metrics"
	"github.com/akshaydubey29/mimirInsights/pkg/monitoring"
	"github.com/akshaydubey29/mimirInsights/pkg/tuning"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Server handles all API requests
type Server struct {
	discoveryEngine *discovery.Engine
	metricsClient   *metrics.Client
	limitsAnalyzer  *limits.Analyzer
	cacheManager    *cache.Manager
	driftDetector   *drift.Detector
	alloyTuner      *tuning.AlloyTuner
	capacityPlanner *capacity.Planner
	llmAssistant    *llm.Assistant
	healthChecker   *monitoring.HealthChecker

	// Prometheus metrics
	requestCounter  *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	errorCounter    *prometheus.CounterVec
}

// NewServer creates a new API server
func NewServer(discoveryEngine *discovery.Engine, metricsClient *metrics.Client, limitsAnalyzer *limits.Analyzer) *Server {
	// Prometheus metrics
	requestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mimir_insights_api_requests_total",
			Help: "Total number of API requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "mimir_insights_api_request_duration_seconds",
			Help: "Request duration in seconds",
		},
		[]string{"method", "endpoint"},
	)

	errorCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mimir_insights_api_errors_total",
			Help: "Total number of API errors",
		},
		[]string{"type"},
	)

	prometheus.MustRegister(requestCounter, requestDuration, errorCounter)

	// Health monitoring configuration
	healthConfig := monitoring.HealthConfig{
		CheckInterval:  30 * time.Second,
		CheckTimeout:   10 * time.Second,
		RetryAttempts:  3,
		AlertThreshold: 2,
		EnableAlerting: false, // Disable by default
	}

	// Create cache manager
	cacheManager := cache.NewManager(discoveryEngine, metricsClient, limitsAnalyzer)

	server := &Server{
		discoveryEngine: discoveryEngine,
		metricsClient:   metricsClient,
		limitsAnalyzer:  limitsAnalyzer,
		cacheManager:    cacheManager,
		driftDetector:   drift.NewDetector(discoveryEngine.GetK8sClient()),
		alloyTuner:      tuning.NewAlloyTuner(discoveryEngine.GetK8sClient()),
		capacityPlanner: capacity.NewPlanner(metricsClient, limitsAnalyzer),
		llmAssistant:    llm.NewAssistant(discoveryEngine.GetConfig(), metricsClient),
		healthChecker:   monitoring.NewHealthChecker(discoveryEngine.GetK8sClient(), healthConfig),
		requestCounter:  requestCounter,
		requestDuration: requestDuration,
		errorCounter:    errorCounter,
	}

	// Start cache manager in background
	go func() {
		if err := cacheManager.Start(context.Background()); err != nil {
			logrus.Errorf("Failed to start cache manager: %v", err)
		}
	}()

	return server
}

// HealthCheck returns the health status of the service
func (s *Server) HealthCheck(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	statusCode, response := s.healthChecker.GetHealthStatus(ctx)

	s.recordMetrics(c, statusCode, start)
	c.JSON(statusCode, response)
}

// GetTenants returns all discovered tenant namespaces
func (s *Server) GetTenants(c *gin.Context) {
	start := time.Now()

	logrus.Infof("🔍 [API] GetTenants called from %s", c.ClientIP())
	logrus.Infof("📋 [API] GetTenants: Starting tenant discovery process")

	// Get cached discovery data
	discoveryResult := s.cacheManager.GetDiscoveryResult()
	if discoveryResult == nil {
		logrus.Warnf("❌ [API] GetTenants: Cache not ready - no discovery result available")
		s.recordError(c, "cache_not_ready", start)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Cache not ready, please try again"})
		return
	}

	logrus.Infof("✅ [API] GetTenants: Cache ready, found %d tenants in cache", len(discoveryResult.TenantNamespaces))
	logrus.Infof("📊 [API] GetTenants: Discovery details - Mimir components: %d, ConfigMaps: %d",
		len(discoveryResult.MimirComponents), len(discoveryResult.ConfigMaps))

	// Log environment information
	if discoveryResult.Environment != nil {
		logrus.Infof("🌍 [API] GetTenants: Environment - Namespace: %s, Total namespaces: %d, Is production: %v",
			discoveryResult.Environment.MimirNamespace,
			discoveryResult.Environment.TotalNamespaces,
			discoveryResult.Environment.IsProduction)
	}

	// Extract tenant information with detailed logging
	tenants := make([]map[string]interface{}, 0, len(discoveryResult.TenantNamespaces))
	for i, tenant := range discoveryResult.TenantNamespaces {
		logrus.Infof("📋 [API] GetTenants: Processing tenant %d/%d: %s", i+1, len(discoveryResult.TenantNamespaces), tenant.Name)

		tenantInfo := map[string]interface{}{
			"name":            tenant.Name,
			"status":          tenant.Status,
			"component_count": tenant.ComponentCount,
			"labels":          tenant.Labels,
		}
		tenants = append(tenants, tenantInfo)

		logrus.Debugf("📋 [API] GetTenants: Tenant %s (status: %s, components: %d, labels: %v)",
			tenant.Name, tenant.Status, tenant.ComponentCount, tenant.Labels)
	}

	response := map[string]interface{}{
		"tenants":      tenants,
		"total_count":  len(tenants),
		"last_updated": discoveryResult.LastUpdated,
		"discovery_info": map[string]interface{}{
			"mimir_components_found": len(discoveryResult.MimirComponents),
			"config_maps_found":      len(discoveryResult.ConfigMaps),
			"environment":            discoveryResult.Environment,
		},
	}

	logrus.Infof("✅ [API] GetTenants: Returning %d tenants, response time: %v", len(tenants), time.Since(start))
	logrus.Infof("📊 [API] GetTenants: Response payload size: %d bytes", len(fmt.Sprintf("%+v", response)))

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, response)
}

// CreateTenant creates a new tenant namespace with required configurations
func (s *Server) CreateTenant(c *gin.Context) {
	start := time.Now()

	var request struct {
		Name        string `json:"name" binding:"required"`
		Namespace   string `json:"namespace" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		s.recordError(c, "validation_error", start)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// For demo purposes, create a basic tenant configuration
	// In a real implementation, this would create the actual Kubernetes namespace
	// and configure Alloy, NGINX, and other components
	tenantConfig := map[string]interface{}{
		"name":        request.Name,
		"namespace":   request.Namespace,
		"description": request.Description,
		"created_at":  time.Now().UTC(),
		"status":      "active",
		"components": map[string]interface{}{
			"alloy": map[string]interface{}{
				"enabled":  true,
				"replicas": 1,
				"image":    "grafana/alloy:latest",
			},
			"nginx": map[string]interface{}{
				"enabled":  true,
				"replicas": 1,
				"image":    "nginx:alpine",
			},
		},
		"limits": map[string]interface{}{
			"max_series":     100000,
			"ingestion_rate": "10000/s",
			"query_timeout":  "300s",
		},
	}

	response := map[string]interface{}{
		"message": "Tenant created successfully",
		"tenant":  tenantConfig,
		"note":    "This is a demo implementation. In production, this would create actual Kubernetes resources.",
	}

	s.recordMetrics(c, http.StatusCreated, start)
	c.JSON(http.StatusCreated, response)
}

// GetLimits returns limit recommendations for all tenants
func (s *Server) GetLimits(c *gin.Context) {
	start := time.Now()

	logrus.Infof("🔍 [API] GetLimits called from %s", c.ClientIP())
	logrus.Infof("📋 [API] GetLimits: Starting limits analysis process")

	// Get tenant names from query parameter or get all cached
	tenantParam := c.Query("tenant")
	var limitsSummary map[string]*limits.TenantLimits

	if tenantParam != "" {
		logrus.Infof("📋 [API] GetLimits: Requesting limits for specific tenant: %s", tenantParam)
		// Get specific tenant limits
		tenantLimits := s.cacheManager.GetTenantLimitsSummary(tenantParam)
		if tenantLimits == nil {
			logrus.Warnf("❌ [API] GetLimits: Tenant %s not found in cache", tenantParam)
			s.recordError(c, "tenant_not_found", start)
			c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
			return
		}
		limitsSummary = map[string]*limits.TenantLimits{tenantParam: tenantLimits}
		logrus.Infof("✅ [API] GetLimits: Found limits for tenant %s", tenantParam)
		logrus.Debugf("📊 [API] GetLimits: Tenant %s limits - %+v", tenantParam, tenantLimits)
	} else {
		logrus.Infof("📋 [API] GetLimits: Requesting limits for all tenants")
		// Get all cached limits
		limitsSummary = s.cacheManager.GetLimitsSummary()
		if limitsSummary == nil {
			logrus.Warnf("❌ [API] GetLimits: Cache not ready - no limits data available")
			s.recordError(c, "cache_not_ready", start)
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Cache not ready, please try again"})
			return
		}
		logrus.Infof("✅ [API] GetLimits: Found limits for %d tenants", len(limitsSummary))

		// Log details about each tenant's limits
		for tenantName, tenantLimits := range limitsSummary {
			logrus.Debugf("📊 [API] GetLimits: Tenant %s limits - %+v", tenantName, tenantLimits)
		}
	}

	// Get auto-discovered limits for additional context
	autoDiscoveredLimits := s.cacheManager.GetAutoDiscoveredLimits()
	logrus.Infof("🔍 [API] GetLimits: Auto-discovered limits - Global: %d, Tenant: %d, Sources: %d",
		len(autoDiscoveredLimits.GlobalLimits),
		len(autoDiscoveredLimits.TenantLimits),
		len(autoDiscoveredLimits.ConfigSources))

	response := map[string]interface{}{
		"limits":        limitsSummary,
		"total_tenants": len(limitsSummary),
		"timestamp":     time.Now().UTC(),
		"auto_discovered": map[string]interface{}{
			"global_limits_count":  len(autoDiscoveredLimits.GlobalLimits),
			"tenant_limits_count":  len(autoDiscoveredLimits.TenantLimits),
			"config_sources_count": len(autoDiscoveredLimits.ConfigSources),
			"last_updated":         autoDiscoveredLimits.LastUpdated,
		},
	}

	logrus.Infof("✅ [API] GetLimits: Returning limits for %d tenants, response time: %v", len(limitsSummary), time.Since(start))
	logrus.Infof("📊 [API] GetLimits: Response payload size: %d bytes", len(fmt.Sprintf("%+v", response)))

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, response)
}

// GetConfig returns configuration information
func (s *Server) GetConfig(c *gin.Context) {
	start := time.Now()

	// Get cached discovery data
	discoveryResult := s.cacheManager.GetDiscoveryResult()
	if discoveryResult == nil {
		s.recordError(c, "cache_not_ready", start)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Cache not ready, please try again"})
		return
	}

	// Build configuration response
	config := map[string]interface{}{
		"mimir_components":  discoveryResult.MimirComponents,
		"tenant_namespaces": discoveryResult.TenantNamespaces,
		"config_maps":       discoveryResult.ConfigMaps,
		"environment":       discoveryResult.Environment,
		"last_updated":      discoveryResult.LastUpdated,
	}

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, config)
}

// GetEnvironment returns comprehensive environment detection information
func (s *Server) GetEnvironment(c *gin.Context) {
	start := time.Now()

	// Get cached environment data
	environment := s.cacheManager.GetEnvironment()
	if environment == nil {
		s.recordError(c, "cache_not_ready", start)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Cache not ready, please try again"})
		return
	}

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, environment)
}

// GetMetrics returns Prometheus metrics
func (s *Server) GetMetrics(c *gin.Context) {
	promhttp.Handler().ServeHTTP(c.Writer, c.Request)
}

// GetAuditLogs returns audit logs (placeholder)
func (s *Server) GetAuditLogs(c *gin.Context) {
	start := time.Now()

	// TODO: Implement actual audit log retrieval
	auditLogs := []map[string]interface{}{
		{
			"timestamp":   time.Now().Add(-1 * time.Hour),
			"action":      "limit_analysis",
			"tenant":      "example-tenant",
			"user":        "system",
			"description": "Analyzed limits for tenant",
		},
	}

	response := map[string]interface{}{
		"audit_logs": auditLogs,
		"total":      len(auditLogs),
	}

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, response)
}

// AnalyzeTenant analyzes a specific tenant
func (s *Server) AnalyzeTenant(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	var request struct {
		TenantName string `json:"tenant_name" binding:"required"`
		TimeRange  string `json:"time_range"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		s.recordError(c, "validation_error", start)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Determine time range
	var timeRange metrics.TimeRange
	switch request.TimeRange {
	case "48h":
		timeRange = metrics.CreateTimeRange(48*time.Hour, "5m")
	case "7d":
		timeRange = metrics.CreateTimeRange(7*24*time.Hour, "15m")
	case "30d":
		timeRange = metrics.CreateTimeRange(30*24*time.Hour, "1h")
	case "60d":
		timeRange = metrics.CreateTimeRange(60*24*time.Hour, "2h")
	default:
		timeRange = metrics.CreateTimeRange(7*24*time.Hour, "15m") // Default to 7d
	}

	// Get tenant metrics
	tenantMetrics, err := s.metricsClient.GetTenantMetrics(ctx, request.TenantName, timeRange)
	if err != nil {
		s.recordError(c, "metrics_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Analyze limits
	tenantLimits, err := s.limitsAnalyzer.AnalyzeTenantLimits(ctx, request.TenantName)
	if err != nil {
		s.recordError(c, "limits_analysis_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := map[string]interface{}{
		"tenant_name":   request.TenantName,
		"metrics":       tenantMetrics,
		"limits":        tenantLimits,
		"analysis_time": time.Now().UTC(),
	}

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, response)
}

// GetDriftStatus returns configuration drift status
func (s *Server) GetDriftStatus(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	// Get target namespaces for drift detection
	namespaceParam := c.Query("namespace")
	var namespaces []string

	if namespaceParam != "" {
		namespaces = []string{namespaceParam}
	} else {
		// Discover all tenant namespaces
		result, err := s.discoveryEngine.DiscoverAll(ctx)
		if err != nil {
			s.recordError(c, "discovery_error", start)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Include Mimir namespace and tenant namespaces
		namespaces = append(namespaces, result.Environment.MimirNamespace)
		for _, tenant := range result.TenantNamespaces {
			namespaces = append(namespaces, tenant.Name)
		}
	}

	// Perform drift detection
	driftReport, err := s.driftDetector.DetectDrift(ctx, namespaces)
	if err != nil {
		s.recordError(c, "drift_detection_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Build enhanced drift response
	response := map[string]interface{}{
		"status":          "completed",
		"last_check":      driftReport.GeneratedAt,
		"total_resources": driftReport.TotalResources,
		"drift_count":     driftReport.DriftedCount,
		"new_count":       driftReport.NewCount,
		"deleted_count":   driftReport.DeletedCount,
		"summary":         driftReport.Summary,
		"drift_details":   driftReport.DriftStatuses,
		"risk_assessment": map[string]interface{}{
			"low_risk":      driftReport.Summary.LowRisk,
			"medium_risk":   driftReport.Summary.MediumRisk,
			"high_risk":     driftReport.Summary.HighRisk,
			"critical_risk": driftReport.Summary.CriticalRisk,
		},
		"recommendations": s.generateDriftRecommendations(driftReport),
	}

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, response)
}

// generateDriftRecommendations generates recommendations based on drift analysis
func (s *Server) generateDriftRecommendations(report *drift.DriftReport) []string {
	var recommendations []string

	if report.Summary.CriticalRisk > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("⚠️ CRITICAL: %d resources have critical configuration drift - immediate review required", report.Summary.CriticalRisk))
	}

	if report.Summary.HighRisk > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("🔴 HIGH: %d resources have high-impact configuration changes", report.Summary.HighRisk))
	}

	if report.NewCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("📦 %d new configuration resources detected - review and establish baselines", report.NewCount))
	}

	if report.DeletedCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("🗑️ %d configuration resources were deleted - verify if intentional", report.DeletedCount))
	}

	if report.DriftedCount == 0 && report.NewCount == 0 && report.DeletedCount == 0 {
		recommendations = append(recommendations, "✅ All monitored configurations are in sync with baselines")
	}

	if report.Summary.MediumRisk > 0 {
		recommendations = append(recommendations, "📋 Schedule regular configuration audits to prevent future drift")
	}

	return recommendations
}

// CreateDriftBaseline creates a baseline for current configurations
func (s *Server) CreateDriftBaseline(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	// Get target namespaces
	var request struct {
		Namespaces []string `json:"namespaces"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		// If no request body, use discovered namespaces
		result, err := s.discoveryEngine.DiscoverAll(ctx)
		if err != nil {
			s.recordError(c, "discovery_error", start)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		request.Namespaces = append(request.Namespaces, result.Environment.MimirNamespace)
		for _, tenant := range result.TenantNamespaces {
			request.Namespaces = append(request.Namespaces, tenant.Name)
		}
	}

	// Create baseline
	err := s.driftDetector.CreateBaseline(ctx, request.Namespaces)
	if err != nil {
		s.recordError(c, "baseline_creation_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := map[string]interface{}{
		"message":    "Configuration baseline created successfully",
		"namespaces": request.Namespaces,
		"created_at": time.Now().UTC(),
	}

	s.recordMetrics(c, http.StatusCreated, start)
	c.JSON(http.StatusCreated, response)
}

// GetCapacityReport generates a comprehensive capacity planning report
func (s *Server) GetCapacityReport(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	// Get report parameters
	reportType := c.DefaultQuery("type", "weekly")
	format := c.DefaultQuery("format", "json")
	namespaceParam := c.Query("namespace")

	// Validate report type
	validTypes := map[string]bool{
		"weekly": true, "monthly": true, "quarterly": true,
	}
	if !validTypes[reportType] {
		s.recordError(c, "invalid_report_type", start)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report type. Use 'weekly', 'monthly', or 'quarterly'"})
		return
	}

	// Get target tenants
	var tenantNames []string
	if namespaceParam != "" {
		tenantNames = []string{namespaceParam}
	} else {
		// Discover all tenant namespaces
		result, err := s.discoveryEngine.DiscoverAll(ctx)
		if err != nil {
			s.recordError(c, "discovery_error", start)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for _, tenant := range result.TenantNamespaces {
			tenantNames = append(tenantNames, tenant.Name)
		}
	}

	if len(tenantNames) == 0 {
		s.recordError(c, "no_tenants_found", start)
		c.JSON(http.StatusNotFound, gin.H{"error": "No tenants found for capacity analysis"})
		return
	}

	// Generate capacity report
	report, err := s.capacityPlanner.GenerateCapacityReport(ctx, reportType, tenantNames)
	if err != nil {
		s.recordError(c, "capacity_report_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Handle different response formats
	switch format {
	case "json":
		s.recordMetrics(c, http.StatusOK, start)
		c.JSON(http.StatusOK, report)
	case "download":
		// Return downloadable format info
		downloadFormats := []string{"json", "csv", "html", "pdf"}
		response := map[string]interface{}{
			"report_id":         report.ReportID,
			"available_formats": downloadFormats,
			"download_endpoint": "/api/capacity/export",
		}
		s.recordMetrics(c, http.StatusOK, start)
		c.JSON(http.StatusOK, response)
	default:
		s.recordError(c, "invalid_format", start)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid format. Use 'json' or 'download'"})
		return
	}
}

// ExportCapacityReport exports capacity report in specified format
func (s *Server) ExportCapacityReport(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	// Get export parameters
	reportType := c.DefaultQuery("type", "weekly")
	format := c.DefaultQuery("format", "json")
	namespaceParam := c.Query("namespace")

	// Validate export format
	validFormats := map[string]capacity.ExportFormat{
		"json": capacity.ExportFormatJSON,
		"csv":  capacity.ExportFormatCSV,
		"html": capacity.ExportFormatHTML,
		"pdf":  capacity.ExportFormatPDF,
	}

	exportFormat, valid := validFormats[format]
	if !valid {
		s.recordError(c, "invalid_export_format", start)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid export format. Use 'json', 'csv', 'html', or 'pdf'"})
		return
	}

	// Get target tenants
	var tenantNames []string
	if namespaceParam != "" {
		tenantNames = []string{namespaceParam}
	} else {
		// Discover all tenant namespaces
		result, err := s.discoveryEngine.DiscoverAll(ctx)
		if err != nil {
			s.recordError(c, "discovery_error", start)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for _, tenant := range result.TenantNamespaces {
			tenantNames = append(tenantNames, tenant.Name)
		}
	}

	// Generate capacity report
	report, err := s.capacityPlanner.GenerateCapacityReport(ctx, reportType, tenantNames)
	if err != nil {
		s.recordError(c, "capacity_report_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Export report
	exporter := capacity.NewExporter()
	data, filename, err := exporter.ExportReport(report, exportFormat)
	if err != nil {
		s.recordError(c, "export_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Set appropriate content type and headers
	contentTypes := map[capacity.ExportFormat]string{
		capacity.ExportFormatJSON: "application/json",
		capacity.ExportFormatCSV:  "text/csv",
		capacity.ExportFormatHTML: "text/html",
		capacity.ExportFormatPDF:  "application/pdf",
	}

	c.Header("Content-Type", contentTypes[exportFormat])
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(data)))

	s.recordMetrics(c, http.StatusOK, start)
	c.Data(http.StatusOK, contentTypes[exportFormat], data)
}

// GetCapacityTrends returns capacity trends analysis
func (s *Server) GetCapacityTrends(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	// Get parameters
	days := c.DefaultQuery("days", "30")
	namespaceParam := c.Query("namespace")

	// Get target tenants
	var tenantNames []string
	if namespaceParam != "" {
		tenantNames = []string{namespaceParam}
	} else {
		// Discover all tenant namespaces
		result, err := s.discoveryEngine.DiscoverAll(ctx)
		if err != nil {
			s.recordError(c, "discovery_error", start)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for _, tenant := range result.TenantNamespaces {
			tenantNames = append(tenantNames, tenant.Name)
		}
	}

	// Generate trend analysis
	response := map[string]interface{}{
		"period":       fmt.Sprintf("Last %s days", days),
		"tenants":      len(tenantNames),
		"trends":       s.generateTrendSummary(ctx, tenantNames),
		"last_updated": time.Now().UTC(),
	}

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, response)
}

// generateTrendSummary generates a summary of capacity trends
func (s *Server) generateTrendSummary(ctx context.Context, tenantNames []string) map[string]interface{} {
	// Simplified trend summary
	return map[string]interface{}{
		"overall_trend":   "increasing",
		"growth_rate":     0.12,
		"risk_level":      "medium",
		"tenants_at_risk": len(tenantNames) / 4, // 25% at risk
		"recommendations": []string{
			"Monitor high-growth tenants",
			"Consider proactive scaling",
			"Review resource allocation",
		},
	}
}

// recordMetrics records Prometheus metrics for the request
func (s *Server) recordMetrics(c *gin.Context, statusCode int, start time.Time) {
	method := c.Request.Method
	endpoint := c.FullPath()
	if endpoint == "" {
		endpoint = c.Request.URL.Path
	}

	duration := time.Since(start).Seconds()

	s.requestCounter.WithLabelValues(method, endpoint, string(rune(statusCode))).Inc()
	s.requestDuration.WithLabelValues(method, endpoint).Observe(duration)
}

// recordError records an error and increments error counter
func (s *Server) recordError(c *gin.Context, errorType string, start time.Time) {
	logrus.Errorf("API error: %s", errorType)

	s.requestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(time.Since(start).Seconds())
	s.errorCounter.WithLabelValues(errorType).Inc()
}

// CORSMiddleware adds CORS headers
func CORSMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
}

// GetAutoDiscoveredMetrics handles GET /api/metrics/discovery
func (s *Server) GetAutoDiscoveredMetrics(c *gin.Context) {
	start := time.Now()

	ctx := c.Request.Context()

	// Get auto-discovered metrics endpoints
	endpoints, err := s.discoveryEngine.GetAutoDiscoveredMetrics(ctx)
	if err != nil {
		s.recordError(c, "metrics_discovery_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to discover metrics endpoints: %v", err)})
		return
	}

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, gin.H{
		"endpoints": endpoints,
		"count":     len(endpoints),
		"timestamp": time.Now().UTC(),
	})
}

// GetAlloyDeployments returns all Alloy deployments across tenant namespaces
func (s *Server) GetAlloyDeployments(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	// Get target namespaces
	namespaceParam := c.Query("namespace")
	var namespaces []string

	if namespaceParam != "" {
		namespaces = []string{namespaceParam}
	} else {
		// Discover all tenant namespaces
		result, err := s.discoveryEngine.DiscoverAll(ctx)
		if err != nil {
			s.recordError(c, "discovery_error", start)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for _, tenant := range result.TenantNamespaces {
			namespaces = append(namespaces, tenant.Name)
		}
	}

	// Get Alloy deployments
	alloyDeployments, err := s.alloyTuner.GetAlloyDeployments(ctx, namespaces)
	if err != nil {
		s.recordError(c, "alloy_discovery_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get scaling recommendations
	recommendations, err := s.alloyTuner.GetAlloyScalingRecommendations(ctx, namespaces)
	if err != nil {
		logrus.Warnf("Failed to get scaling recommendations: %v", err)
		recommendations = []tuning.AlloyScalingRecommendation{}
	}

	response := map[string]interface{}{
		"deployments":     alloyDeployments,
		"recommendations": recommendations,
		"total_count":     len(alloyDeployments),
		"last_updated":    time.Now().UTC(),
	}

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, response)
}

// GetAlloyWorkloads returns all Alloy workloads (Deployments, StatefulSets, DaemonSets) across tenant namespaces
func (s *Server) GetAlloyWorkloads(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	// Get target namespaces
	namespaceParam := c.Query("namespace")
	var namespaces []string

	if namespaceParam != "" {
		namespaces = []string{namespaceParam}
	} else {
		// Discover all tenant namespaces
		result, err := s.discoveryEngine.DiscoverAll(ctx)
		if err != nil {
			s.recordError(c, "discovery_error", start)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for _, tenant := range result.TenantNamespaces {
			namespaces = append(namespaces, tenant.Name)
		}
	}

	// Get all Alloy workloads (Deployments, StatefulSets, DaemonSets)
	alloyWorkloads, err := s.alloyTuner.GetAlloyWorkloads(ctx, namespaces)
	if err != nil {
		s.recordError(c, "alloy_workload_discovery_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Organize workloads by type
	workloadsByType := make(map[string][]map[string]interface{})
	for _, workload := range alloyWorkloads {
		workloadType := workload["type"].(string)
		if workloadsByType[workloadType] == nil {
			workloadsByType[workloadType] = []map[string]interface{}{}
		}
		workloadsByType[workloadType] = append(workloadsByType[workloadType], workload)
	}

	// Get scaling recommendations for deployments and statefulsets
	recommendations, err := s.alloyTuner.GetAlloyScalingRecommendations(ctx, namespaces)
	if err != nil {
		logrus.Warnf("Failed to get scaling recommendations: %v", err)
		recommendations = []tuning.AlloyScalingRecommendation{}
	}

	response := map[string]interface{}{
		"workloads":         alloyWorkloads,
		"workloads_by_type": workloadsByType,
		"type_counts": map[string]int{
			"deployments":  len(workloadsByType["Deployment"]),
			"statefulsets": len(workloadsByType["StatefulSet"]),
			"daemonsets":   len(workloadsByType["DaemonSet"]),
		},
		"recommendations": recommendations,
		"total_count":     len(alloyWorkloads),
		"last_updated":    time.Now().UTC(),
	}

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, response)
}

// ScaleAlloyReplicas scales Alloy deployment replicas
func (s *Server) ScaleAlloyReplicas(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	var request tuning.AlloyReplicaRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		s.recordError(c, "validation_error", start)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Perform scaling operation
	response, err := s.alloyTuner.ScaleAlloyReplicas(ctx, request)
	if err != nil {
		s.recordError(c, "scaling_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, response)
}

// GetAlloyScalingRecommendations returns scaling recommendations for Alloy deployments
func (s *Server) GetAlloyScalingRecommendations(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	// Get target namespaces
	namespaceParam := c.Query("namespace")
	var namespaces []string

	if namespaceParam != "" {
		namespaces = []string{namespaceParam}
	} else {
		// Discover all tenant namespaces
		result, err := s.discoveryEngine.DiscoverAll(ctx)
		if err != nil {
			s.recordError(c, "discovery_error", start)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for _, tenant := range result.TenantNamespaces {
			namespaces = append(namespaces, tenant.Name)
		}
	}

	// Get scaling recommendations
	recommendations, err := s.alloyTuner.GetAlloyScalingRecommendations(ctx, namespaces)
	if err != nil {
		s.recordError(c, "scaling_analysis_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Categorize recommendations by priority
	priorityCount := map[string]int{
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
	}

	for _, rec := range recommendations {
		priorityCount[rec.Priority]++
	}

	response := map[string]interface{}{
		"recommendations": recommendations,
		"summary": map[string]interface{}{
			"total_recommendations": len(recommendations),
			"priority_breakdown":    priorityCount,
		},
		"last_updated": time.Now().UTC(),
	}

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, response)
}

// ProcessLLMQuery processes a natural language query about metrics
func (s *Server) ProcessLLMQuery(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	var request llm.QueryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		s.recordError(c, "validation_error", start)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default values
	if request.MaxTokens == 0 {
		request.MaxTokens = 500
	}
	if request.Temperature == 0 {
		request.Temperature = 0.7
	}

	// Process the query
	response, err := s.llmAssistant.ProcessQuery(ctx, request)
	if err != nil {
		s.recordError(c, "llm_processing_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, response)
}

// GetLLMCapabilities returns the capabilities of the LLM assistant
func (s *Server) GetLLMCapabilities(c *gin.Context) {
	start := time.Now()

	capabilities := map[string]interface{}{
		"features": []string{
			"limit_analysis",
			"capacity_planning",
			"drift_detection",
			"tenant_optimization",
			"configuration_audit",
		},
		"supported_models": []string{"gpt-4", "gpt-3.5-turbo"},
		"max_tokens":       4000,
		"temperature":      0.1,
	}

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, capabilities)
}

// GetCacheStatus returns the status of the cache manager
func (s *Server) GetCacheStatus(c *gin.Context) {
	start := time.Now()

	status := s.cacheManager.GetCacheStatus()

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, status)
}

// ForceCacheRefresh triggers an immediate cache refresh
func (s *Server) ForceCacheRefresh(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	if err := s.cacheManager.ForceRefresh(ctx); err != nil {
		s.recordError(c, "cache_refresh_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := map[string]interface{}{
		"message":   "Cache refresh triggered successfully",
		"timestamp": time.Now().UTC(),
	}

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, response)
}

// GetComprehensiveTenantDiscovery returns comprehensive tenant discovery using multiple strategies
func (s *Server) GetComprehensiveTenantDiscovery(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	logrus.Infof("🔍 [API] GetComprehensiveTenantDiscovery called from %s", c.ClientIP())
	logrus.Infof("📋 [API] GetComprehensiveTenantDiscovery: Starting multi-strategy tenant discovery")

	// Get cached tenant discovery results
	result, err := s.cacheManager.GetTenantDiscovery(ctx)
	if err != nil {
		logrus.Errorf("❌ [API] GetComprehensiveTenantDiscovery: Failed to get tenant discovery: %v", err)
		s.recordError(c, "discovery_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logrus.Infof("✅ [API] GetComprehensiveTenantDiscovery: Tenant discovery served from cache")
	logrus.Infof("📊 [API] GetComprehensiveTenantDiscovery: Found %d tenants using %d strategies",
		len(result.ConsolidatedTenants), len(result.Strategies))

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, result)
}

// GetComprehensiveMimirDiscovery returns comprehensive Mimir discovery using multiple strategies
func (s *Server) GetComprehensiveMimirDiscovery(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	logrus.Infof("🔍 [API] GetComprehensiveMimirDiscovery called from %s", c.ClientIP())
	logrus.Infof("📋 [API] GetComprehensiveMimirDiscovery: Starting multi-strategy Mimir discovery")

	// Get cached Mimir discovery results
	result, err := s.cacheManager.GetMimirDiscovery(ctx)
	if err != nil {
		logrus.Errorf("❌ [API] GetComprehensiveMimirDiscovery: Failed to get Mimir discovery: %v", err)
		s.recordError(c, "mimir_discovery_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logrus.Infof("✅ [API] GetComprehensiveMimirDiscovery: Mimir discovery served from cache")
	logrus.Infof("📊 [API] GetComprehensiveMimirDiscovery: Found %d Mimir components using %d strategies",
		len(result.ConsolidatedComponents), len(result.Strategies))

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, result)
}

// GetDiscoveryDetails returns comprehensive discovery information
func (s *Server) GetDiscoveryDetails(c *gin.Context) {
	start := time.Now()

	logrus.Infof("🔍 [API] GetDiscoveryDetails called from %s", c.ClientIP())
	logrus.Infof("📋 [API] GetDiscoveryDetails: Starting comprehensive discovery analysis")

	// Get cached discovery data
	discoveryResult := s.cacheManager.GetDiscoveryResult()
	if discoveryResult == nil {
		logrus.Warnf("❌ [API] GetDiscoveryDetails: Cache not ready - no discovery result available")
		s.recordError(c, "cache_not_ready", start)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Cache not ready, please try again"})
		return
	}

	logrus.Infof("✅ [API] GetDiscoveryDetails: Cache ready, analyzing discovery results")

	// Analyze Mimir components
	mimirComponents := discoveryResult.MimirComponents
	logrus.Infof("🔍 [API] GetDiscoveryDetails: Found %d Mimir components", len(mimirComponents))

	componentAnalysis := make(map[string]interface{})
	for i, component := range mimirComponents {
		logrus.Infof("📋 [API] GetDiscoveryDetails: Component %d/%d: %s (type: %s, namespace: %s, confidence: %.1f%%)",
			i+1, len(mimirComponents), component.Name, component.Type, component.Namespace, component.Validation.ConfidenceScore)

		componentAnalysis[component.Name] = map[string]interface{}{
			"type":              component.Type,
			"namespace":         component.Namespace,
			"status":            component.Status,
			"replicas":          component.Replicas,
			"image":             component.Image,
			"version":           component.Version,
			"confidence_score":  component.Validation.ConfidenceScore,
			"matched_by":        component.Validation.MatchedBy,
			"validation_info":   component.Validation.ValidationInfo,
			"metrics_endpoints": component.MetricsEndpoints,
			"service_endpoints": component.ServiceEndpoints,
		}
	}

	// Analyze tenant namespaces
	tenantNamespaces := discoveryResult.TenantNamespaces
	logrus.Infof("🔍 [API] GetDiscoveryDetails: Found %d tenant namespaces", len(tenantNamespaces))

	tenantAnalysis := make(map[string]interface{})
	for i, tenant := range tenantNamespaces {
		logrus.Infof("📋 [API] GetDiscoveryDetails: Tenant %d/%d: %s (status: %s, components: %d, confidence: %.1f%%)",
			i+1, len(tenantNamespaces), tenant.Name, tenant.Status, tenant.ComponentCount, tenant.Validation.ConfidenceScore)

		tenantAnalysis[tenant.Name] = map[string]interface{}{
			"status":           tenant.Status,
			"component_count":  tenant.ComponentCount,
			"confidence_score": tenant.Validation.ConfidenceScore,
			"matched_by":       tenant.Validation.MatchedBy,
			"labels":           tenant.Labels,
			"annotations":      tenant.Annotations,
			"alloy_config":     tenant.AlloyConfig,
			"consul_config":    tenant.ConsulConfig,
			"nginx_config":     tenant.NginxConfig,
			"mimir_limits":     tenant.MimirLimits,
		}
	}

	// Analyze ConfigMaps
	configMaps := discoveryResult.ConfigMaps
	logrus.Infof("🔍 [API] GetDiscoveryDetails: Found %d relevant ConfigMaps", len(configMaps))

	configMapAnalysis := make(map[string]interface{})
	for i, configMap := range configMaps {
		logrus.Infof("📋 [API] GetDiscoveryDetails: ConfigMap %d/%d: %s (namespace: %s, data keys: %d)",
			i+1, len(configMaps), configMap.Name, configMap.Namespace, len(configMap.Data))

		configMapAnalysis[configMap.Name] = map[string]interface{}{
			"namespace": configMap.Namespace,
			"data_keys": len(configMap.Data),
			"labels":    configMap.Labels,
		}
	}

	// Environment analysis
	environment := discoveryResult.Environment
	logrus.Infof("🌍 [API] GetDiscoveryDetails: Environment analysis - Namespace: %s, Total namespaces: %d, Is production: %v",
		environment.MimirNamespace, environment.TotalNamespaces, environment.IsProduction)

	response := map[string]interface{}{
		"discovery_summary": map[string]interface{}{
			"mimir_components_found":    len(mimirComponents),
			"tenant_namespaces_found":   len(tenantNamespaces),
			"config_maps_found":         len(configMaps),
			"last_updated":              discoveryResult.LastUpdated,
			"auto_discovered_namespace": discoveryResult.AutoDiscoveredNS,
		},
		"mimir_components": map[string]interface{}{
			"count":   len(mimirComponents),
			"details": componentAnalysis,
			"analysis": map[string]interface{}{
				"total_confidence": calculateAverageConfidence(mimirComponents),
				"component_types":  getComponentTypeDistribution(mimirComponents),
				"namespaces":       getNamespaceDistribution(mimirComponents),
			},
		},
		"tenant_namespaces": map[string]interface{}{
			"count":   len(tenantNamespaces),
			"details": tenantAnalysis,
			"analysis": map[string]interface{}{
				"total_confidence":    calculateAverageTenantConfidence(tenantNamespaces),
				"status_distribution": getTenantStatusDistribution(tenantNamespaces),
			},
		},
		"config_maps": map[string]interface{}{
			"count":   len(configMaps),
			"details": configMapAnalysis,
		},
		"environment":     environment,
		"recommendations": generateDiscoveryRecommendations(discoveryResult),
	}

	logrus.Infof("✅ [API] GetDiscoveryDetails: Returning comprehensive discovery analysis, response time: %v", time.Since(start))
	logrus.Infof("📊 [API] GetDiscoveryDetails: Response payload size: %d bytes", len(fmt.Sprintf("%+v", response)))

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, response)
}

// Helper functions for discovery analysis
func calculateAverageConfidence(components []discovery.MimirComponent) float64 {
	if len(components) == 0 {
		return 0.0
	}
	total := 0.0
	for _, component := range components {
		total += component.Validation.ConfidenceScore
	}
	return total / float64(len(components))
}

func calculateAverageTenantConfidence(tenants []discovery.TenantNamespace) float64 {
	if len(tenants) == 0 {
		return 0.0
	}
	total := 0.0
	for _, tenant := range tenants {
		total += tenant.Validation.ConfidenceScore
	}
	return total / float64(len(tenants))
}

func getComponentTypeDistribution(components []discovery.MimirComponent) map[string]int {
	distribution := make(map[string]int)
	for _, component := range components {
		distribution[component.Type]++
	}
	return distribution
}

func getNamespaceDistribution(components []discovery.MimirComponent) map[string]int {
	distribution := make(map[string]int)
	for _, component := range components {
		distribution[component.Namespace]++
	}
	return distribution
}

func getTenantStatusDistribution(tenants []discovery.TenantNamespace) map[string]int {
	distribution := make(map[string]int)
	for _, tenant := range tenants {
		distribution[tenant.Status]++
	}
	return distribution
}

func generateDiscoveryRecommendations(result *discovery.DiscoveryResult) []string {
	var recommendations []string

	// Check if Mimir components were found
	if len(result.MimirComponents) == 0 {
		recommendations = append(recommendations, "⚠️ No Mimir components found - check if Mimir is deployed in the cluster")
	} else {
		// Check component confidence scores
		lowConfidenceCount := 0
		for _, component := range result.MimirComponents {
			if component.Validation.ConfidenceScore < 50 {
				lowConfidenceCount++
			}
		}
		if lowConfidenceCount > 0 {
			recommendations = append(recommendations, fmt.Sprintf("⚠️ %d Mimir components have low confidence scores - review discovery patterns", lowConfidenceCount))
		}
	}

	// Check if tenant namespaces were found
	if len(result.TenantNamespaces) == 0 {
		recommendations = append(recommendations, "⚠️ No tenant namespaces found - check tenant discovery patterns")
	} else {
		// Check tenant confidence scores
		lowConfidenceCount := 0
		for _, tenant := range result.TenantNamespaces {
			if tenant.Validation.ConfidenceScore < 50 {
				lowConfidenceCount++
			}
		}
		if lowConfidenceCount > 0 {
			recommendations = append(recommendations, fmt.Sprintf("⚠️ %d tenant namespaces have low confidence scores - review tenant patterns", lowConfidenceCount))
		}
	}

	// Check environment configuration
	if result.Environment.MimirNamespace == "" {
		recommendations = append(recommendations, "⚠️ Mimir namespace not detected - check cluster configuration")
	}

	return recommendations
}

// GetMemoryStats returns detailed memory statistics
func (s *Server) GetMemoryStats(c *gin.Context) {
	start := time.Now()

	logrus.Infof("🔍 [API] GetMemoryStats called from %s", c.ClientIP())

	// Get memory statistics from cache manager
	memoryStats := s.cacheManager.GetMemoryStats()

	logrus.Infof("✅ [API] GetMemoryStats: Memory statistics retrieved successfully")

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, memoryStats)
}

// ForceMemoryEviction forces an immediate memory eviction cycle
func (s *Server) ForceMemoryEviction(c *gin.Context) {
	start := time.Now()

	logrus.Infof("🔍 [API] ForceMemoryEviction called from %s", c.ClientIP())
	logrus.Info("🔄 [API] ForceMemoryEviction: Forcing memory eviction")

	// Force memory eviction
	err := s.cacheManager.ForceMemoryEviction()
	if err != nil {
		logrus.Errorf("❌ [API] ForceMemoryEviction: Failed to force eviction: %v", err)
		s.recordError(c, "memory_eviction_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logrus.Infof("✅ [API] ForceMemoryEviction: Memory eviction completed successfully")

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, gin.H{
		"message":   "Memory eviction completed successfully",
		"timestamp": time.Now(),
	})
}

// ResetMemoryStats resets memory statistics
func (s *Server) ResetMemoryStats(c *gin.Context) {
	start := time.Now()

	logrus.Infof("🔍 [API] ResetMemoryStats called from %s", c.ClientIP())
	logrus.Info("🔄 [API] ResetMemoryStats: Resetting memory statistics")

	// Reset memory statistics
	s.cacheManager.ResetMemoryStats()

	logrus.Infof("✅ [API] ResetMemoryStats: Memory statistics reset successfully")

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, gin.H{
		"message":   "Memory statistics reset successfully",
		"timestamp": time.Now(),
	})
}
