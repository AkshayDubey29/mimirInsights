package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

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

	return &Server{
		discoveryEngine: discoveryEngine,
		metricsClient:   metricsClient,
		limitsAnalyzer:  limitsAnalyzer,
		driftDetector:   drift.NewDetector(discoveryEngine.GetK8sClient()),
		alloyTuner:      tuning.NewAlloyTuner(discoveryEngine.GetK8sClient()),
		capacityPlanner: capacity.NewPlanner(metricsClient, limitsAnalyzer),
		llmAssistant:    llm.NewAssistant(discoveryEngine.GetConfig(), metricsClient),
		healthChecker:   monitoring.NewHealthChecker(discoveryEngine.GetK8sClient(), healthConfig),
		requestCounter:  requestCounter,
		requestDuration: requestDuration,
		errorCounter:    errorCounter,
	}
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
	ctx := c.Request.Context()

	// Perform discovery
	result, err := s.discoveryEngine.DiscoverAll(ctx)
	if err != nil {
		s.recordError(c, "discovery_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Extract tenant information
	tenants := make([]map[string]interface{}, 0, len(result.TenantNamespaces))
	for _, tenant := range result.TenantNamespaces {
		tenantInfo := map[string]interface{}{
			"name":            tenant.Name,
			"status":          tenant.Status,
			"component_count": tenant.ComponentCount,
			"labels":          tenant.Labels,
		}
		tenants = append(tenants, tenantInfo)
	}

	response := map[string]interface{}{"tenants": tenants, "total_count": len(tenants), "last_updated": result.LastUpdated}

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
	ctx := c.Request.Context()

	// Get tenant names from query parameter or discover all
	tenantParam := c.Query("tenant")
	var tenantNames []string

	if tenantParam != "" {
		tenantNames = []string{tenantParam}
	} else {
		// Discover all tenants
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

	// Analyze limits for all tenants
	limitsSummary, err := s.limitsAnalyzer.GetTenantLimitsSummary(ctx, tenantNames)
	if err != nil {
		s.recordError(c, "limits_analysis_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := map[string]interface{}{"limits": limitsSummary,
		"total_tenants": len(limitsSummary), "timestamp": time.Now().UTC(),
	}

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, response)
}

// GetConfig returns configuration information
func (s *Server) GetConfig(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	// Get discovery result
	result, err := s.discoveryEngine.DiscoverAll(ctx)
	if err != nil {
		s.recordError(c, "discovery_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Build configuration response
	config := map[string]interface{}{
		"mimir_components":  result.MimirComponents,
		"tenant_namespaces": result.TenantNamespaces,
		"config_maps":       result.ConfigMaps,
		"environment":       result.Environment,
		"last_updated":      result.LastUpdated,
	}

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, config)
}

// GetEnvironment returns comprehensive environment detection information
func (s *Server) GetEnvironment(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	// Get discovery result with environment information
	result, err := s.discoveryEngine.DiscoverAll(ctx)
	if err != nil {
		s.recordError(c, "discovery_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get auto-discovered limits
	autoDiscovery := limits.NewAutoDiscovery(s.discoveryEngine.GetK8sClient())
	discoveredLimits, err := autoDiscovery.DiscoverAllLimits(ctx, result.Environment.MimirNamespace)
	if err != nil {
		logrus.Warnf("Failed to get auto-discovered limits: %v", err)
		discoveredLimits = &limits.DiscoveredLimits{
			GlobalLimits:  make(map[string]interface{}),
			TenantLimits:  make(map[string]limits.TenantLimit),
			ConfigSources: []limits.ConfigSource{},
			LastUpdated:   time.Now(),
		}
	}

	// Build comprehensive environment response
	environment := map[string]interface{}{
		"cluster_info":         result.Environment,
		"auto_discovered":      discoveredLimits,
		"mimir_components":     result.MimirComponents,
		"detected_tenants":     result.Environment.DetectedTenants,
		"data_source_status":   result.Environment.DataSource,
		"is_production":        result.Environment.IsProduction,
		"total_config_sources": len(discoveredLimits.ConfigSources),
		"last_updated":         result.LastUpdated,
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

// GetLLMCapabilities returns the LLM assistant capabilities
func (s *Server) GetLLMCapabilities(c *gin.Context) {
	start := time.Now()

	capabilities := s.llmAssistant.GetAssistantCapabilities()

	response := map[string]interface{}{
		"capabilities": capabilities,
		"status":       "available",
		"last_updated": time.Now().UTC(),
		"example_queries": []string{
			"Why did rejection rate spike yesterday?",
			"What's causing the high memory usage?",
			"Explain the current ingestion trends",
			"How can I optimize this tenant's performance?",
			"What alerts should I set up for monitoring?",
		},
	}

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, response)
}
