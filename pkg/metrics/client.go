package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/akshaydubey29/mimirInsights/pkg/config"
	"github.com/sirupsen/logrus"
)

// Client handles metrics queries to Mimir
type Client struct {
	baseURL    string
	httpClient *http.Client
	config     *config.Config
}

// MetricQuery represents a Prometheus query
type MetricQuery struct {
	Query  string    `json:"query"`
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
	Step   string    `json:"step"`
	Tenant string    `json:"tenant,omitempty"`
}

// MetricResponse represents a Prometheus query response
type MetricResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Values [][]interface{}   `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

// MetricValue represents a single metric value
type MetricValue struct {
	Timestamp time.Time
	Value     float64
	Labels    map[string]string
}

// MetricSeries represents a series of metric values
type MetricSeries struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
	Values []MetricValue     `json:"values"`
}

// TenantMetrics represents metrics for a specific tenant
type TenantMetrics struct {
	TenantName string                    `json:"tenant_name"`
	Metrics    map[string][]MetricSeries `json:"metrics"`
	TimeRange  TimeRange                 `json:"time_range"`
}

// TimeRange represents a time range for metrics
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Step  string    `json:"step"`
}

// NewClient creates a new metrics client
func NewClient() *Client {
	cfg := config.Get()

	client := &http.Client{
		Timeout: time.Duration(cfg.Mimir.Timeout) * time.Second,
	}

	return &Client{
		baseURL:    cfg.Mimir.APIURL,
		httpClient: client,
		config:     cfg,
	}
}

// QueryMetrics executes a Prometheus query
func (c *Client) QueryMetrics(ctx context.Context, query MetricQuery) (*MetricResponse, error) {
	// Build query URL - use the correct Mimir API path
	u, err := url.Parse(c.baseURL + "/prometheus/api/v1/query_range")
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	q.Set("query", query.Query)
	q.Set("start", query.Start.Format(time.RFC3339))
	q.Set("end", query.End.Format(time.RFC3339))
	q.Set("step", query.Step)

	u.RawQuery = q.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add X-Scope-OrgID header for multi-tenant Mimir
	if c.config.Mimir.OrgID != "" {
		req.Header.Set("X-Scope-OrgID", c.config.Mimir.OrgID)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var metricResp MetricResponse
	if err := json.NewDecoder(resp.Body).Decode(&metricResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if metricResp.Status != "success" {
		return nil, fmt.Errorf("query failed with status: %s", metricResp.Status)
	}

	return &metricResp, nil
}

// GetIngestionRate gets the ingestion rate for a tenant
func (c *Client) GetIngestionRate(ctx context.Context, tenant string, timeRange TimeRange) ([]MetricSeries, error) {
	query := MetricQuery{
		Query:  fmt.Sprintf(`cortex_distributor_ingestion_rate{tenant="%s"}`, tenant),
		Start:  timeRange.Start,
		End:    timeRange.End,
		Step:   timeRange.Step,
		Tenant: tenant,
	}

	resp, err := c.QueryMetrics(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query ingestion rate: %w", err)
	}

	return c.parseMetricResponse(resp, "ingestion_rate"), nil
}

// GetRejectedSamples gets the rejected samples count for a tenant
func (c *Client) GetRejectedSamples(ctx context.Context, tenant string, timeRange TimeRange) ([]MetricSeries, error) {
	query := MetricQuery{
		Query:  fmt.Sprintf(`cortex_distributor_rejected_samples_total{tenant="%s"}`, tenant),
		Start:  timeRange.Start,
		End:    timeRange.End,
		Step:   timeRange.Step,
		Tenant: tenant,
	}

	resp, err := c.QueryMetrics(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rejected samples: %w", err)
	}

	return c.parseMetricResponse(resp, "rejected_samples"), nil
}

// GetTenantLimitsReached gets the tenant limits reached count
func (c *Client) GetTenantLimitsReached(ctx context.Context, tenant string, timeRange TimeRange) ([]MetricSeries, error) {
	query := MetricQuery{
		Query:  fmt.Sprintf(`cortex_distributor_tenant_limits_reached_total{tenant="%s"}`, tenant),
		Start:  timeRange.Start,
		End:    timeRange.End,
		Step:   timeRange.Step,
		Tenant: tenant,
	}

	resp, err := c.QueryMetrics(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tenant limits reached: %w", err)
	}

	return c.parseMetricResponse(resp, "limits_reached"), nil
}

// GetActiveSeries gets the active series count for a tenant
func (c *Client) GetActiveSeries(ctx context.Context, tenant string, timeRange TimeRange) ([]MetricSeries, error) {
	query := MetricQuery{
		Query:  fmt.Sprintf(`cortex_ingester_active_series{tenant="%s"}`, tenant),
		Start:  timeRange.Start,
		End:    timeRange.End,
		Step:   timeRange.Step,
		Tenant: tenant,
	}

	resp, err := c.QueryMetrics(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active series: %w", err)
	}

	return c.parseMetricResponse(resp, "active_series"), nil
}

// GetMemoryUsage gets the memory usage for a tenant
func (c *Client) GetMemoryUsage(ctx context.Context, tenant string, timeRange TimeRange) ([]MetricSeries, error) {
	query := MetricQuery{
		Query:  fmt.Sprintf(`cortex_ingester_memory_usage_bytes{tenant="%s"}`, tenant),
		Start:  timeRange.Start,
		End:    timeRange.End,
		Step:   timeRange.Step,
		Tenant: tenant,
	}

	resp, err := c.QueryMetrics(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query memory usage: %w", err)
	}

	return c.parseMetricResponse(resp, "memory_usage"), nil
}

// GetProcessMemory gets the process memory usage
func (c *Client) GetProcessMemory(ctx context.Context, timeRange TimeRange) ([]MetricSeries, error) {
	query := MetricQuery{
		Query: "process_resident_memory_bytes",
		Start: timeRange.Start,
		End:   timeRange.End,
		Step:  timeRange.Step,
	}

	resp, err := c.QueryMetrics(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query process memory: %w", err)
	}

	return c.parseMetricResponse(resp, "process_memory"), nil
}

// GetTenantMetrics gets all relevant metrics for a tenant
func (c *Client) GetTenantMetrics(ctx context.Context, tenant string, timeRange TimeRange) (*TenantMetrics, error) {
	logrus.Infof("Getting metrics for tenant: %s", tenant)

	metrics := &TenantMetrics{
		TenantName: tenant,
		TimeRange:  timeRange,
		Metrics:    make(map[string][]MetricSeries),
	}

	// Get ingestion rate
	ingestionRate, err := c.GetIngestionRate(ctx, tenant, timeRange)
	if err != nil {
		logrus.Warnf("Failed to get ingestion rate for %s: %v", tenant, err)
	} else {
		metrics.Metrics["ingestion_rate"] = ingestionRate
	}

	// Get rejected samples
	rejectedSamples, err := c.GetRejectedSamples(ctx, tenant, timeRange)
	if err != nil {
		logrus.Warnf("Failed to get rejected samples for %s: %v", tenant, err)
	} else {
		metrics.Metrics["rejected_samples"] = rejectedSamples
	}

	// Get tenant limits reached
	limitsReached, err := c.GetTenantLimitsReached(ctx, tenant, timeRange)
	if err != nil {
		logrus.Warnf("Failed to get limits reached for %s: %v", tenant, err)
	} else {
		metrics.Metrics["limits_reached"] = limitsReached
	}

	// Get active series
	activeSeries, err := c.GetActiveSeries(ctx, tenant, timeRange)
	if err != nil {
		logrus.Warnf("Failed to get active series for %s: %v", tenant, err)
	} else {
		metrics.Metrics["active_series"] = activeSeries
	}

	// Get memory usage
	memoryUsage, err := c.GetMemoryUsage(ctx, tenant, timeRange)
	if err != nil {
		logrus.Warnf("Failed to get memory usage for %s: %v", tenant, err)
	} else {
		metrics.Metrics["memory_usage"] = memoryUsage
	}

	return metrics, nil
}

// GetPeakValues gets the peak values for metrics over a time range
func (c *Client) GetPeakValues(ctx context.Context, tenant string, timeRange TimeRange) (map[string]float64, error) {
	peakValues := make(map[string]float64)

	// Get all metrics
	tenantMetrics, err := c.GetTenantMetrics(ctx, tenant, timeRange)
	if err != nil {
		return nil, err
	}

	// Calculate peak values for each metric type
	for metricType, series := range tenantMetrics.Metrics {
		var maxValue float64
		for _, s := range series {
			for _, v := range s.Values {
				if v.Value > maxValue {
					maxValue = v.Value
				}
			}
		}
		peakValues[metricType] = maxValue
	}

	return peakValues, nil
}

// parseMetricResponse converts a MetricResponse to MetricSeries
func (c *Client) parseMetricResponse(resp *MetricResponse, metricName string) []MetricSeries {
	var series []MetricSeries

	for _, result := range resp.Data.Result {
		s := MetricSeries{
			Name:   metricName,
			Labels: result.Metric,
			Values: make([]MetricValue, 0, len(result.Values)),
		}

		for _, value := range result.Values {
			if len(value) >= 2 {
				timestamp, _ := value[0].(float64)
				val, _ := value[1].(string)

				if valFloat, err := strconv.ParseFloat(val, 64); err == nil {
					s.Values = append(s.Values, MetricValue{
						Timestamp: time.Unix(int64(timestamp), 0),
						Value:     valFloat,
						Labels:    result.Metric,
					})
				}
			}
		}

		series = append(series, s)
	}

	return series
}

// CreateTimeRange creates a TimeRange for the specified duration
func CreateTimeRange(duration time.Duration, step string) TimeRange {
	end := time.Now()
	start := end.Add(-duration)

	return TimeRange{
		Start: start,
		End:   end,
		Step:  step,
	}
}

// GetStandardTimeRanges returns standard time ranges for analysis
func GetStandardTimeRanges() map[string]TimeRange {
	return map[string]TimeRange{
		"48h": CreateTimeRange(48*time.Hour, "5m"),
		"7d":  CreateTimeRange(7*24*time.Hour, "15m"),
		"30d": CreateTimeRange(30*24*time.Hour, "1h"),
		"60d": CreateTimeRange(60*24*time.Hour, "2h"),
	}
}

// GetRealMetricsFromEndpoints collects real metrics from all discovered Mimir endpoints
func (c *Client) GetRealMetricsFromEndpoints(ctx context.Context, endpoints []string, timeRange TimeRange) (map[string]*TenantMetrics, error) {
	logrus.Infof("Collecting real metrics from %d discovered Mimir endpoints", len(endpoints))

	allMetrics := make(map[string]*TenantMetrics)

	// For each endpoint, try to collect metrics
	for _, endpoint := range endpoints {
		logrus.Infof("Querying metrics from endpoint: %s", endpoint)

		// Temporarily update the base URL for this endpoint
		originalBaseURL := c.baseURL
		c.baseURL = endpoint

		// Try to get basic metrics to test connectivity
		query := MetricQuery{
			Query: "up",
			Start: timeRange.Start,
			End:   timeRange.End,
			Step:  timeRange.Step,
		}

		_, err := c.QueryMetrics(ctx, query)
		if err != nil {
			logrus.Warnf("Failed to query endpoint %s: %v", endpoint, err)
			c.baseURL = originalBaseURL
			continue
		}

		logrus.Infof("Successfully connected to endpoint: %s", endpoint)

		// Try to get tenant-specific metrics
		// For now, we'll use a default tenant name since we're in single-tenant mode
		tenantName := "default"

		tenantMetrics := &TenantMetrics{
			TenantName: tenantName,
			TimeRange:  timeRange,
			Metrics:    make(map[string][]MetricSeries),
		}

		// Collect various metrics
		metricsToCollect := map[string]string{
			"ingestion_rate":   "cortex_distributor_ingestion_rate",
			"rejected_samples": "cortex_distributor_rejected_samples_total",
			"active_series":    "cortex_ingester_active_series",
			"memory_usage":     "cortex_ingester_memory_usage_bytes",
			"limits_reached":   "cortex_distributor_tenant_limits_reached_total",
			"process_memory":   "process_resident_memory_bytes",
			"build_info":       "cortex_build_info",
		}

		for metricName, queryString := range metricsToCollect {
			query := MetricQuery{
				Query: queryString,
				Start: timeRange.Start,
				End:   timeRange.End,
				Step:  timeRange.Step,
			}

			resp, err := c.QueryMetrics(ctx, query)
			if err != nil {
				logrus.Debugf("Failed to get %s from %s: %v", metricName, endpoint, err)
				continue
			}

			series := c.parseMetricResponse(resp, metricName)
			if len(series) > 0 {
				tenantMetrics.Metrics[metricName] = series
				logrus.Infof("Collected %d series for %s from %s", len(series), metricName, endpoint)
			}
		}

		allMetrics[endpoint] = tenantMetrics
		c.baseURL = originalBaseURL
	}

	logrus.Infof("Successfully collected metrics from %d endpoints", len(allMetrics))
	return allMetrics, nil
}

// GetProductionMetricsData returns production metrics data with real Mimir data
func (c *Client) GetProductionMetricsData(ctx context.Context, timeRange TimeRange) (map[string]interface{}, error) {
	logrus.Infof("Getting production metrics data")

	// Get discovered endpoints from discovery engine
	// For now, we'll use the configured base URL and try to discover endpoints
	endpoints := []string{c.baseURL}

	// Try to collect real metrics
	realMetrics, err := c.GetRealMetricsFromEndpoints(ctx, endpoints, timeRange)
	if err != nil {
		logrus.Warnf("Failed to collect real metrics: %v", err)
		return nil, err
	}

	// Convert to a format suitable for the UI
	productionData := map[string]interface{}{
		"data_source":  "production",
		"endpoints":    endpoints,
		"metrics":      realMetrics,
		"time_range":   timeRange,
		"collected_at": time.Now().UTC(),
	}

	return productionData, nil
}
