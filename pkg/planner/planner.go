package planner

import (
	"context"
	"fmt"
	"time"

	"github.com/akshaydubey29/mimirInsights/pkg/limits"
	"github.com/akshaydubey29/mimirInsights/pkg/metrics"
	"github.com/sirupsen/logrus"
)

// Planner handles capacity planning and forecasting
type Planner struct {
	metricsClient   *metrics.Client
	limitsAnalyzer  *limits.Analyzer
}

// CapacityReport represents a capacity planning report
type CapacityReport struct {
	TenantName       string                 `json:"tenant_name"`
	ReportType       string                 `json:"report_type"` // weekly, monthly
	GeneratedAt      time.Time              `json:"generated_at"`
	TimeRange        metrics.TimeRange      `json:"time_range"`
	CurrentUsage     UsageMetrics           `json:"current_usage"`
	Trends           TrendAnalysis          `json:"trends"`
	Projections      ProjectionData         `json:"projections"`
	Recommendations  []string               `json:"recommendations"`
	RiskAssessment   RiskAssessment         `json:"risk_assessment"`
	ResourceOptimization ResourceOptimization `json:"resource_optimization"`
}

// UsageMetrics represents current usage statistics
type UsageMetrics struct {
	IngestionRate    float64 `json:"ingestion_rate"`
	ActiveSeries     int64   `json:"active_series"`
	MemoryUsage      float64 `json:"memory_usage"`
	RejectedSamples  int64   `json:"rejected_samples"`
	LimitsReached    int64   `json:"limits_reached"`
}

// TrendAnalysis represents trend analysis over time
type TrendAnalysis struct {
	IngestionGrowthRate  float64 `json:"ingestion_growth_rate"`
	SeriesGrowthRate     float64 `json:"series_growth_rate"`
	MemoryGrowthRate     float64 `json:"memory_growth_rate"`
	RejectionTrend       string  `json:"rejection_trend"` // increasing, decreasing, stable
	SeasonalPatterns     []SeasonalPattern `json:"seasonal_patterns"`
}

// SeasonalPattern represents seasonal usage patterns
type SeasonalPattern struct {
	Pattern     string  `json:"pattern"` // daily, weekly, monthly
	PeakHours   []int   `json:"peak_hours"`
	PeakDays    []int   `json:"peak_days"`
	Multiplier  float64 `json:"multiplier"`
}

// ProjectionData represents future projections
type ProjectionData struct {
	NextWeek  UsageProjection `json:"next_week"`
	NextMonth UsageProjection `json:"next_month"`
	NextQuarter UsageProjection `json:"next_quarter"`
}

// UsageProjection represents projected usage
type UsageProjection struct {
	ProjectedIngestionRate float64 `json:"projected_ingestion_rate"`
	ProjectedActiveSeries  int64   `json:"projected_active_series"`
	ProjectedMemoryUsage   float64 `json:"projected_memory_usage"`
	ConfidenceLevel        float64 `json:"confidence_level"`
}

// RiskAssessment represents risk analysis
type RiskAssessment struct {
	OverallRisk     string   `json:"overall_risk"` // low, medium, high, critical
	RiskFactors     []string `json:"risk_factors"`
	MitigationSteps []string `json:"mitigation_steps"`
	AlertThresholds map[string]float64 `json:"alert_thresholds"`
}

// ResourceOptimization represents optimization recommendations
type ResourceOptimization struct {
	AlloyReplicas    int     `json:"recommended_alloy_replicas"`
	CPURequest       float64 `json:"recommended_cpu_request"`
	MemoryRequest    float64 `json:"recommended_memory_request"`
	StorageRequest   float64 `json:"recommended_storage_request"`
	CostOptimization []string `json:"cost_optimization_tips"`
}

// NewPlanner creates a new capacity planner
func NewPlanner(metricsClient *metrics.Client, limitsAnalyzer *limits.Analyzer) *Planner {
	return &Planner{
		metricsClient:  metricsClient,
		limitsAnalyzer: limitsAnalyzer,
	}
}

// GenerateWeeklyReport generates a weekly capacity report
func (p *Planner) GenerateWeeklyReport(ctx context.Context, tenantName string) (*CapacityReport, error) {
	return p.generateReport(ctx, tenantName, "weekly", 7*24*time.Hour)
}

