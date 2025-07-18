package limits

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/akshaydubey29/mimirInsights/pkg/config"
	"github.com/akshaydubey29/mimirInsights/pkg/k8s"
	"github.com/akshaydubey29/mimirInsights/pkg/metrics"
	"github.com/sirupsen/logrus"
)

// Analyzer provides AI-driven limit recommendations
type Analyzer struct {
	metricsClient *metrics.Client
	config        *config.Config
	autoDiscovery *AutoDiscovery
}

// LimitRecommendation represents a recommended limit value
type LimitRecommendation struct {
	LimitName        string    `json:"limit_name"`
	CurrentValue     float64   `json:"current_value"`
	ObservedPeak     float64   `json:"observed_peak"`
	RecommendedValue float64   `json:"recommended_value"`
	BufferPercent    float64   `json:"buffer_percent"`
	RiskLevel        string    `json:"risk_level"`
	Reason           string    `json:"reason"`
	LastUpdated      time.Time `json:"last_updated"`
}

// TenantLimits represents all limits for a tenant
type TenantLimits struct {
	TenantName      string                 `json:"tenant_name"`
	Recommendations []LimitRecommendation  `json:"recommendations"`
	MissingLimits   []string               `json:"missing_limits"`
	CurrentConfig   map[string]interface{} `json:"current_config"`
	AnalysisTime    time.Time              `json:"analysis_time"`
	RiskScore       float64                `json:"risk_score"`
}

// LimitType represents different types of Mimir limits
type LimitType struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	DefaultValue float64 `json:"default_value"`
	Unit         string  `json:"unit"`
	Category     string  `json:"category"`
}

// RiskLevel represents the risk level of a limit
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// NewAnalyzer creates a new limits analyzer
func NewAnalyzer(metricsClient *metrics.Client) *Analyzer {
	k8sClient, err := k8s.NewClient()
	if err != nil {
		logrus.Fatalf("Failed to create k8s client for analyzer: %v", err)
	}

	return &Analyzer{
		metricsClient: metricsClient,
		config:        config.Get(),
		autoDiscovery: NewAutoDiscovery(k8sClient),
	}
}

// AnalyzeTenantLimits analyzes and recommends limits for a tenant
func (a *Analyzer) AnalyzeTenantLimits(ctx context.Context, tenantName string) (*TenantLimits, error) {
	logrus.Infof("Analyzing limits for tenant: %s", tenantName)

	limits := &TenantLimits{
		TenantName:    tenantName,
		AnalysisTime:  time.Now(),
		CurrentConfig: make(map[string]interface{}),
	}

	// Get current configuration
	currentConfig, err := a.getCurrentTenantConfig(ctx, tenantName)
	if err != nil {
		return nil, fmt.Errorf("failed to get current config: %w", err)
	}
	limits.CurrentConfig = currentConfig

	// Analyze each limit type
	limitTypes := a.getLimitTypes()
	var recommendations []LimitRecommendation
	var missingLimits []string

	for _, limitType := range limitTypes {
		recommendation, err := a.analyzeLimit(ctx, tenantName, limitType, currentConfig)
		if err != nil {
			logrus.Warnf("Failed to analyze limit %s for %s: %v", limitType.Name, tenantName, err)
			continue
		}

		if recommendation == nil {
			// Limit is missing
			missingLimits = append(missingLimits, limitType.Name)
		} else {
			recommendations = append(recommendations, *recommendation)
		}
	}

	limits.Recommendations = recommendations
	limits.MissingLimits = missingLimits

	// Calculate overall risk score
	limits.RiskScore = a.calculateRiskScore(recommendations, missingLimits)

	return limits, nil
}

// analyzeLimit analyzes a specific limit for a tenant
func (a *Analyzer) analyzeLimit(ctx context.Context, tenantName string, limitType LimitType, currentConfig map[string]interface{}) (*LimitRecommendation, error) {
	// Get current value
	currentValue := a.getCurrentLimitValue(limitType.Name, currentConfig)
	if currentValue == nil {
		return nil, nil // Limit is missing
	}

	// Get observed peak from metrics
	observedPeak, err := a.getObservedPeak(ctx, tenantName, limitType)
	if err != nil {
		return nil, fmt.Errorf("failed to get observed peak: %w", err)
	}

	// Calculate recommended value with buffer
	bufferPercent := a.getBufferPercent(limitType)
	recommendedValue := observedPeak * (1 + bufferPercent/100)

	// Determine risk level
	riskLevel := a.calculateRiskLevel(currentValue.(float64), observedPeak, recommendedValue)

	// Generate reason
	reason := a.generateReason(limitType, currentValue.(float64), observedPeak, recommendedValue, riskLevel)

	recommendation := &LimitRecommendation{
		LimitName:        limitType.Name,
		CurrentValue:     currentValue.(float64),
		ObservedPeak:     observedPeak,
		RecommendedValue: recommendedValue,
		BufferPercent:    bufferPercent,
		RiskLevel:        string(riskLevel),
		Reason:           reason,
		LastUpdated:      time.Now(),
	}

	return recommendation, nil
}

