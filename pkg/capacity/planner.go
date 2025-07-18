package capacity

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/akshaydubey29/mimirInsights/pkg/limits"
	"github.com/akshaydubey29/mimirInsights/pkg/metrics"
	"github.com/sirupsen/logrus"
)

// Planner handles capacity planning and forecasting
type Planner struct {
	metricsClient  *metrics.Client
	limitsAnalyzer *limits.Analyzer
}

// CapacityReport represents a comprehensive capacity planning report
type CapacityReport struct {
	ReportID        string                 `json:"report_id"`
	GeneratedAt     time.Time              `json:"generated_at"`
	ReportType      string                 `json:"report_type"` // "weekly", "monthly", "quarterly"
	Period          ReportPeriod           `json:"period"`
	Summary         CapacitySummary        `json:"summary"`
	TenantReports   []TenantCapacityReport `json:"tenant_reports"`
	Forecasting     ForecastingSummary     `json:"forecasting"`
	Recommendations []string               `json:"recommendations"`
	RiskAssessment  RiskAssessment         `json:"risk_assessment"`
	TrendAnalysis   TrendAnalysis          `json:"trend_analysis"`
}

// ReportPeriod represents the time period for the report
type ReportPeriod struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Duration  string    `json:"duration"`
}

// CapacitySummary provides overall capacity summary
type CapacitySummary struct {
	TotalTenants          int     `json:"total_tenants"`
	TotalIngestionRate    float64 `json:"total_ingestion_rate"`
	TotalActiveSeries     int64   `json:"total_active_series"`
	AverageUtilization    float64 `json:"average_utilization"`
	CapacityUtilization   float64 `json:"capacity_utilization"`
	ProjectedGrowthRate   float64 `json:"projected_growth_rate"`
	EstimatedCapacityDays int     `json:"estimated_capacity_days"`
}

// TenantCapacityReport represents capacity analysis for a single tenant
type TenantCapacityReport struct {
	TenantName         string             `json:"tenant_name"`
	CurrentCapacity    TenantCapacity     `json:"current_capacity"`
	UtilizationTrend   UtilizationTrend   `json:"utilization_trend"`
	Forecasting        TenantForecast     `json:"forecasting"`
	Recommendations    []string           `json:"recommendations"`
	RiskLevel          string             `json:"risk_level"`
	BottleneckAnalysis BottleneckAnalysis `json:"bottleneck_analysis"`
}

// TenantCapacity represents current capacity metrics for a tenant
type TenantCapacity struct {
	IngestionRate float64 `json:"ingestion_rate"`
	ActiveSeries  int64   `json:"active_series"`
	MemoryUsage   float64 `json:"memory_usage"`
	CPUUsage      float64 `json:"cpu_usage"`
	StorageUsage  float64 `json:"storage_usage"`
	QueueDepth    int     `json:"queue_depth"`
	ErrorRate     float64 `json:"error_rate"`
}

// UtilizationTrend represents utilization trends over time
type UtilizationTrend struct {
	Direction       string  `json:"direction"` // "increasing", "decreasing", "stable"
	GrowthRate      float64 `json:"growth_rate"`
	Seasonality     string  `json:"seasonality"`
	PeakUtilization float64 `json:"peak_utilization"`
	LowUtilization  float64 `json:"low_utilization"`
	TrendConfidence float64 `json:"trend_confidence"`
}

// TenantForecast represents capacity forecasting for a tenant
type TenantForecast struct {
	TimeHorizon            string         `json:"time_horizon"` // "30d", "60d", "90d"
	PredictedCapacity      TenantCapacity `json:"predicted_capacity"`
	ConfidenceInterval     float64        `json:"confidence_interval"`
	CapacityExhaustionDate *time.Time     `json:"capacity_exhaustion_date,omitempty"`
	RecommendedActions     []string       `json:"recommended_actions"`
}

// BottleneckAnalysis identifies capacity bottlenecks
type BottleneckAnalysis struct {
	PrimaryBottleneck    string   `json:"primary_bottleneck"`
	SecondaryBottlenecks []string `json:"secondary_bottlenecks"`
	ImpactAssessment     string   `json:"impact_assessment"`
	MitigationSteps      []string `json:"mitigation_steps"`
}