// GenerateMonthlyReport generates a monthly capacity report
func (p *Planner) GenerateMonthlyReport(ctx context.Context, tenantName string) (*CapacityReport, error) {
	return p.generateReport(ctx, tenantName, "monthly", 30*24*time.Hour)
}

// generateReport generates a capacity report for the specified time range
func (p *Planner) generateReport(ctx context.Context, tenantName, reportType string, duration time.Duration) (*CapacityReport, error) {
	logrus.Infof("Generating %s capacity report for tenant: %s", reportType, tenantName)

	timeRange := metrics.CreateTimeRange(duration, "1h")
	
	// Get current metrics
	tenantMetrics, err := p.metricsClient.GetTenantMetrics(ctx, tenantName, timeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant metrics: %w", err)
	}

	// Analyze current usage
	currentUsage := p.analyzeCurrentUsage(tenantMetrics)

	// Perform trend analysis
	trends := p.analyzeTrends(tenantMetrics)

	// Generate projections
	projections := p.generateProjections(currentUsage, trends)

	// Assess risks
	riskAssessment := p.assessRisks(currentUsage, trends, projections)

	// Generate recommendations
	recommendations := p.generateRecommendations(currentUsage, trends, riskAssessment)

	// Optimize resources
	resourceOptimization := p.optimizeResources(currentUsage, projections)

	report := &CapacityReport{
		TenantName:           tenantName,
		ReportType:           reportType,
		GeneratedAt:          time.Now(),
		TimeRange:            timeRange,
		CurrentUsage:         currentUsage,
		Trends:               trends,
		Projections:          projections,
		Recommendations:      recommendations,
		RiskAssessment:       riskAssessment,
		ResourceOptimization: resourceOptimization,
	}

	return report, nil
}

// analyzeCurrentUsage analyzes current usage metrics
func (p *Planner) analyzeCurrentUsage(tenantMetrics *metrics.TenantMetrics) UsageMetrics {
	usage := UsageMetrics{}

	// Calculate current usage from metrics
	if ingestionSeries, exists := tenantMetrics.Metrics["ingestion_rate"]; exists && len(ingestionSeries) > 0 {
		// Get latest value
		if len(ingestionSeries[0].Values) > 0 {
			usage.IngestionRate = ingestionSeries[0].Values[len(ingestionSeries[0].Values)-1].Value
		}
	}

	if activeSeries, exists := tenantMetrics.Metrics["active_series"]; exists && len(activeSeries) > 0 {
		if len(activeSeries[0].Values) > 0 {
			usage.ActiveSeries = int64(activeSeries[0].Values[len(activeSeries[0].Values)-1].Value)
		}
	}

	if memorySeries, exists := tenantMetrics.Metrics["memory_usage"]; exists && len(memorySeries) > 0 {
		if len(memorySeries[0].Values) > 0 {
			usage.MemoryUsage = memorySeries[0].Values[len(memorySeries[0].Values)-1].Value
		}
	}

	if rejectedSeries, exists := tenantMetrics.Metrics["rejected_samples"]; exists && len(rejectedSeries) > 0 {
		if len(rejectedSeries[0].Values) > 0 {
			usage.RejectedSamples = int64(rejectedSeries[0].Values[len(rejectedSeries[0].Values)-1].Value)
		}
	}

	return usage
}

// analyzeTrends analyzes trends in the metrics
func (p *Planner) analyzeTrends(tenantMetrics *metrics.TenantMetrics) TrendAnalysis {
	trends := TrendAnalysis{
		RejectionTrend: "stable",
		SeasonalPatterns: []SeasonalPattern{
			{
				Pattern:    "daily",
				PeakHours:  []int{9, 10, 11, 14, 15, 16},
				Multiplier: 1.3,
			},
			{
				Pattern:   "weekly",
				PeakDays:  []int{1, 2, 3, 4, 5}, // Monday to Friday
				Multiplier: 1.2,
			},
		},
	}

	// Calculate growth rates (simplified)
	trends.IngestionGrowthRate = p.calculateGrowthRate(tenantMetrics, "ingestion_rate")
	trends.SeriesGrowthRate = p.calculateGrowthRate(tenantMetrics, "active_series")
	trends.MemoryGrowthRate = p.calculateGrowthRate(tenantMetrics, "memory_usage")

	// Determine rejection trend
	rejectionGrowth := p.calculateGrowthRate(tenantMetrics, "rejected_samples")
	if rejectionGrowth > 0.1 {
		trends.RejectionTrend = "increasing"
	} else if rejectionGrowth < -0.1 {
		trends.RejectionTrend = "decreasing"
	}

	return trends
}

