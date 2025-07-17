package drift

import (
	"context"
	"fmt"
	"time"

	"github.com/akshaydubey29/mimirInsights/pkg/discovery"
	"github.com/akshaydubey29/mimirInsights/pkg/k8s"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Detector handles configuration drift detection
type Detector struct {
	k8sClient       *k8s.Client
	discoveryEngine *discovery.Engine
}

// DriftResult represents detected configuration drift
type DriftResult struct {
	TenantName    string                 `json:"tenant_name"`
	ConfigType    string                 `json:"config_type"`
	Expected      map[string]interface{} `json:"expected"`
	Actual        map[string]interface{} `json:"actual"`
	DriftDetected bool                   `json:"drift_detected"`
	LastChecked   time.Time              `json:"last_checked"`
	Severity      string                 `json:"severity"`
	Description   string                 `json:"description"`
}

// DriftSummary represents overall drift status
type DriftSummary struct {
	TotalTenants    int           `json:"total_tenants"`
	TenantsWithDrift int          `json:"tenants_with_drift"`
	DriftResults    []DriftResult `json:"drift_results"`
	LastScan        time.Time     `json:"last_scan"`
}

// NewDetector creates a new drift detector
func NewDetector(k8sClient *k8s.Client, discoveryEngine *discovery.Engine) *Detector {
	return &Detector{
		k8sClient:       k8sClient,
		discoveryEngine: discoveryEngine,
	}
}

// DetectDrift performs drift detection across all tenants
func (d *Detector) DetectDrift(ctx context.Context) error {
	logrus.Info("Starting configuration drift detection")

	// Discover all tenants
	result, err := d.discoveryEngine.DiscoverAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover tenants: %w", err)
	}

	for _, tenant := range result.TenantNamespaces {
		if err := d.detectTenantDrift(ctx, tenant.Name); err != nil {
			logrus.Warnf("Failed to detect drift for tenant %s: %v", tenant.Name, err)
		}
	}

	return nil
}

// detectTenantDrift detects drift for a specific tenant
func (d *Detector) detectTenantDrift(ctx context.Context, tenantName string) error {
	// Check Alloy configuration drift
	if err := d.detectAlloyDrift(ctx, tenantName); err != nil {
		return fmt.Errorf("failed to detect Alloy drift: %w", err)
	}

	// Check Mimir limits drift
	if err := d.detectLimitsDrift(ctx, tenantName); err != nil {
		return fmt.Errorf("failed to detect limits drift: %w", err)
	}

	return nil
}

// detectAlloyDrift detects drift in Alloy configuration
func (d *Detector) detectAlloyDrift(ctx context.Context, tenantName string) error {
	// Get current Alloy ConfigMap
	configMap, err := d.k8sClient.GetConfigMap(ctx, tenantName, "alloy-config", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get Alloy ConfigMap: %w", err)
	}

	// Parse current configuration
	var currentConfig map[string]interface{}
	if err := yaml.Unmarshal([]byte(configMap.Data["config.yaml"]), &currentConfig); err != nil {
		return fmt.Errorf("failed to parse current config: %w", err)
	}

	// Generate expected configuration
	expectedConfig := d.generateExpectedAlloyConfig(tenantName)

	// Compare configurations
	driftDetected := d.compareConfigurations(currentConfig, expectedConfig)

	if driftDetected {
		logrus.Warnf("Configuration drift detected for tenant %s Alloy config", tenantName)
		// Store drift result for later retrieval
		d.storeDriftResult(DriftResult{
			TenantName:    tenantName,
			ConfigType:    "alloy",
			Expected:      expectedConfig,
			Actual:        currentConfig,
			DriftDetected: true,
			LastChecked:   time.Now(),
			Severity:      "medium",
			Description:   "Alloy configuration has drifted from expected state",
		})
	}

	return nil
}

// detectLimitsDrift detects drift in Mimir limits configuration
func (d *Detector) detectLimitsDrift(ctx context.Context, tenantName string) error {
	// Get runtime-overrides ConfigMap from Mimir namespace
	configMap, err := d.k8sClient.GetConfigMap(ctx, "mimir", "runtime-overrides", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get runtime-overrides ConfigMap: %w", err)
	}

	// Parse current limits
	var currentLimits map[string]interface{}
	if err := yaml.Unmarshal([]byte(configMap.Data["overrides.yaml"]), &currentLimits); err != nil {
		return fmt.Errorf("failed to parse current limits: %w", err)
	}

	// Get tenant-specific limits
	tenantLimits, exists := currentLimits[tenantName]
	if !exists {
		logrus.Warnf("No limits found for tenant %s", tenantName)
		return nil
	}

	// Generate expected limits based on recommendations
	expectedLimits := d.generateExpectedLimits(tenantName)

	// Compare limits
	driftDetected := d.compareConfigurations(tenantLimits.(map[string]interface{}), expectedLimits)

	if driftDetected {
		logrus.Warnf("Limits drift detected for tenant %s", tenantName)
		d.storeDriftResult(DriftResult{
			TenantName:    tenantName,
			ConfigType:    "limits",
			Expected:      expectedLimits,
			Actual:        tenantLimits.(map[string]interface{}),
			DriftDetected: true,
			LastChecked:   time.Now(),
			Severity:      "high",
			Description:   "Mimir limits have drifted from recommended values",
		})
	}

	return nil
}