// ForecastingSummary provides overall forecasting summary
type ForecastingSummary struct {
	GlobalTrend            string     `json:"global_trend"`
	PredictedGrowthRate    float64    `json:"predicted_growth_rate"`
	CapacityExhaustionDate *time.Time `json:"capacity_exhaustion_date,omitempty"`
	ScalingRecommendations []string   `json:"scaling_recommendations"`
	ConfidenceLevel        float64    `json:"confidence_level"`
}

// RiskAssessment provides risk analysis
type RiskAssessment struct {
	OverallRiskLevel     string               `json:"overall_risk_level"`
	RiskFactors          []RiskFactor         `json:"risk_factors"`
	MitigationStrategies []MitigationStrategy `json:"mitigation_strategies"`
	AlertThresholds      map[string]float64   `json:"alert_thresholds"`
}

// RiskFactor represents a specific risk factor
type RiskFactor struct {
	Factor      string  `json:"factor"`
	Severity    string  `json:"severity"`
	Probability float64 `json:"probability"`
	Impact      string  `json:"impact"`
	Description string  `json:"description"`
}

// MitigationStrategy represents a risk mitigation strategy
type MitigationStrategy struct {
	Strategy      string   `json:"strategy"`
	Priority      string   `json:"priority"`
	Timeline      string   `json:"timeline"`
	Steps         []string `json:"steps"`
	Effectiveness float64  `json:"effectiveness"`
}

// TrendAnalysis provides trend analysis over multiple periods
type TrendAnalysis struct {
	WeeklyTrends     []PeriodTrend     `json:"weekly_trends"`
	MonthlyTrends    []PeriodTrend     `json:"monthly_trends"`
	SeasonalPatterns []SeasonalPattern `json:"seasonal_patterns"`
	Anomalies        []TrendAnomaly    `json:"anomalies"`
}

// PeriodTrend represents trends for a specific period
type PeriodTrend struct {
	Period     string    `json:"period"`
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	GrowthRate float64   `json:"growth_rate"`
	Direction  string    `json:"direction"`
	Confidence float64   `json:"confidence"`
}

// SeasonalPattern represents seasonal usage patterns
type SeasonalPattern struct {
	Pattern     string  `json:"pattern"`
	Description string  `json:"description"`
	Impact      float64 `json:"impact"`
	Timing      string  `json:"timing"`
}