// calculateGrowthRate calculates growth rate for a metric
func (p *Planner) calculateGrowthRate(tenantMetrics *metrics.TenantMetrics, metricName string) float64 {
	if series, exists := tenantMetrics.Metrics[metricName]; exists && len(series) > 0 {
		values := series[0].Values
		if len(values) >= 2 {
			first := values[0].Value
			last := values[len(values)-1].Value
			if first > 0 {
				return (last - first) / first
			}
		}
	}
	return 0.0
}

// generateProjections generates future projections
func (p *Planner) generateProjections(usage UsageMetrics, trends TrendAnalysis) ProjectionData {
	return ProjectionData{
		NextWeek: UsageProjection{
			ProjectedIngestionRate: usage.IngestionRate * (1 + trends.IngestionGrowthRate*0.1),
			ProjectedActiveSeries:  int64(float64(usage.ActiveSeries) * (1 + trends.SeriesGrowthRate*0.1)),
			ProjectedMemoryUsage:   usage.MemoryUsage * (1 + trends.MemoryGrowthRate*0.1),
			ConfidenceLevel:        0.85,
		},
		NextMonth: UsageProjection{
			ProjectedIngestionRate: usage.IngestionRate * (1 + trends.IngestionGrowthRate*0.4),
			ProjectedActiveSeries:  int64(float64(usage.ActiveSeries) * (1 + trends.SeriesGrowthRate*0.4)),
			ProjectedMemoryUsage:   usage.MemoryUsage * (1 + trends.MemoryGrowthRate*0.4),
			ConfidenceLevel:        0.70,
		},
		NextQuarter: UsageProjection{
			ProjectedIngestionRate: usage.IngestionRate * (1 + trends.IngestionGrowthRate*1.2),
			ProjectedActiveSeries:  int64(float64(usage.ActiveSeries) * (1 + trends.SeriesGrowthRate*1.2)),
			ProjectedMemoryUsage:   usage.MemoryUsage * (1 + trends.MemoryGrowthRate*1.2),
			ConfidenceLevel:        0.60,
		},
	}
}

// assessRisks assesses capacity risks
func (p *Planner) assessRisks(usage UsageMetrics, trends TrendAnalysis, projections ProjectionData) RiskAssessment {
	risk := RiskAssessment{
		OverallRisk: "low",
		RiskFactors: []string{},
		MitigationSteps: []string{},
		AlertThresholds: map[string]float64{
			"ingestion_rate": usage.IngestionRate * 0.8,
			"memory_usage":   usage.MemoryUsage * 0.8,
			"rejected_samples": float64(usage.RejectedSamples) * 1.5,
		},
	}

	// Assess ingestion growth risk
	if trends.IngestionGrowthRate > 0.5 {
		risk.OverallRisk = "high"
		risk.RiskFactors = append(risk.RiskFactors, "High ingestion growth rate")
		risk.MitigationSteps = append(risk.MitigationSteps, "Consider increasing ingestion rate limits")
	}

	// Assess memory growth risk
	if trends.MemoryGrowthRate > 0.3 {
		if risk.OverallRisk == "low" {
			risk.OverallRisk = "medium"
		}
		risk.RiskFactors = append(risk.RiskFactors, "Increasing memory usage trend")
		risk.MitigationSteps = append(risk.MitigationSteps, "Monitor memory usage and consider scaling")
	}

	// Assess rejection trend risk
	if trends.RejectionTrend == "increasing" {
		risk.OverallRisk = "high"
		risk.RiskFactors = append(risk.RiskFactors, "Increasing sample rejections")
		risk.MitigationSteps = append(risk.MitigationSteps, "Review and adjust tenant limits")
	}

	return risk
}