// compareConfigurations compares two configuration maps
func (d *Detector) compareConfigurations(current, expected map[string]interface{}) bool {
	// Simple comparison - in production, this would be more sophisticated
	for key, expectedValue := range expected {
		if currentValue, exists := current[key]; !exists || currentValue != expectedValue {
			return true
		}
	}
	return false
}

// generateExpectedAlloyConfig generates expected Alloy configuration
func (d *Detector) generateExpectedAlloyConfig(tenantName string) map[string]interface{} {
	// This would generate the expected configuration based on best practices
	return map[string]interface{}{
		"scrape_interval": "15s",
		"evaluation_interval": "15s",
		"external_labels": map[string]string{
			"tenant": tenantName,
		},
	}
}

// generateExpectedLimits generates expected limits based on recommendations
func (d *Detector) generateExpectedLimits(tenantName string) map[string]interface{} {
	// This would use the limits analyzer to generate recommended limits
	return map[string]interface{}{
		"ingestion_rate": 10000,
		"max_global_series_per_user": 5000000,
		"max_label_names_per_series": 30,
	}
}

// storeDriftResult stores drift result for later retrieval
func (d *Detector) storeDriftResult(result DriftResult) {
	// In production, this would store to a database or cache
	logrus.Infof("Storing drift result for tenant %s: %+v", result.TenantName, result)
}

// GetDriftSummary returns overall drift status
func (d *Detector) GetDriftSummary(ctx context.Context) (*DriftSummary, error) {
	// This would retrieve stored drift results
	return &DriftSummary{
		TotalTenants:     3,
		TenantsWithDrift: 1,
		DriftResults: []DriftResult{
			{
				TenantName:    "team-a",
				ConfigType:    "limits",
				DriftDetected: true,
				LastChecked:   time.Now(),
				Severity:      "medium",
				Description:   "Ingestion rate limit manually modified",
			},
		},
		LastScan: time.Now(),
	}, nil
}

// FixDrift attempts to fix configuration drift for a tenant
func (d *Detector) FixDrift(ctx context.Context, tenantName, configType string) error {
	logrus.Infof("Attempting to fix drift for tenant %s, config type %s", tenantName, configType)

	switch configType {
	case "alloy":
		return d.fixAlloyDrift(ctx, tenantName)
	case "limits":
		return d.fixLimitsDrift(ctx, tenantName)
	default:
		return fmt.Errorf("unsupported config type: %s", configType)
	}
}

// fixAlloyDrift fixes Alloy configuration drift
func (d *Detector) fixAlloyDrift(ctx context.Context, tenantName string) error {
	expectedConfig := d.generateExpectedAlloyConfig(tenantName)
	
	// Convert to YAML
	configYAML, err := yaml.Marshal(expectedConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Update ConfigMap
	configMap, err := d.k8sClient.GetConfigMap(ctx, tenantName, "alloy-config", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	configMap.Data["config.yaml"] = string(configYAML)

	_, err = d.k8sClient.UpdateConfigMap(ctx, tenantName, configMap, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ConfigMap: %w", err)
	}

	logrus.Infof("Fixed Alloy drift for tenant %s", tenantName)
	return nil
}

// fixLimitsDrift fixes Mimir limits drift
func (d *Detector) fixLimitsDrift(ctx context.Context, tenantName string) error {
	// Get runtime-overrides ConfigMap
	configMap, err := d.k8sClient.GetConfigMap(ctx, "mimir", "runtime-overrides", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get runtime-overrides ConfigMap: %w", err)
	}

	// Parse current overrides
	var overrides map[string]interface{}
	if err := yaml.Unmarshal([]byte(configMap.Data["overrides.yaml"]), &overrides); err != nil {
		return fmt.Errorf("failed to parse overrides: %w", err)
	}

	// Update tenant limits with expected values
	expectedLimits := d.generateExpectedLimits(tenantName)
	overrides[tenantName] = expectedLimits

	// Convert back to YAML
	overridesYAML, err := yaml.Marshal(overrides)
	if err != nil {
		return fmt.Errorf("failed to marshal overrides: %w", err)
	}

	configMap.Data["overrides.yaml"] = string(overridesYAML)

	_, err = d.k8sClient.UpdateConfigMap(ctx, "mimir", configMap, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ConfigMap: %w", err)
	}

	logrus.Infof("Fixed limits drift for tenant %s", tenantName)
	return nil
}