// TrendAnomaly represents detected anomalies in usage patterns
type TrendAnomaly struct {
	DetectedAt  time.Time `json:"detected_at"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Impact      float64   `json:"impact"`
}

// ExportFormat represents different export formats
type ExportFormat string

const (
	ExportFormatPDF  ExportFormat = "pdf"
	ExportFormatCSV  ExportFormat = "csv"
	ExportFormatJSON ExportFormat = "json"
	ExportFormatHTML ExportFormat = "html"
)

// NewPlanner creates a new capacity planner
func NewPlanner(metricsClient *metrics.Client, limitsAnalyzer *limits.Analyzer) *Planner {
	return &Planner{
		metricsClient:  metricsClient,
		limitsAnalyzer: limitsAnalyzer,
	}
}

// GenerateCapacityReport generates a comprehensive capacity planning report
func (p *Planner) GenerateCapacityReport(ctx context.Context, reportType string, tenantNames []string) (*CapacityReport, error) {
	logrus.Infof("Generating %s capacity report for %d tenants", reportType, len(tenantNames))

	reportID := fmt.Sprintf("%s_%d", reportType, time.Now().Unix())
	period := p.getReportPeriod(reportType)

	report := &CapacityReport{
		ReportID:    reportID,
		GeneratedAt: time.Now(),
		ReportType:  reportType,
		Period:      period,
	}

	// Generate tenant capacity reports
	var tenantReports []TenantCapacityReport
	totalIngestionRate := 0.0
	totalActiveSeries := int64(0)

	for _, tenantName := range tenantNames {
		tenantReport, err := p.generateTenantCapacityReport(ctx, tenantName, period)
		if err != nil {
			logrus.Warnf("Failed to generate capacity report for tenant %s: %v", tenantName, err)
			continue
		}

		tenantReports = append(tenantReports, *tenantReport)
		totalIngestionRate += tenantReport.CurrentCapacity.IngestionRate
		totalActiveSeries += tenantReport.CurrentCapacity.ActiveSeries
	}

	report.TenantReports = tenantReports

	// Generate summary
	report.Summary = p.generateCapacitySummary(tenantReports, totalIngestionRate, totalActiveSeries)

	// Generate forecasting
	report.Forecasting = p.generateForecastingSummary(tenantReports)

	// Generate risk assessment
	report.RiskAssessment = p.generateRiskAssessment(tenantReports)

	// Generate trend analysis
	report.TrendAnalysis = p.generateTrendAnalysis(ctx, tenantNames, period)

	// Generate recommendations
	report.Recommendations = p.generateRecommendations(report)

	logrus.Infof("Successfully generated %s capacity report with %d tenant reports",
		reportType, len(tenantReports))

	return report, nil
}

// generateTenantCapacityReport generates capacity report for a single tenant
func (p *Planner) generateTenantCapacityReport(ctx context.Context, tenantName string, period ReportPeriod) (*TenantCapacityReport, error) {
	// Get current capacity metrics
	currentCapacity, err := p.getCurrentTenantCapacity(ctx, tenantName)
	if err != nil {
		return nil, fmt.Errorf("failed to get current capacity: %w", err)
	}

	// Analyze utilization trends
	utilizationTrend, err := p.analyzeUtilizationTrend(ctx, tenantName, period)
	if err != nil {
		logrus.Warnf("Failed to analyze utilization trend for %s: %v", tenantName, err)
		utilizationTrend = &UtilizationTrend{Direction: "stable", GrowthRate: 0}
	}

	// Generate forecasting
	forecasting := p.generateTenantForecast(currentCapacity, utilizationTrend)

	// Analyze bottlenecks
	bottleneckAnalysis := p.analyzeBottlenecks(currentCapacity, utilizationTrend)

	// Calculate risk level
	riskLevel := p.calculateTenantRiskLevel(currentCapacity, utilizationTrend, forecasting)

	// Generate recommendations
	recommendations := p.generateTenantRecommendations(currentCapacity, utilizationTrend, forecasting, bottleneckAnalysis)

	return &TenantCapacityReport{
		TenantName:         tenantName,
		CurrentCapacity:    *currentCapacity,
		UtilizationTrend:   *utilizationTrend,
		Forecasting:        forecasting,
		Recommendations:    recommendations,
		RiskLevel:          riskLevel,
		BottleneckAnalysis: bottleneckAnalysis,
	}, nil
}

// getCurrentTenantCapacity gets current capacity metrics for a tenant
func (p *Planner) getCurrentTenantCapacity(ctx context.Context, tenantName string) (*TenantCapacity, error) {
	// Get metrics for the last 24 hours
	timeRange := metrics.CreateTimeRange(24*time.Hour, "5m")
	tenantMetrics, err := p.metricsClient.GetTenantMetrics(ctx, tenantName, timeRange)
	if err != nil {
		return nil, err
	}

	// Calculate average values
	capacity := &TenantCapacity{
		IngestionRate: p.calculateAverageFromSeries(tenantMetrics.Metrics["ingestion_rate"]),
		ActiveSeries:  int64(p.calculateAverageFromSeries(tenantMetrics.Metrics["active_series"])),
		MemoryUsage:   p.calculateAverageFromSeries(tenantMetrics.Metrics["memory_usage"]),
		QueueDepth:    int(p.calculateAverageFromSeries(tenantMetrics.Metrics["queue_depth"])),
		ErrorRate:     p.calculateAverageFromSeries(tenantMetrics.Metrics["rejected_samples"]),
	}

	// Simulate CPU and storage usage (in real implementation, get from metrics)
	capacity.CPUUsage = 45.0 + (capacity.IngestionRate / 1000.0 * 10.0)
	capacity.StorageUsage = float64(capacity.ActiveSeries) * 0.001 // 1KB per series

	return capacity, nil
}

// calculateAverageFromSeries calculates average value from metric series
func (p *Planner) calculateAverageFromSeries(series []metrics.MetricSeries) float64 {
	if len(series) == 0 {
		return 0.0
	}

	total := 0.0
	count := 0

	for _, s := range series {
		for _, value := range s.Values {
			total += value.Value
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	return total / float64(count)
}

// analyzeUtilizationTrend analyzes utilization trends for a tenant
func (p *Planner) analyzeUtilizationTrend(ctx context.Context, tenantName string, period ReportPeriod) (*UtilizationTrend, error) {
	// Get historical metrics for trend analysis
	timeRange := metrics.CreateTimeRange(period.EndDate.Sub(period.StartDate), "1h")
	tenantMetrics, err := p.metricsClient.GetTenantMetrics(ctx, tenantName, timeRange)
	if err != nil {
		return nil, err
	}

	// Analyze ingestion rate trend
	ingestionSeries := tenantMetrics.Metrics["ingestion_rate"]
	growthRate := p.calculateGrowthRate(ingestionSeries)

	direction := "stable"
	if growthRate > 0.05 {
		direction = "increasing"
	} else if growthRate < -0.05 {
		direction = "decreasing"
	}

	// Calculate peak and low utilization
	peakUtil, lowUtil := p.calculateUtilizationRange(ingestionSeries)

	return &UtilizationTrend{
		Direction:       direction,
		GrowthRate:      growthRate,
		Seasonality:     p.detectSeasonality(ingestionSeries),
		PeakUtilization: peakUtil,
		LowUtilization:  lowUtil,
		TrendConfidence: 0.8, // Simplified confidence calculation
	}, nil
}

// calculateGrowthRate calculates growth rate from metric series
func (p *Planner) calculateGrowthRate(series []metrics.MetricSeries) float64 {
	if len(series) == 0 {
		return 0.0
	}

	// Simple linear regression for growth rate
	var values []float64
	for _, s := range series {
		for _, value := range s.Values {
			values = append(values, value.Value)
		}
	}

	if len(values) < 2 {
		return 0.0
	}

	// Calculate simple slope
	firstValue := values[0]
	lastValue := values[len(values)-1]

	if firstValue == 0 {
		return 0.0
	}

	return (lastValue - firstValue) / firstValue
}

// calculateUtilizationRange calculates peak and low utilization
func (p *Planner) calculateUtilizationRange(series []metrics.MetricSeries) (float64, float64) {
	if len(series) == 0 {
		return 0.0, 0.0
	}

	var values []float64
	for _, s := range series {
		for _, value := range s.Values {
			values = append(values, value.Value)
		}
	}

	if len(values) == 0 {
		return 0.0, 0.0
	}

	sort.Float64s(values)
	return values[len(values)-1], values[0] // max, min
}

// detectSeasonality detects seasonal patterns in metrics
func (p *Planner) detectSeasonality(series []metrics.MetricSeries) string {
	// Simplified seasonality detection
	// In a real implementation, this would use more sophisticated time series analysis
	return "daily" // Default assumption
}

// generateTenantForecast generates forecasting for a tenant
func (p *Planner) generateTenantForecast(currentCapacity *TenantCapacity, trend *UtilizationTrend) TenantForecast {
	// Simple forecasting based on current trend
	growthMultiplier := 1.0 + (trend.GrowthRate * 30) // 30 days projection

	predictedCapacity := TenantCapacity{
		IngestionRate: currentCapacity.IngestionRate * growthMultiplier,
		ActiveSeries:  int64(float64(currentCapacity.ActiveSeries) * growthMultiplier),
		MemoryUsage:   currentCapacity.MemoryUsage * growthMultiplier,
		CPUUsage:      currentCapacity.CPUUsage * growthMultiplier,
		StorageUsage:  currentCapacity.StorageUsage * growthMultiplier,
		QueueDepth:    int(float64(currentCapacity.QueueDepth) * growthMultiplier),
		ErrorRate:     currentCapacity.ErrorRate,
	}

	var exhaustionDate *time.Time
	if trend.GrowthRate > 0.1 { // High growth rate
		// Estimate when capacity might be exhausted (simplified)
		daysToExhaustion := int(90.0 / (trend.GrowthRate * 100)) // Rough estimation
		if daysToExhaustion > 0 && daysToExhaustion < 180 {
			exhaustion := time.Now().AddDate(0, 0, daysToExhaustion)
			exhaustionDate = &exhaustion
		}
	}

	var recommendedActions []string
	if predictedCapacity.CPUUsage > 80 {
		recommendedActions = append(recommendedActions, "Scale up CPU resources")
	}
	if predictedCapacity.MemoryUsage > 80 {
		recommendedActions = append(recommendedActions, "Increase memory allocation")
	}
	if trend.GrowthRate > 0.2 {
		recommendedActions = append(recommendedActions, "Consider horizontal scaling")
	}

	return TenantForecast{
		TimeHorizon:            "30d",
		PredictedCapacity:      predictedCapacity,
		ConfidenceInterval:     0.7,
		CapacityExhaustionDate: exhaustionDate,
		RecommendedActions:     recommendedActions,
	}
}

// analyzeBottlenecks analyzes capacity bottlenecks
func (p *Planner) analyzeBottlenecks(capacity *TenantCapacity, trend *UtilizationTrend) BottleneckAnalysis {
	var bottlenecks []string
	var primaryBottleneck string

	// Identify bottlenecks based on utilization
	utilizationFactors := map[string]float64{
		"CPU":     capacity.CPUUsage,
		"Memory":  capacity.MemoryUsage,
		"Storage": capacity.StorageUsage,
	}

	maxUtilization := 0.0
	for resource, utilization := range utilizationFactors {
		if utilization > 70 {
			bottlenecks = append(bottlenecks, resource)
		}
		if utilization > maxUtilization {
			maxUtilization = utilization
			primaryBottleneck = resource
		}
	}

	if capacity.ErrorRate > 5.0 {
		bottlenecks = append(bottlenecks, "Error Rate")
		if capacity.ErrorRate > maxUtilization {
			primaryBottleneck = "Error Rate"
		}
	}

	if primaryBottleneck == "" {
		primaryBottleneck = "None"
	}

	// Generate mitigation steps
	var mitigationSteps []string
	switch primaryBottleneck {
	case "CPU":
		mitigationSteps = []string{
			"Increase CPU limits",
			"Optimize query performance",
			"Consider horizontal scaling",
		}
	case "Memory":
		mitigationSteps = []string{
			"Increase memory limits",
			"Optimize memory usage",
			"Review series cardinality",
		}
	case "Storage":
		mitigationSteps = []string{
			"Increase storage capacity",
			"Implement data retention policies",
			"Optimize compression",
		}
	case "Error Rate":
		mitigationSteps = []string{
			"Investigate error root causes",
			"Review limit configurations",
			"Implement retry mechanisms",
		}
	default:
		mitigationSteps = []string{"Monitor resource usage trends"}
	}

	impact := "low"
	if maxUtilization > 80 {
		impact = "high"
	} else if maxUtilization > 60 {
		impact = "medium"
	}

	return BottleneckAnalysis{
		PrimaryBottleneck:    primaryBottleneck,
		SecondaryBottlenecks: bottlenecks,
		ImpactAssessment:     impact,
		MitigationSteps:      mitigationSteps,
	}
}

// calculateTenantRiskLevel calculates risk level for a tenant
func (p *Planner) calculateTenantRiskLevel(capacity *TenantCapacity, trend *UtilizationTrend, forecast TenantForecast) string {
	riskScore := 0.0

	// CPU risk
	if capacity.CPUUsage > 90 {
		riskScore += 3.0
	} else if capacity.CPUUsage > 80 {
		riskScore += 2.0
	} else if capacity.CPUUsage > 70 {
		riskScore += 1.0
	}

	// Memory risk
	if capacity.MemoryUsage > 90 {
		riskScore += 3.0
	} else if capacity.MemoryUsage > 80 {
		riskScore += 2.0
	} else if capacity.MemoryUsage > 70 {
		riskScore += 1.0
	}

	// Growth trend risk
	if trend.GrowthRate > 0.3 {
		riskScore += 2.0
	} else if trend.GrowthRate > 0.1 {
		riskScore += 1.0
	}

	// Error rate risk
	if capacity.ErrorRate > 10 {
		riskScore += 2.0
	} else if capacity.ErrorRate > 5 {
		riskScore += 1.0
	}

	// Capacity exhaustion risk
	if forecast.CapacityExhaustionDate != nil {
		daysToExhaustion := int(forecast.CapacityExhaustionDate.Sub(time.Now()).Hours() / 24)
		if daysToExhaustion < 30 {
			riskScore += 3.0
		} else if daysToExhaustion < 60 {
			riskScore += 2.0
		} else if daysToExhaustion < 90 {
			riskScore += 1.0
		}
	}

	// Convert score to risk level
	if riskScore >= 6 {
		return "critical"
	} else if riskScore >= 4 {
		return "high"
	} else if riskScore >= 2 {
		return "medium"
	}
	return "low"
}

// getReportPeriod gets the time period for a report type
func (p *Planner) getReportPeriod(reportType string) ReportPeriod {
	endDate := time.Now()
	var startDate time.Time
	var duration string

	switch reportType {
	case "weekly":
		startDate = endDate.AddDate(0, 0, -7)
		duration = "7 days"
	case "monthly":
		startDate = endDate.AddDate(0, -1, 0)
		duration = "30 days"
	case "quarterly":
		startDate = endDate.AddDate(0, -3, 0)
		duration = "90 days"
	default:
		startDate = endDate.AddDate(0, 0, -7)
		duration = "7 days"
	}

	return ReportPeriod{
		StartDate: startDate,
		EndDate:   endDate,
		Duration:  duration,
	}
}

// Additional helper methods would continue here...
// For brevity, I'll implement the key remaining methods

// generateCapacitySummary generates overall capacity summary
func (p *Planner) generateCapacitySummary(tenantReports []TenantCapacityReport, totalIngestionRate float64, totalActiveSeries int64) CapacitySummary {
	if len(tenantReports) == 0 {
		return CapacitySummary{}
	}

	totalUtilization := 0.0
	for _, report := range tenantReports {
		cpuMemAvg := (report.CurrentCapacity.CPUUsage + report.CurrentCapacity.MemoryUsage) / 2
		totalUtilization += cpuMemAvg
	}
	avgUtilization := totalUtilization / float64(len(tenantReports))

	// Simplified capacity calculations
	capacityUtilization := math.Min(avgUtilization, 100.0)
	projectedGrowthRate := 0.1  // Default 10% growth
	estimatedCapacityDays := 90 // Default 90 days

	return CapacitySummary{
		TotalTenants:          len(tenantReports),
		TotalIngestionRate:    totalIngestionRate,
		TotalActiveSeries:     totalActiveSeries,
		AverageUtilization:    avgUtilization,
		CapacityUtilization:   capacityUtilization,
		ProjectedGrowthRate:   projectedGrowthRate,
		EstimatedCapacityDays: estimatedCapacityDays,
	}
}

// generateForecastingSummary generates forecasting summary
func (p *Planner) generateForecastingSummary(tenantReports []TenantCapacityReport) ForecastingSummary {
	if len(tenantReports) == 0 {
		return ForecastingSummary{}
	}

	increasingTrends := 0
	totalGrowthRate := 0.0
	var earliestExhaustion *time.Time

	for _, report := range tenantReports {
		if report.UtilizationTrend.Direction == "increasing" {
			increasingTrends++
		}
		totalGrowthRate += report.UtilizationTrend.GrowthRate

		if report.Forecasting.CapacityExhaustionDate != nil {
			if earliestExhaustion == nil || report.Forecasting.CapacityExhaustionDate.Before(*earliestExhaustion) {
				earliestExhaustion = report.Forecasting.CapacityExhaustionDate
			}
		}
	}

	avgGrowthRate := totalGrowthRate / float64(len(tenantReports))

	globalTrend := "stable"
	if float64(increasingTrends)/float64(len(tenantReports)) > 0.6 {
		globalTrend = "increasing"
	}

	var scalingRecommendations []string
	if avgGrowthRate > 0.2 {
		scalingRecommendations = append(scalingRecommendations, "Consider cluster-level scaling")
	}
	if earliestExhaustion != nil {
		scalingRecommendations = append(scalingRecommendations, "Address capacity exhaustion risks")
	}

	return ForecastingSummary{
		GlobalTrend:            globalTrend,
		PredictedGrowthRate:    avgGrowthRate,
		CapacityExhaustionDate: earliestExhaustion,
		ScalingRecommendations: scalingRecommendations,
		ConfidenceLevel:        0.75,
	}
}

// generateRiskAssessment generates risk assessment
func (p *Planner) generateRiskAssessment(tenantReports []TenantCapacityReport) RiskAssessment {
	riskCounts := map[string]int{
		"low": 0, "medium": 0, "high": 0, "critical": 0,
	}

	for _, report := range tenantReports {
		riskCounts[report.RiskLevel]++
	}

	overallRisk := "low"
	if riskCounts["critical"] > 0 {
		overallRisk = "critical"
	} else if riskCounts["high"] > 0 {
		overallRisk = "high"
	} else if riskCounts["medium"] > 0 {
		overallRisk = "medium"
	}

	// Generate risk factors
	var riskFactors []RiskFactor
	if riskCounts["high"]+riskCounts["critical"] > 0 {
		riskFactors = append(riskFactors, RiskFactor{
			Factor:      "High Resource Utilization",
			Severity:    "high",
			Probability: 0.8,
			Impact:      "Service degradation or outages",
			Description: "Multiple tenants showing high resource utilization",
		})
	}

	return RiskAssessment{
		OverallRiskLevel: overallRisk,
		RiskFactors:      riskFactors,
		AlertThresholds: map[string]float64{
			"cpu_utilization":    80.0,
			"memory_utilization": 85.0,
			"error_rate":         5.0,
		},
	}
}

// generateTrendAnalysis generates trend analysis
func (p *Planner) generateTrendAnalysis(ctx context.Context, tenantNames []string, period ReportPeriod) TrendAnalysis {
	// Simplified trend analysis
	return TrendAnalysis{
		WeeklyTrends: []PeriodTrend{
			{
				Period:     "Week 1",
				StartDate:  period.StartDate,
				EndDate:    period.StartDate.AddDate(0, 0, 7),
				GrowthRate: 0.05,
				Direction:  "increasing",
				Confidence: 0.8,
			},
		},
		MonthlyTrends: []PeriodTrend{
			{
				Period:     "Current Month",
				StartDate:  period.StartDate,
				EndDate:    period.EndDate,
				GrowthRate: 0.12,
				Direction:  "increasing",
				Confidence: 0.75,
			},
		},
		SeasonalPatterns: []SeasonalPattern{
			{
				Pattern:     "Daily Peak",
				Description: "Higher usage during business hours",
				Impact:      0.3,
				Timing:      "9 AM - 5 PM",
			},
		},
		Anomalies: []TrendAnomaly{},
	}
}

// generateRecommendations generates overall recommendations
func (p *Planner) generateRecommendations(report *CapacityReport) []string {
	var recommendations []string

	if report.Summary.AverageUtilization > 80 {
		recommendations = append(recommendations, "🔴 URGENT: Average utilization is high - immediate scaling required")
	}

	if report.RiskAssessment.OverallRiskLevel == "critical" {
		recommendations = append(recommendations, "⚠️ CRITICAL: Multiple high-risk tenants detected - review immediately")
	}

	if len(report.Forecasting.ScalingRecommendations) > 0 {
		recommendations = append(recommendations, "📈 Scale cluster resources based on growth projections")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "✅ Capacity utilization is within acceptable ranges")
	}

	recommendations = append(recommendations, "📊 Schedule regular capacity reviews")
	recommendations = append(recommendations, "🔄 Monitor trends and adjust thresholds as needed")

	return recommendations
}

// generateTenantRecommendations generates recommendations for a specific tenant
func (p *Planner) generateTenantRecommendations(capacity *TenantCapacity, trend *UtilizationTrend, forecast TenantForecast, bottlenecks BottleneckAnalysis) []string {
	var recommendations []string

	if capacity.CPUUsage > 80 {
		recommendations = append(recommendations, "Increase CPU allocation")
	}
	if capacity.MemoryUsage > 80 {
		recommendations = append(recommendations, "Increase memory allocation")
	}
	if capacity.ErrorRate > 5 {
		recommendations = append(recommendations, "Investigate and reduce error rate")
	}
	if trend.GrowthRate > 0.2 {
		recommendations = append(recommendations, "Plan for rapid growth - consider preemptive scaling")
	}
	if bottlenecks.PrimaryBottleneck != "None" {
		recommendations = append(recommendations, fmt.Sprintf("Address %s bottleneck", bottlenecks.PrimaryBottleneck))
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Monitor current usage patterns")
	}

	return recommendations
}