// getObservedPeak gets the observed peak value for a limit type
func (a *Analyzer) getObservedPeak(ctx context.Context, tenantName string, limitType LimitType) (float64, error) {
	// Get metrics for different time ranges
	timeRanges := metrics.GetStandardTimeRanges()
	var maxPeak float64
	for rangeName, timeRange := range timeRanges {
		peakValues, err := a.metricsClient.GetPeakValues(ctx, tenantName, timeRange)
		if err != nil {
			logrus.Warnf("Failed to get peak values for %s in %s: %v", tenantName, rangeName, err)
			continue
		}

		// Map limit type to metric
		metricName := a.mapLimitToMetric(limitType.Name)
		if peakValue, exists := peakValues[metricName]; exists {
			if peakValue > maxPeak {
				maxPeak = peakValue
			}
		}
	}

	return maxPeak, nil
}

// mapLimitToMetric maps a limit name to its corresponding metric
func (a *Analyzer) mapLimitToMetric(limitName string) string {
	limitName = strings.ToLower(limitName)

	switch {
	case strings.Contains(limitName, "ingestion_rate"):
		return "ingestion_rate"
	case strings.Contains(limitName, "series_per_user"):
		return "active_series"
	case strings.Contains(limitName, "memory"):
		return "memory_usage"
	case strings.Contains(limitName, "rejected"):
		return "rejected_samples"
	case strings.Contains(limitName, "limits_reached"):
		return "limits_reached"
	default:
		return "ingestion_rate" // Default fallback
	}
}

// getCurrentLimitValue gets the current value for a limit from configuration
func (a *Analyzer) getCurrentLimitValue(limitName string, config map[string]interface{}) interface{} {
	// Check direct key
	if value, exists := config[limitName]; exists {
		return value
	}

	// Check nested keys (e.g., limits.ingestion_rate)
	parts := strings.Split(limitName, ".")
	if len(parts) > 1 {
		if nested, exists := config[parts[0]]; exists {
			if nestedMap, ok := nested.(map[string]interface{}); ok {
				return nestedMap[parts[1]]
			}
		}
	}

	return nil
}

// getBufferPercent gets the buffer percentage for a limit type
func (a *Analyzer) getBufferPercent(limitType LimitType) float64 {
	// Different buffer percentages based on limit category
	switch limitType.Category {
	case "critical":
		return 20 // 20% buffer for critical limits
	case "important":
		return 15 // 15% buffer for important limits
	default:
		return 10 // 10% buffer for regular limits
	}
}

// calculateRiskLevel calculates the risk level for a limit
func (a *Analyzer) calculateRiskLevel(currentValue, observedPeak, recommendedValue float64) RiskLevel {
	// Calculate utilization percentage
	utilization := (observedPeak / currentValue) * 100
	switch {
	case utilization >= 95:
		return RiskCritical
	case utilization >= 80:
		return RiskHigh
	case utilization >= 60:
		return RiskMedium
	default:
		return RiskLow
	}
}

// generateReason generates a human-readable reason for the recommendation
func (a *Analyzer) generateReason(limitType LimitType, currentValue, observedPeak, recommendedValue float64, riskLevel RiskLevel) string {
	utilization := (observedPeak / currentValue) * 100
	switch riskLevel {
	case RiskCritical:
		return fmt.Sprintf("Critical: Current utilization is %.1f%% of limit. Immediate action required to prevent rejections.", utilization)
	case RiskHigh:
		return fmt.Sprintf("High: Current utilization is %.1f%% of limit. Consider increasing limit to prevent future issues.", utilization)
	case RiskMedium:
		return fmt.Sprintf("Medium: Current utilization is %.1f%% of limit. Monitor closely and consider optimization.", utilization)
	case RiskLow:
		return fmt.Sprintf("Low: Current utilization is %.1f%% of limit. Limit appears adequate for current usage.", utilization)
	default:
		return "Unable to determine risk level."
	}
}

// calculateRiskScore calculates an overall risk score for the tenant
func (a *Analyzer) calculateRiskScore(recommendations []LimitRecommendation, missingLimits []string) float64 {
	if len(recommendations) == 0 && len(missingLimits) == 0 {
		return 0
	}

	var totalScore float64
	var totalWeight float64
	// Score based on recommendations
	for _, rec := range recommendations {
		weight := a.getRiskWeight(rec.RiskLevel)
		score := a.getRiskScore(rec.RiskLevel)
		totalScore += score * weight
		totalWeight += weight
	}

	// Penalty for missing limits
	missingPenalty := float64(len(missingLimits)) * 10
	totalScore += missingPenalty

	if totalWeight > 0 {
		return totalScore / totalWeight
	}

	return totalScore
}

// getRiskWeight gets the weight for a risk level
func (a *Analyzer) getRiskWeight(riskLevel string) float64 {
	switch riskLevel {
	case string(RiskCritical):
		return 4
	case string(RiskHigh):
		return 3
	case string(RiskMedium):
		return 2
	case string(RiskLow):
		return 1.0
	default:
		return 1.0
	}
}