// generateRecommendations generates capacity recommendations
func (p *Planner) generateRecommendations(usage UsageMetrics, trends TrendAnalysis, risk RiskAssessment) []string {
	recommendations := []string{}

	if trends.IngestionGrowthRate > 0.2 {
		recommendations = append(recommendations, "Consider increasing ingestion rate limits by 20%")
	}

	if trends.SeriesGrowthRate > 0.3 {
		recommendations = append(recommendations, "Monitor series cardinality and consider optimization")
	}

	if usage.RejectedSamples > 1000 {
		recommendations = append(recommendations, "Investigate causes of sample rejections")
	}

	if risk.OverallRisk == "high" {
		recommendations = append(recommendations, "Immediate attention required - review all limits")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Current capacity appears adequate")
	}

	return recommendations
}

// optimizeResources generates resource optimization recommendations
func (p *Planner) optimizeResources(usage UsageMetrics, projections ProjectionData) ResourceOptimization {
	optimization := ResourceOptimization{
		AlloyReplicas:  2,
		CPURequest:     0.5,
		MemoryRequest:  1.0,
		StorageRequest: 10.0,
		CostOptimization: []string{
			"Consider using spot instances for non-critical workloads",
			"Implement data retention policies to reduce storage costs",
			"Optimize scrape intervals based on actual needs",
		},
	}

	// Adjust based on projections
	if projections.NextMonth.ProjectedIngestionRate > usage.IngestionRate*1.5 {
		optimization.AlloyReplicas = 3
		optimization.CPURequest = 1.0
		optimization.MemoryRequest = 2.0
	}

	if projections.NextMonth.ProjectedMemoryUsage > usage.MemoryUsage*2 {
		optimization.MemoryRequest = optimization.MemoryRequest * 1.5
		optimization.StorageRequest = optimization.StorageRequest * 1.5
	}

	return optimization
}

// ExportReport exports a capacity report in the specified format
func (p *Planner) ExportReport(report *CapacityReport, format string) ([]byte, error) {
	switch format {
	case "json":
		return p.exportJSON(report)
	case "csv":
		return p.exportCSV(report)
	case "markdown":
		return p.exportMarkdown(report)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportJSON exports report as JSON
func (p *Planner) exportJSON(report *CapacityReport) ([]byte, error) {
	// Implementation would marshal to JSON
	return []byte("{}"), nil
}

// exportCSV exports report as CSV
func (p *Planner) exportCSV(report *CapacityReport) ([]byte, error) {
	// Implementation would generate CSV format
	csv := fmt.Sprintf("Tenant,Report Type,Generated At,Ingestion Rate,Active Series,Memory Usage\n")
	csv += fmt.Sprintf("%s,%s,%s,%.2f,%d,%.2f\n",
		report.TenantName,
		report.ReportType,
		report.GeneratedAt.Format(time.RFC3339),
		report.CurrentUsage.IngestionRate,
		report.CurrentUsage.ActiveSeries,
		report.CurrentUsage.MemoryUsage,
	)
	return []byte(csv), nil
}

// exportMarkdown exports report as Markdown
func (p *Planner) exportMarkdown(report *CapacityReport) ([]byte, error) {
	markdown := fmt.Sprintf(`# Capacity Report: %s

**Report Type:** %s  
**Generated:** %s  
**Time Range:** %s to %s

## Current Usage
- **Ingestion Rate:** %.2f samples/sec
- **Active Series:** %d
- **Memory Usage:** %.2f GB
- **Rejected Samples:** %d

## Risk Assessment
**Overall Risk:** %s

### Risk Factors
%s

### Recommendations
%s
`,
		report.TenantName,
		report.ReportType,
		report.GeneratedAt.Format(time.RFC3339),
		report.TimeRange.Start.Format(time.RFC3339),
		report.TimeRange.End.Format(time.RFC3339),
		report.CurrentUsage.IngestionRate,
		report.CurrentUsage.ActiveSeries,
		report.CurrentUsage.MemoryUsage/1024/1024/1024,
		report.CurrentUsage.RejectedSamples,
		report.RiskAssessment.OverallRisk,
		p.formatList(report.RiskAssessment.RiskFactors),
		p.formatList(report.Recommendations),
	)

	return []byte(markdown), nil
}

// formatList formats a string slice as a markdown list
func (p *Planner) formatList(items []string) string {
	result := ""
	for _, item := range items {
		result += fmt.Sprintf("- %s\n", item)
	}
	return result
}