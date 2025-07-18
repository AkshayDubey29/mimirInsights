package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/akshaydubey29/mimirInsights/pkg/k8s"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// HealthChecker provides comprehensive health monitoring
type HealthChecker struct {
	k8sClient    *k8s.Client
	checks       map[string]HealthCheck
	alertManager *AlertManager
	metrics      *HealthMetrics
	config       HealthConfig
	mutex        sync.RWMutex
}

// HealthCheck interface for individual health checks
type HealthCheck interface {
	Name() string
	Check(ctx context.Context) HealthResult
	Timeout() time.Duration
	Critical() bool
}

// HealthResult represents the result of a health check
type HealthResult struct {
	Status    HealthStatus           `json:"status"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Error     error                  `json:"error,omitempty"`
}

// HealthStatus represents health check status
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnknown   HealthStatus = "unknown"
)

// HealthConfig configures health monitoring
type HealthConfig struct {
	CheckInterval   time.Duration `json:"check_interval"`
	CheckTimeout    time.Duration `json:"check_timeout"`
	RetryAttempts   int           `json:"retry_attempts"`
	AlertThreshold  int           `json:"alert_threshold"`
	EnableAlerting  bool          `json:"enable_alerting"`
	SlackWebhookURL string        `json:"slack_webhook_url"`
	PagerDutyKey    string        `json:"pagerduty_key"`
}

// SystemHealth represents overall system health
type SystemHealth struct {
	Status          HealthStatus            `json:"status"`
	Timestamp       time.Time               `json:"timestamp"`
	CheckResults    map[string]HealthResult `json:"check_results"`
	Summary         HealthSummary           `json:"summary"`
	Alerts          []ActiveAlert           `json:"alerts"`
	Recommendations []string                `json:"recommendations"`
}

// HealthSummary provides summary statistics
type HealthSummary struct {
	TotalChecks     int `json:"total_checks"`
	HealthyChecks   int `json:"healthy_checks"`
	DegradedChecks  int `json:"degraded_checks"`
	UnhealthyChecks int `json:"unhealthy_checks"`
	CriticalIssues  int `json:"critical_issues"`
}

// HealthMetrics provides Prometheus metrics for health monitoring
type HealthMetrics struct {
	CheckDuration   *prometheus.HistogramVec
	CheckStatus     *prometheus.GaugeVec
	AlertsTriggered *prometheus.CounterVec
	SystemStatus    *prometheus.GaugeVec
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(k8sClient *k8s.Client, config HealthConfig) *HealthChecker {
	metrics := &HealthMetrics{
		CheckDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "mimir_insights_health_check_duration_seconds",
				Help: "Duration of health checks",
			},
			[]string{"check_name", "status"},
		),
		CheckStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "mimir_insights_health_check_status",
				Help: "Health check status (1=healthy, 0=unhealthy)",
			},
			[]string{"check_name", "type"},
		),
		AlertsTriggered: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "mimir_insights_alerts_triggered_total",
				Help: "Total number of alerts triggered",
			},
			[]string{"check_name", "severity"},
		),
		SystemStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "mimir_insights_system_status",
				Help: "Overall system status",
			},
			[]string{"component"},
		),
	}

	prometheus.MustRegister(
		metrics.CheckDuration,
		metrics.CheckStatus,
		metrics.AlertsTriggered,
		metrics.SystemStatus,
	)

	hc := &HealthChecker{
		k8sClient:    k8sClient,
		checks:       make(map[string]HealthCheck),
		alertManager: NewAlertManager(config),
		metrics:      metrics,
		config:       config,
	}

	// Register default health checks
	hc.registerDefaultChecks()

	return hc
}

// registerDefaultChecks registers standard health checks
func (hc *HealthChecker) registerDefaultChecks() {
	hc.RegisterCheck(NewKubernetesHealthCheck(hc.k8sClient))
	hc.RegisterCheck(NewDatabaseHealthCheck())
	hc.RegisterCheck(NewMemoryHealthCheck())
	hc.RegisterCheck(NewDiskHealthCheck())
	hc.RegisterCheck(NewNetworkHealthCheck())
}

// RegisterCheck registers a new health check
func (hc *HealthChecker) RegisterCheck(check HealthCheck) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()
	hc.checks[check.Name()] = check
}

// UnregisterCheck removes a health check
func (hc *HealthChecker) UnregisterCheck(name string) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()
	delete(hc.checks, name)
}

// CheckHealth performs all health checks and returns system health
func (hc *HealthChecker) CheckHealth(ctx context.Context) *SystemHealth {
	start := time.Now()
	hc.mutex.RLock()
	checks := make(map[string]HealthCheck)
	for name, check := range hc.checks {
		checks[name] = check
	}
	hc.mutex.RUnlock()

	results := make(map[string]HealthResult)
	var wg sync.WaitGroup
	resultsChan := make(chan struct {
		name   string
		result HealthResult
	}, len(checks))

	// Run checks concurrently
	for name, check := range checks {
		wg.Add(1)
		go func(name string, check HealthCheck) {
			defer wg.Done()

			checkCtx, cancel := context.WithTimeout(ctx, check.Timeout())
			defer cancel()

			checkStart := time.Now()
			result := check.Check(checkCtx)
			result.Duration = time.Since(checkStart)
			result.Timestamp = time.Now()

			// Record metrics
			status := "unhealthy"
			if result.Status == StatusHealthy {
				status = "healthy"
			}
			hc.metrics.CheckDuration.WithLabelValues(name, status).Observe(result.Duration.Seconds())

			statusValue := 0.0
			if result.Status == StatusHealthy {
				statusValue = 1.0
			}
			hc.metrics.CheckStatus.WithLabelValues(name, "current").Set(statusValue)

			resultsChan <- struct {
				name   string
				result HealthResult
			}{name, result}
		}(name, check)
	}

	// Close channel when all checks complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for result := range resultsChan {
		results[result.name] = result.result
	}

	// Calculate overall status and summary
	summary := hc.calculateSummary(results)
	overallStatus := hc.calculateOverallStatus(results)

	// Check for alerts
	alerts := hc.checkForAlerts(results)

	// Generate recommendations
	recommendations := hc.generateRecommendations(results, summary)

	systemHealth := &SystemHealth{
		Status:          overallStatus,
		Timestamp:       time.Now(),
		CheckResults:    results,
		Summary:         summary,
		Alerts:          alerts,
		Recommendations: recommendations,
	}

	// Record overall system status
	systemStatusValue := 0.0
	if overallStatus == StatusHealthy {
		systemStatusValue = 1.0
	}
	hc.metrics.SystemStatus.WithLabelValues("overall").Set(systemStatusValue)

	logrus.Infof("Health check completed in %v: %s (%d/%d healthy)",
		time.Since(start), overallStatus, summary.HealthyChecks, summary.TotalChecks)

	return systemHealth
}

// calculateSummary calculates health summary statistics
func (hc *HealthChecker) calculateSummary(results map[string]HealthResult) HealthSummary {
	summary := HealthSummary{
		TotalChecks: len(results),
	}

	for _, result := range results {
		switch result.Status {
		case StatusHealthy:
			summary.HealthyChecks++
		case StatusDegraded:
			summary.DegradedChecks++
		case StatusUnhealthy:
			summary.UnhealthyChecks++
		}
	}

	return summary
}

// calculateOverallStatus determines overall system status
func (hc *HealthChecker) calculateOverallStatus(results map[string]HealthResult) HealthStatus {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	criticalUnhealthy := 0
	totalUnhealthy := 0
	totalDegraded := 0

	for name, result := range results {
		check := hc.checks[name]
		if result.Status == StatusUnhealthy {
			totalUnhealthy++
			if check.Critical() {
				criticalUnhealthy++
			}
		} else if result.Status == StatusDegraded {
			totalDegraded++
		}
	}

	// Critical components unhealthy = overall unhealthy
	if criticalUnhealthy > 0 {
		return StatusUnhealthy
	}

	// Multiple unhealthy = degraded
	if totalUnhealthy > 1 {
		return StatusDegraded
	}

	// Any unhealthy or multiple degraded = degraded
	if totalUnhealthy > 0 || totalDegraded > 1 {
		return StatusDegraded
	}

	// Single degraded = degraded
	if totalDegraded > 0 {
		return StatusDegraded
	}

	return StatusHealthy
}

// checkForAlerts checks for alerting conditions
func (hc *HealthChecker) checkForAlerts(results map[string]HealthResult) []ActiveAlert {
	var alerts []ActiveAlert

	for name, result := range results {
		if result.Status == StatusUnhealthy {
			alert := ActiveAlert{
				Name:        fmt.Sprintf("Health check failed: %s", name),
				Description: result.Message,
				Severity:    "critical",
				Timestamp:   result.Timestamp,
				Component:   name,
				Status:      "firing",
			}
			alerts = append(alerts, alert)

			// Trigger alert
			if hc.config.EnableAlerting {
				hc.alertManager.TriggerAlert(alert)
				hc.metrics.AlertsTriggered.WithLabelValues(name, "critical").Inc()
			}
		}
	}

	return alerts
}

// generateRecommendations generates actionable recommendations
func (hc *HealthChecker) generateRecommendations(results map[string]HealthResult, summary HealthSummary) []string {
	var recommendations []string

	if summary.CriticalIssues > 0 {
		recommendations = append(recommendations, "🚨 Critical issues detected - immediate attention required")
	}

	if summary.UnhealthyChecks > 0 {
		recommendations = append(recommendations, fmt.Sprintf("🔧 %d unhealthy components need investigation", summary.UnhealthyChecks))
	}

	if summary.DegradedChecks > 0 {
		recommendations = append(recommendations, fmt.Sprintf("⚠️ %d components showing degraded performance", summary.DegradedChecks))
	}

	if float64(summary.HealthyChecks)/float64(summary.TotalChecks) < 0.8 {
		recommendations = append(recommendations, "📊 Consider increasing monitoring frequency during incidents")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "✅ All systems operational - continue regular monitoring")
	}

	return recommendations
}

// StartPeriodicChecks starts periodic health checking
func (hc *HealthChecker) StartPeriodicChecks(ctx context.Context) {
	ticker := time.NewTicker(hc.config.CheckInterval)
	defer ticker.Stop()

	logrus.Infof("Starting periodic health checks every %v", hc.config.CheckInterval)

	for {
		select {
		case <-ctx.Done():
			logrus.Info("Stopping periodic health checks")
			return
		case <-ticker.C:
			health := hc.CheckHealth(ctx)
			if health.Status != StatusHealthy {
				logrus.Warnf("System health: %s (%d issues detected)", health.Status, len(health.Alerts))
			}
		}
	}
}

// GetHealthStatus returns current health status for HTTP endpoint
func (hc *HealthChecker) GetHealthStatus(ctx context.Context) (int, map[string]interface{}) {
	health := hc.CheckHealth(ctx)

	status := http.StatusOK
	if health.Status == StatusUnhealthy {
		status = http.StatusServiceUnavailable
	} else if health.Status == StatusDegraded {
		status = http.StatusPartialContent
	}

	response := map[string]interface{}{
		"status":    health.Status,
		"timestamp": health.Timestamp,
		"summary":   health.Summary,
		"checks":    health.CheckResults,
	}

	// Include alerts if any
	if len(health.Alerts) > 0 {
		response["alerts"] = health.Alerts
	}

	// Include recommendations
	if len(health.Recommendations) > 0 {
		response["recommendations"] = health.Recommendations
	}

	return status, response
}