// getRiskScore gets the score for a risk level
func (a *Analyzer) getRiskScore(riskLevel string) float64 {
	switch riskLevel {
	case string(RiskCritical):
		return 100
	case string(RiskHigh):
		return 75
	case string(RiskMedium):
		return 50
	case string(RiskLow):
		return 25.0
	default:
		return 0.0
	}
}

// getLimitTypes returns all supported Mimir limit types
func (a *Analyzer) getLimitTypes() []LimitType {
	return []LimitType{
		{
			Name:         "max_global_series_per_user",
			Description:  "Maximum number of active series per user",
			DefaultValue: 5000000,
			Unit:         "series",
			Category:     "critical",
		},
		{
			Name:         "max_label_names_per_series",
			Description:  "Maximum number of label names per series",
			DefaultValue: 30,
			Unit:         "labels",
			Category:     "important",
		},
		{
			Name:         "max_label_value_length",
			Description:  "Maximum length of label values",
			DefaultValue: 2048,
			Unit:         "bytes",
			Category:     "important",
		},
		{
			Name:         "ingestion_rate",
			Description:  "Maximum ingestion rate per user",
			DefaultValue: 10000,
			Unit:         "samples/sec",
			Category:     "critical",
		},
		{
			Name:         "ingestion_burst_size",
			Description:  "Maximum burst size for ingestion",
			DefaultValue: 20000,
			Unit:         "samples",
			Category:     "important",
		},
		{
			Name:         "max_global_series_per_metric",
			Description:  "Maximum number of active series per metric",
			DefaultValue: 100000,
			Unit:         "series",
			Category:     "important",
		},
		{
			Name:         "max_global_exemplars_per_user",
			Description:  "Maximum number of exemplars per user",
			DefaultValue: 100000,
			Unit:         "exemplars",
			Category:     "regular",
		},
		{
			Name:         "max_global_metadata_per_user",
			Description:  "Maximum number of metadata entries per user",
			DefaultValue: 100000,
			Unit:         "entries",
			Category:     "regular",
		},
		{
			Name:         "max_global_metadata_per_metric",
			Description:  "Maximum number of metadata entries per metric",
			DefaultValue: 10000,
			Unit:         "entries",
			Category:     "regular",
		},
		{
			Name:         "max_global_exemplars_per_metric",
			Description:  "Maximum number of exemplars per metric",
			DefaultValue: 10000,
			Unit:         "exemplars",
			Category:     "regular",
		},
	}
}

// getCurrentTenantConfig gets the current configuration for a tenant
func (a *Analyzer) getCurrentTenantConfig(ctx context.Context, tenantName string) (map[string]interface{}, error) {
	// Use auto-discovery to get actual tenant configuration
	discovered, err := a.autoDiscovery.DiscoverAllLimits(ctx, a.config.Mimir.Namespace)
	if err != nil {
		logrus.Warnf("Failed to auto-discover limits, falling back to empty config: %v", err)
		return make(map[string]interface{}), nil
	}

	// Start with global limits as defaults
	config := make(map[string]interface{})
	for key, value := range discovered.GlobalLimits {
		config[key] = value
	}

	// Override with tenant-specific limits if they exist
	if tenantLimit, exists := discovered.TenantLimits[tenantName]; exists {
		for key, value := range tenantLimit.Limits {
			config[key] = value
		}
		logrus.Infof("Found tenant-specific limits for %s from source: %s", tenantName, tenantLimit.Source)
	}

	// If no tenant-specific config found, try alternative tenant identifiers
	if len(config) == 0 {
		// Try with different naming patterns
		alternativeNames := []string{
			fmt.Sprintf("tenant-%s", tenantName),
			fmt.Sprintf("%s-tenant", tenantName),
			strings.ToLower(tenantName),
			strings.ToUpper(tenantName),
		}

		for _, altName := range alternativeNames {
			if tenantLimit, exists := discovered.TenantLimits[altName]; exists {
				for key, value := range tenantLimit.Limits {
					config[key] = value
				}
				logrus.Infof("Found tenant limits for %s using alternative name %s", tenantName, altName)
				break
			}
		}
	}

	// If still no configuration found, return global limits only
	if len(config) == 0 {
		logrus.Warnf("No specific configuration found for tenant %s, using global limits only", tenantName)
		config = discovered.GlobalLimits
	}

	return config, nil
}

// GetTenantLimitsSummary gets a summary of limits for all tenants
func (a *Analyzer) GetTenantLimitsSummary(ctx context.Context, tenantNames []string) (map[string]*TenantLimits, error) {
	summary := make(map[string]*TenantLimits)

	for _, tenantName := range tenantNames {
		limits, err := a.AnalyzeTenantLimits(ctx, tenantName)
		if err != nil {
			logrus.Warnf("Failed to analyze limits for %s: %v", tenantName, err)
			continue
		}
		summary[tenantName] = limits
	}

	return summary, nil
}
