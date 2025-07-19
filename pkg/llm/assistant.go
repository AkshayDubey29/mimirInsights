package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/akshaydubey29/mimirInsights/pkg/config"
	"github.com/akshaydubey29/mimirInsights/pkg/metrics"
	"github.com/sirupsen/logrus"
)

// Assistant provides LLM-based assistance for metrics interpretation
type Assistant struct {
	config        *config.Config
	metricsClient *metrics.Client
	llmClient     LLMClient
}

// LLMClient interface for different LLM providers
type LLMClient interface {
	GenerateResponse(ctx context.Context, prompt string, maxTokens int) (*LLMResponse, error)
	IsEnabled() bool
}

// LLMResponse represents a response from the LLM
type LLMResponse struct {
	Content     string    `json:"content"`
	Confidence  float64   `json:"confidence"`
	TokensUsed  int       `json:"tokens_used"`
	Model       string    `json:"model"`
	GeneratedAt time.Time `json:"generated_at"`
}

// QueryRequest represents a user query to the assistant
type QueryRequest struct {
	Query       string            `json:"query" binding:"required"`
	Context     QueryContext      `json:"context"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// QueryContext provides context for the query
type QueryContext struct {
	TenantName  string    `json:"tenant_name,omitempty"`
	TimeRange   string    `json:"time_range,omitempty"`
	MetricTypes []string  `json:"metric_types,omitempty"`
	IncludeLogs bool      `json:"include_logs"`
	Timestamp   time.Time `json:"timestamp,omitempty"`
}

// AssistantResponse represents the assistant's response
type AssistantResponse struct {
	Answer           string           `json:"answer"`
	Confidence       float64          `json:"confidence"`
	Sources          []MetricSource   `json:"sources"`
	Recommendations  []string         `json:"recommendations"`
	RelatedQueries   []string         `json:"related_queries"`
	Analysis         MetricAnalysis   `json:"analysis"`
	ResponseMetadata ResponseMetadata `json:"metadata"`
}

// MetricSource represents a source of information used in the response
type MetricSource struct {
	Type        string    `json:"type"` // "metric", "log", "config", "trend"
	Name        string    `json:"name"`
	Value       string    `json:"value"`
	Timestamp   time.Time `json:"timestamp"`
	Relevance   float64   `json:"relevance"`
	Description string    `json:"description"`
}

// MetricAnalysis provides detailed analysis of metrics
type MetricAnalysis struct {
	TrendDirection   string              `json:"trend_direction"`
	Anomalies        []AnomalyDetection  `json:"anomalies"`
	Correlations     []MetricCorrelation `json:"correlations"`
	RootCause        string              `json:"root_cause,omitempty"`
	Severity         string              `json:"severity"`
	ImpactAssessment string              `json:"impact_assessment"`
}

// AnomalyDetection represents detected anomalies
type AnomalyDetection struct {
	MetricName    string    `json:"metric_name"`
	AnomalyType   string    `json:"anomaly_type"` // "spike", "drop", "pattern_break"
	Timestamp     time.Time `json:"timestamp"`
	Severity      string    `json:"severity"`
	ExpectedValue float64   `json:"expected_value"`
	ActualValue   float64   `json:"actual_value"`
	Deviation     float64   `json:"deviation"`
	Description   string    `json:"description"`
}

// MetricCorrelation represents correlations between metrics
type MetricCorrelation struct {
	Metric1     string  `json:"metric1"`
	Metric2     string  `json:"metric2"`
	Correlation float64 `json:"correlation"`
	Strength    string  `json:"strength"`
	Description string  `json:"description"`
}

// ResponseMetadata contains metadata about the response
type ResponseMetadata struct {
	ProcessingTime  time.Duration `json:"processing_time"`
	TokensUsed      int           `json:"tokens_used"`
	Model           string        `json:"model"`
	QueryComplexity string        `json:"query_complexity"`
	DataPoints      int           `json:"data_points"`
	GeneratedAt     time.Time     `json:"generated_at"`
}

// NewAssistant creates a new LLM assistant
func NewAssistant() (*Assistant, error) {
	cfg := config.Get()
	var llmClient LLMClient
	var err error

	switch cfg.LLM.Provider {
	case "openai":
		llmClient, err = NewOpenAIClient()
	case "anthropic":
		llmClient, err = NewAnthropicClient()
	case "ollama":
		llmClient, err = NewOllamaClient()
	default:
		logrus.Warnf("Unknown LLM provider: %s, no LLM integration available", cfg.LLM.Provider)
		return &Assistant{
			config:        cfg,
			metricsClient: nil,
			llmClient:     nil,
		}, nil
	}

	if err != nil {
		logrus.Warnf("Failed to initialize LLM client: %v", err)
		return &Assistant{
			config:        cfg,
			metricsClient: nil,
			llmClient:     nil,
		}, nil
	}

	return &Assistant{
		config:        cfg,
		metricsClient: nil, // TODO: Initialize metrics client
		llmClient:     llmClient,
	}, nil
}

// ProcessQuery processes a natural language query about metrics
func (a *Assistant) ProcessQuery(ctx context.Context, req QueryRequest) (*AssistantResponse, error) {
	start := time.Now()

	logrus.Infof("Processing LLM query: %s", req.Query)

	// Analyze query intent and extract context
	queryIntent := a.analyzeQueryIntent(req.Query)

	// Gather relevant metrics and context
	metricSources, err := a.gatherRelevantData(ctx, req, queryIntent)
	if err != nil {
		return nil, fmt.Errorf("failed to gather relevant data: %w", err)
	}

	// Build context-aware prompt
	prompt := a.buildPrompt(req, metricSources, queryIntent)

	// Get LLM response
	llmResponse, err := a.llmClient.GenerateResponse(ctx, prompt, req.MaxTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM response: %w", err)
	}

	// Analyze metrics for additional insights
	analysis := a.analyzeMetrics(metricSources)

	// Generate recommendations
	recommendations := a.generateRecommendations(req, metricSources, analysis)

	// Generate related queries
	relatedQueries := a.generateRelatedQueries(req.Query, queryIntent)

	response := &AssistantResponse{
		Answer:          llmResponse.Content,
		Confidence:      llmResponse.Confidence,
		Sources:         metricSources,
		Recommendations: recommendations,
		RelatedQueries:  relatedQueries,
		Analysis:        analysis,
		ResponseMetadata: ResponseMetadata{
			ProcessingTime:  time.Since(start),
			TokensUsed:      llmResponse.TokensUsed,
			Model:           llmResponse.Model,
			QueryComplexity: a.assessQueryComplexity(req.Query),
			DataPoints:      len(metricSources),
			GeneratedAt:     time.Now(),
		},
	}

	logrus.Infof("Processed LLM query in %v, used %d tokens",
		response.ResponseMetadata.ProcessingTime, response.ResponseMetadata.TokensUsed)

	return response, nil
}

// analyzeQueryIntent analyzes the intent behind a user query
func (a *Assistant) analyzeQueryIntent(query string) QueryIntent {
	queryLower := strings.ToLower(query)

	// Define intent patterns
	intentPatterns := map[string][]string{
		"troubleshooting": {"why", "error", "failure", "issue", "problem", "spike", "drop", "rejection"},
		"trending":        {"trend", "increase", "decrease", "growing", "declining", "over time"},
		"comparison":      {"compare", "vs", "versus", "difference", "between"},
		"prediction":      {"predict", "forecast", "future", "will", "expect"},
		"optimization":    {"optimize", "improve", "better", "reduce", "increase"},
		"explanation":     {"what", "how", "explain", "describe", "tell me about"},
	}

	intent := QueryIntent{
		Type:       "explanation", // default
		Confidence: 0.5,
		Keywords:   []string{},
	}

	// Match patterns
	maxMatches := 0
	for intentType, patterns := range intentPatterns {
		matches := 0
		for _, pattern := range patterns {
			if strings.Contains(queryLower, pattern) {
				matches++
				intent.Keywords = append(intent.Keywords, pattern)
			}
		}
		if matches > maxMatches {
			maxMatches = matches
			intent.Type = intentType
			intent.Confidence = float64(matches) / float64(len(patterns))
		}
	}

	// Extract time references
	timeKeywords := []string{"yesterday", "today", "last hour", "last week", "last month"}
	for _, keyword := range timeKeywords {
		if strings.Contains(queryLower, keyword) {
			intent.TimeReference = keyword
			break
		}
	}

	// Extract metric references
	metricKeywords := []string{"ingestion", "series", "memory", "cpu", "latency", "errors", "rejection"}
	for _, keyword := range metricKeywords {
		if strings.Contains(queryLower, keyword) {
			intent.MetricTypes = append(intent.MetricTypes, keyword)
		}
	}

	return intent
}

// QueryIntent represents the analyzed intent of a query
type QueryIntent struct {
	Type          string   `json:"type"`
	Confidence    float64  `json:"confidence"`
	Keywords      []string `json:"keywords"`
	TimeReference string   `json:"time_reference"`
	MetricTypes   []string `json:"metric_types"`
}

// gatherRelevantData gathers relevant metrics and context data
func (a *Assistant) gatherRelevantData(ctx context.Context, req QueryRequest, intent QueryIntent) ([]MetricSource, error) {
	var sources []MetricSource

	// Determine time range
	timeRange := a.determineTimeRange(req.Context.TimeRange, intent.TimeReference)

	// Get relevant metrics based on intent
	metricNames := a.selectRelevantMetrics(intent.MetricTypes, req.Context.TenantName)

	for _, metricName := range metricNames {
		tenantMetrics, err := a.metricsClient.GetTenantMetrics(ctx, req.Context.TenantName, timeRange)
		if err != nil {
			logrus.Warnf("Failed to get metrics for %s: %v", metricName, err)
			continue
		}

		// Convert metrics to sources
		metricSources := a.convertMetricsToSources(tenantMetrics, metricName)
		sources = append(sources, metricSources...)
	}

	// Add synthetic data if no real metrics available
	if len(sources) == 0 {
		sources = a.generateSyntheticSources(req.Context.TenantName, intent)
	}

	return sources, nil
}

// determineTimeRange determines the appropriate time range for the query
func (a *Assistant) determineTimeRange(contextTimeRange, timeReference string) metrics.TimeRange {
	if contextTimeRange != "" {
		// For now, return a basic time range - in production this would parse the string
		return metrics.CreateTimeRange(24*time.Hour, "5m")
	}

	// Default based on time reference
	switch timeReference {
	case "yesterday":
		return metrics.CreateTimeRange(24*time.Hour, "5m")
	case "last hour":
		return metrics.CreateTimeRange(time.Hour, "1m")
	case "last week":
		return metrics.CreateTimeRange(7*24*time.Hour, "1h")
	default:
		return metrics.CreateTimeRange(24*time.Hour, "5m")
	}
}

// selectRelevantMetrics selects relevant metrics based on intent
func (a *Assistant) selectRelevantMetrics(metricTypes []string, tenantName string) []string {
	if len(metricTypes) == 0 {
		// Default metrics for general queries
		return []string{"ingestion_rate", "active_series", "rejected_samples"}
	}

	var metrics []string
	for _, metricType := range metricTypes {
		switch metricType {
		case "ingestion":
			metrics = append(metrics, "ingestion_rate", "samples_per_second")
		case "series":
			metrics = append(metrics, "active_series", "series_count")
		case "memory":
			metrics = append(metrics, "memory_usage", "memory_utilization")
		case "cpu":
			metrics = append(metrics, "cpu_usage", "cpu_utilization")
		case "errors", "rejection":
			metrics = append(metrics, "rejected_samples", "error_rate")
		case "latency":
			metrics = append(metrics, "query_duration", "write_latency")
		}
	}

	if len(metrics) == 0 {
		metrics = []string{"ingestion_rate", "active_series"}
	}

	return metrics
}

// convertMetricsToSources converts metrics to MetricSource objects
func (a *Assistant) convertMetricsToSources(tenantMetrics *metrics.TenantMetrics, metricName string) []MetricSource {
	var sources []MetricSource

	if metricSeries, exists := tenantMetrics.Metrics[metricName]; exists {
		for _, series := range metricSeries {
			for i, value := range series.Values {
				if i >= 5 { // Limit to recent values
					break
				}
				sources = append(sources, MetricSource{
					Type:        "metric",
					Name:        metricName,
					Value:       fmt.Sprintf("%.2f", value.Value),
					Timestamp:   value.Timestamp,
					Relevance:   0.8 - float64(i)*0.1, // Decreasing relevance for older values
					Description: fmt.Sprintf("%s value at %s", metricName, value.Timestamp.Format("15:04:05")),
				})
			}
		}
	}

	return sources
}

// generateSyntheticSources generates synthetic data sources for demonstration
func (a *Assistant) generateSyntheticSources(tenantName string, intent QueryIntent) []MetricSource {
	now := time.Now()
	sources := []MetricSource{
		{
			Type:        "metric",
			Name:        "ingestion_rate",
			Value:       "1250.5",
			Timestamp:   now.Add(-5 * time.Minute),
			Relevance:   0.9,
			Description: "Current ingestion rate in samples/sec",
		},
		{
			Type:        "metric",
			Name:        "rejected_samples",
			Value:       "45",
			Timestamp:   now.Add(-2 * time.Minute),
			Relevance:   0.8,
			Description: "Recent rejected samples count",
		},
		{
			Type:        "trend",
			Name:        "ingestion_trend",
			Value:       "increasing",
			Timestamp:   now,
			Relevance:   0.7,
			Description: "Ingestion rate trend over last hour",
		},
	}

	// Add specific sources based on intent
	if intent.Type == "troubleshooting" {
		sources = append(sources, MetricSource{
			Type:        "log",
			Name:        "error_logs",
			Value:       "rate limit exceeded for tenant",
			Timestamp:   now.Add(-10 * time.Minute),
			Relevance:   0.9,
			Description: "Error log entry related to the issue",
		})
	}

	return sources
}

// buildPrompt builds a context-aware prompt for the LLM
func (a *Assistant) buildPrompt(req QueryRequest, sources []MetricSource, intent QueryIntent) string {
	prompt := fmt.Sprintf(`You are a Mimir monitoring expert assistant. Analyze the following metrics and provide insights.

User Query: %s
Query Intent: %s
Tenant: %s

Current Metrics Data:
`, req.Query, intent.Type, req.Context.TenantName)

	// Add metrics context
	for _, source := range sources {
		prompt += fmt.Sprintf("- %s: %s (%s) - %s\n",
			source.Name, source.Value, source.Timestamp.Format("15:04"), source.Description)
	}

	prompt += `
Please provide:
1. A clear explanation of what the metrics indicate
2. Analysis of any patterns, trends, or anomalies
3. Potential root causes if issues are detected
4. Actionable recommendations for improvement
5. Keep the response concise and practical

Focus on being helpful and accurate based on the provided metrics data.`

	return prompt
}

// analyzeMetrics performs analysis on the gathered metrics
func (a *Assistant) analyzeMetrics(sources []MetricSource) MetricAnalysis {
	analysis := MetricAnalysis{
		TrendDirection:   "stable",
		Anomalies:        []AnomalyDetection{},
		Correlations:     []MetricCorrelation{},
		Severity:         "low",
		ImpactAssessment: "minimal impact detected",
	}

	// Analyze trends and anomalies (simplified implementation)
	ingestionValues := []float64{}
	rejectionValues := []float64{}

	for _, source := range sources {
		if source.Type == "metric" {
			if value, err := parseFloat(source.Value); err == nil {
				switch source.Name {
				case "ingestion_rate":
					ingestionValues = append(ingestionValues, value)
				case "rejected_samples":
					rejectionValues = append(rejectionValues, value)
				}
			}
		}
	}

	// Simple trend analysis
	if len(ingestionValues) > 1 {
		if ingestionValues[0] > ingestionValues[len(ingestionValues)-1]*1.1 {
			analysis.TrendDirection = "increasing"
		} else if ingestionValues[0] < ingestionValues[len(ingestionValues)-1]*0.9 {
			analysis.TrendDirection = "decreasing"
		}
	}

	// Simple anomaly detection
	if len(rejectionValues) > 0 {
		avgRejections := calculateAverage(rejectionValues)
		if avgRejections > 100 {
			analysis.Anomalies = append(analysis.Anomalies, AnomalyDetection{
				MetricName:    "rejected_samples",
				AnomalyType:   "spike",
				Timestamp:     time.Now(),
				Severity:      "high",
				ExpectedValue: 10,
				ActualValue:   avgRejections,
				Deviation:     (avgRejections - 10) / 10,
				Description:   "Unusually high rejection rate detected",
			})
			analysis.Severity = "high"
			analysis.ImpactAssessment = "significant impact on data ingestion"
		}
	}

	return analysis
}

// generateRecommendations generates actionable recommendations
func (a *Assistant) generateRecommendations(req QueryRequest, sources []MetricSource, analysis MetricAnalysis) []string {
	var recommendations []string

	// Base recommendations on analysis
	if analysis.Severity == "high" {
		recommendations = append(recommendations, "⚠️ Immediate attention required - high severity issues detected")
	}

	if analysis.TrendDirection == "increasing" {
		recommendations = append(recommendations, "📈 Monitor increasing trend - consider scaling if sustained")
	}

	if len(analysis.Anomalies) > 0 {
		recommendations = append(recommendations, "🔍 Investigate detected anomalies for root cause")
	}

	// Default recommendations
	if len(recommendations) == 0 {
		recommendations = append(recommendations,
			"📊 Continue monitoring current metrics patterns",
			"🔄 Set up alerts for threshold breaches",
			"📈 Review historical trends for context")
	}

	return recommendations
}

// generateRelatedQueries generates related queries the user might ask
func (a *Assistant) generateRelatedQueries(originalQuery string, intent QueryIntent) []string {
	baseQueries := []string{
		"What are the current resource limits for this tenant?",
		"How does this compare to historical patterns?",
		"What alerts should I set up for monitoring?",
		"Are there any configuration optimizations recommended?",
	}

	// Customize based on intent
	switch intent.Type {
	case "troubleshooting":
		return []string{
			"What caused this issue?",
			"How can I prevent this in the future?",
			"What's the impact on other tenants?",
			"What configuration changes might help?",
		}
	case "trending":
		return []string{
			"What's driving this trend?",
			"When did this trend start?",
			"How does this compare to other tenants?",
			"Should I be concerned about this trend?",
		}
	case "optimization":
		return []string{
			"What configuration changes would help?",
			"How can I reduce resource usage?",
			"What are the best practices for this scenario?",
			"What monitoring should I add?",
		}
	}

	return baseQueries
}

// assessQueryComplexity assesses the complexity of a query
func (a *Assistant) assessQueryComplexity(query string) string {
	words := strings.Fields(query)
	wordCount := len(words)

	if wordCount < 5 {
		return "simple"
	} else if wordCount < 15 {
		return "medium"
	}
	return "complex"
}

// Helper functions
func parseFloat(s string) (float64, error) {
	// Simple float parsing, could be enhanced
	var value float64
	_, err := fmt.Sscanf(s, "%f", &value)
	return value, err
}

func calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// GetAssistantCapabilities returns the capabilities of the assistant
func (a *Assistant) GetAssistantCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"enabled": a.llmClient.IsEnabled(),
		"supported_queries": []string{
			"Metrics analysis and interpretation",
			"Troubleshooting assistance",
			"Trend analysis and forecasting",
			"Root cause analysis",
			"Configuration recommendations",
			"Capacity planning insights",
		},
		"supported_intents": []string{
			"troubleshooting", "trending", "comparison",
			"prediction", "optimization", "explanation",
		},
		"model_info": map[string]interface{}{
			"provider":   a.config.LLM.Provider,
			"model":      a.config.LLM.Model,
			"max_tokens": a.config.LLM.MaxTokens,
		},
	}
}
