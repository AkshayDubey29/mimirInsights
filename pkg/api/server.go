package api

import (
	"context"
	"net/http"
	"time"

	"github.com/akshaydubey29/mimirInsights/pkg/discovery"
	"github.com/akshaydubey29/mimirInsights/pkg/limits"
	"github.com/akshaydubey29/mimirInsights/pkg/metrics"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server handles all API requests
type Server struct {
	discoveryEngine *discovery.Engine
	metricsClient   *metrics.Client
	limitsAnalyzer  *limits.Analyzer

	// Prometheus metrics
	requestCounter  *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	errorCounter    *prometheus.CounterVec
}

// NewServer creates a new API server
func NewServer(discoveryEngine *discovery.Engine, metricsClient *metrics.Client, limitsAnalyzer *limits.Analyzer) *Server {
	// Initialize Prometheus metrics
	requestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mimir_insights_requests_total",
			Help: "Total number of API requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mimir_insights_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	errorCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mimir_insights_errors_total",
			Help: "Total number of API errors",
		},
		[]string{"method", "endpoint", "error_type"},
	)

	// Register metrics
	prometheus.MustRegister(requestCounter, requestDuration, errorCounter)

	return &Server{
		discoveryEngine: discoveryEngine,
		metricsClient:   metricsClient,
		limitsAnalyzer:  limitsAnalyzer,
		requestCounter:  requestCounter,
		requestDuration: requestDuration,
		errorCounter:    errorCounter,
	}
}

// HealthCheck handles health check requests
func (s *Server) HealthCheck(c *gin.Context) {
	start := time.Now()

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   100,
		"uptime":    time.Since(start).String(),
	}

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, health)
}

// GetTenants returns all discovered tenant namespaces
func (s *Server) GetTenants(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	// Perform discovery
	result, err := s.discoveryEngine.DiscoverAll(ctx)
	if err != nil {
		s.recordError(c, "discovery_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{error: err.Error()})
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
			c.JSON(http.StatusInternalServerError, gin.H{error: err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{error: err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{error: err.Error()})
		return
	}

	// Build configuration response
	config := map[string]interface{}{
		"mimir_components":  result.MimirComponents,
		"tenant_namespaces": result.TenantNamespaces,
		"config_maps":       result.ConfigMaps, "last_updated": result.LastUpdated,
	}

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, config)
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
		c.JSON(http.StatusBadRequest, gin.H{error: err.Error()})
		return
	}

	// Determine time range
	var timeRange metrics.TimeRange
	switch request.TimeRange {
	case "48h":
		timeRange = metrics.CreateTimeRange(48*time.Hour, 5*time.Minute)
	case "7d":
		timeRange = metrics.CreateTimeRange(7*24*time.Hour, 15*time.Minute)
	case "30d":
		timeRange = metrics.CreateTimeRange(30*24*time.Hour, "1h")
	case "60d":
		timeRange = metrics.CreateTimeRange(60*24*time.Hour, 2*time.Hour)
	default:
		timeRange = metrics.CreateTimeRange(7*24*time.Hour, 15*time.Minute) // Default to 7d
	}

	// Get tenant metrics
	tenantMetrics, err := s.metricsClient.GetTenantMetrics(ctx, request.TenantName, timeRange)
	if err != nil {
		s.recordError(c, "metrics_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{error: err.Error()})
		return
	}

	// Analyze limits
	tenantLimits, err := s.limitsAnalyzer.AnalyzeTenantLimits(ctx, request.TenantName)
	if err != nil {
		s.recordError(c, "limits_analysis_error", start)
		c.JSON(http.StatusInternalServerError, gin.H{error: err.Error()})
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

	// TODO: Implement drift detection
	driftStatus := map[string]interface{}{
		"status":        "no_drift",
		"last_check":    time.Now().UTC(),
		"drift_count":   0,
		"drift_details": map[string]interface{}{},
	}

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, driftStatus)
}

// GetCapacityReport returns capacity planning report
func (s *Server) GetCapacityReport(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	// Get report type from query parameter
	reportType := c.DefaultQuery("type", "weekly")
	tenantName := c.Query("tenant")

	var tenantNames []string
	if tenantName != "" {
		tenantNames = []string{tenantName}
	} else {
		// Get all tenants
		result, err := s.discoveryEngine.DiscoverAll(ctx)
		if err != nil {
			s.recordError(c, "discovery_error", start)
			c.JSON(http.StatusInternalServerError, gin.H{error: err.Error()})
			return
		}

		for _, tenant := range result.TenantNamespaces {
			tenantNames = append(tenantNames, tenant.Name)
		}
	}

	// Generate capacity report
	report := s.generateCapacityReport(ctx, tenantNames, reportType)

	s.recordMetrics(c, http.StatusOK, start)
	c.JSON(http.StatusOK, report)
}

// generateCapacityReport generates a capacity planning report
func (s *Server) generateCapacityReport(ctx context.Context, tenantNames []string, reportType string) map[string]interface{} {
	// TODO: Implement comprehensive capacity report generation
	report := map[string]interface{}{
		"report_type":  reportType,
		"generated_at": time.Now().UTC(),
		"tenants":      tenantNames,
		"summary": map[string]interface{}{
			"total_tenants": len(tenantNames),
			"high_risk":     0,
			"medium_risk":   0,
			"low_risk":      len(tenantNames),
		},
		"recommendations": []string{
			"Monitor ingestion rates closely",
			"Consider scaling up for high-utilization tenants",
			"Review and optimize label cardinality",
		},
	}

	return report
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

// recordError records error metrics
func (s *Server) recordError(c *gin.Context, errorType string, start time.Time) {
	method := c.Request.Method
	endpoint := c.FullPath()
	if endpoint == "" {
		endpoint = c.Request.URL.Path
	}

	s.errorCounter.WithLabelValues(method, endpoint, errorType).Inc()
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
