package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/akshaydubey29/mimirInsights/pkg/config"
	"github.com/sirupsen/logrus"
)

// Assistant provides LLM-based query assistance
type Assistant struct {
	config     *config.Config
	httpClient *http.Client
}

// QueryRequest represents an LLM query request
type QueryRequest struct {
	Question    string            `json:"question"`
	Context     map[string]interface{} `json:"context,omitempty"`
	TenantName  string            `json:"tenant_name,omitempty"`
	TimeRange   string            `json:"time_range,omitempty"`
}

// QueryResponse represents an LLM query response
type QueryResponse struct {
	Answer      string            `json:"answer"`
	Confidence  float64           `json:"confidence"`
	Sources     []string          `json:"sources"`
	Suggestions []string          `json:"suggestions"`
	Timestamp   time.Time         `json:"timestamp"`
}

// OpenAIRequest represents OpenAI API request format
type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse represents OpenAI API response format
type OpenAIResponse struct {
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a response choice
type Choice struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewAssistant creates a new LLM assistant
func NewAssistant() *Assistant {
	cfg := config.Get()
	
	return &Assistant{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Query processes a natural language query about metrics
func (a *Assistant) Query(ctx context.Context, request QueryRequest) (*QueryResponse, error) {
	if !a.config.LLM.Enabled {
		return &QueryResponse{
			Answer:     "LLM assistant is not enabled. Please enable it in the configuration.",
			Confidence: 0.0,
			Timestamp:  time.Now(),
		}, nil
	}

	logrus.Infof("Processing LLM query: %s", request.Question)

	// Build context-aware prompt
	prompt := a.buildPrompt(request)

	// Query the LLM
	response, err := a.queryLLM(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to query LLM: %w", err)
	}

	// Parse and enhance response
	queryResponse := a.parseResponse(response, request)

	return queryResponse, nil
}

// buildPrompt builds a context-aware prompt for the LLM
func (a *Assistant) buildPrompt(request QueryRequest) string {
	var prompt strings.Builder

	prompt.WriteString("You are MimirInsights AI Assistant, an expert in Grafana Mimir observability and metrics analysis.\n\n")
	
	prompt.WriteString("CONTEXT:\n")
	prompt.WriteString("- You help users understand Mimir metrics, tenant configurations, and observability issues\n")
	prompt.WriteString("- You can analyze ingestion rates, rejections, series cardinality, and resource usage\n")
	prompt.WriteString("- You provide actionable recommendations for limit optimization\n\n")

	if request.TenantName != "" {
		prompt.WriteString(fmt.Sprintf("TENANT: %s\n", request.TenantName))
	}

	if request.TimeRange != "" {
		prompt.WriteString(fmt.Sprintf("TIME RANGE: %s\n", request.TimeRange))
	}

	if request.Context != nil {
		prompt.WriteString("METRICS CONTEXT:\n")
		for key, value := range request.Context {
			prompt.WriteString(fmt.Sprintf("- %s: %v\n", key, value))
		}
	}

	prompt.WriteString("\nQUESTION: ")
	prompt.WriteString(request.Question)
	prompt.WriteString("\n\n")

	prompt.WriteString("Please provide a detailed, actionable answer that:\n")
	prompt.WriteString("1. Directly addresses the question\n")
	prompt.WriteString("2. References specific metrics when relevant\n")
	prompt.WriteString("3. Suggests concrete next steps\n")
	prompt.WriteString("4. Explains potential root causes\n")
	prompt.WriteString("5. Recommends monitoring or alerting improvements\n")

	return prompt.String()
}

// queryLLM queries the configured LLM provider
func (a *Assistant) queryLLM(ctx context.Context, prompt string) (string, error) {
	switch a.config.LLM.Provider {
	case "openai":
		return a.queryOpenAI(ctx, prompt)
	case "local":
		return a.queryLocal(ctx, prompt)
	default:
		return "", fmt.Errorf("unsupported LLM provider: %s", a.config.LLM.Provider)
	}
}

// queryOpenAI queries OpenAI API
func (a *Assistant) queryOpenAI(ctx context.Context, prompt string) (string, error) {
	if a.config.LLM.APIKey == "" {
		return "", fmt.Errorf("OpenAI API key not configured")
	}

	request := OpenAIRequest{
		Model: a.config.LLM.Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   a.config.LLM.MaxTokens,
		Temperature: 0.7,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := "https://api.openai.com/v1/chat/completions"
	if a.config.LLM.Endpoint != "" {
		endpoint = a.config.LLM.Endpoint
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.config.LLM.APIKey))

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API returned status %d", resp.StatusCode)
	}

	var response OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return response.Choices[0].Message.Content, nil
}

// queryLocal queries a local LLM endpoint
func (a *Assistant) queryLocal(ctx context.Context, prompt string) (string, error) {
	// Implementation for local LLM (e.g., Ollama, local GPT)
	// This would depend on the specific local LLM setup
	
	if a.config.LLM.Endpoint == "" {
		return "", fmt.Errorf("local LLM endpoint not configured")
	}

	// Simple example for a local endpoint
	requestBody := map[string]interface{}{
		"prompt":     prompt,
		"max_tokens": a.config.LLM.MaxTokens,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.config.LLM.Endpoint, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if text, ok := response["text"].(string); ok {
		return text, nil
	}

	return "", fmt.Errorf("unexpected response format")
}

// parseResponse parses and enhances the LLM response
func (a *Assistant) parseResponse(llmResponse string, request QueryRequest) *QueryResponse {
	response := &QueryResponse{
		Answer:    llmResponse,
		Timestamp: time.Now(),
	}

	// Calculate confidence based on response characteristics
	response.Confidence = a.calculateConfidence(llmResponse, request)

	// Extract sources (if any are mentioned)
	response.Sources = a.extractSources(llmResponse)

	// Generate follow-up suggestions
	response.Suggestions = a.generateSuggestions(request)

	return response
}

// calculateConfidence calculates confidence score for the response
func (a *Assistant) calculateConfidence(response string, request QueryRequest) float64 {
	confidence := 0.5 // Base confidence

	// Increase confidence if response mentions specific metrics
	metricKeywords := []string{"ingestion_rate", "active_series", "memory_usage", "rejected_samples"}
	for _, keyword := range metricKeywords {
		if strings.Contains(strings.ToLower(response), keyword) {
			confidence += 0.1
		}
	}

	// Increase confidence if response provides specific numbers
	if strings.Contains(response, "%") || strings.Contains(response, "samples/sec") {
		confidence += 0.1
	}

	// Increase confidence if context was provided
	if request.Context != nil && len(request.Context) > 0 {
		confidence += 0.2
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// extractSources extracts potential sources mentioned in the response
func (a *Assistant) extractSources(response string) []string {
	sources := []string{}

	// Look for metric names that might be sources
	metricPatterns := []string{
		"cortex_distributor_",
		"cortex_ingester_",
		"cortex_querier_",
		"process_",
	}

	for _, pattern := range metricPatterns {
		if strings.Contains(response, pattern) {
			sources = append(sources, "Mimir Metrics")
			break
		}
	}

	// Look for configuration references
	if strings.Contains(response, "ConfigMap") || strings.Contains(response, "runtime-overrides") {
		sources = append(sources, "Mimir Configuration")
	}

	// Look for Kubernetes references
	if strings.Contains(response, "Pod") || strings.Contains(response, "Deployment") {
		sources = append(sources, "Kubernetes Resources")
	}

	return sources
}

// generateSuggestions generates follow-up suggestions
func (a *Assistant) generateSuggestions(request QueryRequest) []string {
	suggestions := []string{}

	// Generate suggestions based on question type
	question := strings.ToLower(request.Question)

	if strings.Contains(question, "rejection") || strings.Contains(question, "rejected") {
		suggestions = append(suggestions, "Check current ingestion rate limits")
		suggestions = append(suggestions, "Review series cardinality for the tenant")
		suggestions = append(suggestions, "Analyze ingestion patterns over time")
	}

	if strings.Contains(question, "memory") {
		suggestions = append(suggestions, "Monitor memory usage trends")
		suggestions = append(suggestions, "Check for memory leaks in applications")
		suggestions = append(suggestions, "Review series retention policies")
	}

	if strings.Contains(question, "ingestion") {
		suggestions = append(suggestions, "Examine ingestion rate patterns")
		suggestions = append(suggestions, "Check Alloy configuration")
		suggestions = append(suggestions, "Review application metrics generation")
	}

	// Default suggestions if none match
	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Check the tenant's metrics dashboard")
		suggestions = append(suggestions, "Review recent configuration changes")
		suggestions = append(suggestions, "Monitor alerts and notifications")
	}

	return suggestions
}

// GetSampleQuestions returns sample questions users can ask
func (a *Assistant) GetSampleQuestions() []string {
	return []string{
		"Why did ingestion rejection spike for tenant eats yesterday?",
		"What's causing high memory usage in the transportation tenant?",
		"How can I optimize series cardinality for my tenant?",
		"Why are my metrics being rejected by Mimir?",
		"What's the recommended ingestion rate limit for my usage pattern?",
		"How do I troubleshoot Alloy configuration issues?",
		"What's causing the increase in active series count?",
		"How can I reduce memory usage in my Mimir deployment?",
		"Why is my tenant hitting rate limits?",
		"What are the best practices for metric labeling?",
	}
}

// IsEnabled returns whether the LLM assistant is enabled
func (a *Assistant) IsEnabled() bool {
	return a.config.LLM.Enabled
}