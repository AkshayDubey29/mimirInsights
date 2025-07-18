package limits

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/akshaydubey29/mimirInsights/pkg/k8s"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AutoDiscovery handles automatic discovery of limits from Mimir configurations
type AutoDiscovery struct {
	k8sClient *k8s.Client
}

// DiscoveredLimits represents auto-discovered limit configurations
type DiscoveredLimits struct {
	GlobalLimits  map[string]interface{} `json:"global_limits"`
	TenantLimits  map[string]TenantLimit `json:"tenant_limits"`
	ConfigSources []ConfigSource         `json:"config_sources"`
	LastUpdated   time.Time              `json:"last_updated"`
}

// TenantLimit represents limits for a specific tenant
type TenantLimit struct {
	TenantID    string                 `json:"tenant_id"`
	Limits      map[string]interface{} `json:"limits"`
	Source      string                 `json:"source"`
	LastUpdated time.Time              `json:"last_updated"`
}

// ConfigSource represents a source of configuration
type ConfigSource struct {
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Type      string    `json:"type"` // "configmap", "secret", "runtime-override"
	Keys      []string  `json:"keys"`
	LastSeen  time.Time `json:"last_seen"`
}

// NewAutoDiscovery creates a new auto-discovery instance
func NewAutoDiscovery(k8sClient *k8s.Client) *AutoDiscovery {
	return &AutoDiscovery{
		k8sClient: k8sClient,
	}
}

// DiscoverAllLimits discovers limits from all available sources
func (ad *AutoDiscovery) DiscoverAllLimits(ctx context.Context, mimirNamespace string) (*DiscoveredLimits, error) {
	logrus.Info("Starting AI-enabled auto-discovery of Mimir limits")

	discovered := &DiscoveredLimits{
		GlobalLimits:  make(map[string]interface{}),
		TenantLimits:  make(map[string]TenantLimit),
		ConfigSources: []ConfigSource{},
		LastUpdated:   time.Now(),
	}

	// Discover from runtime overrides
	if err := ad.discoverRuntimeOverrides(ctx, mimirNamespace, discovered); err != nil {
		logrus.Warnf("Failed to discover runtime overrides: %v", err)
	}

	// Discover from main Mimir config
	if err := ad.discoverMimirConfig(ctx, mimirNamespace, discovered); err != nil {
		logrus.Warnf("Failed to discover main Mimir config: %v", err)
	}

	// Discover from tenant-specific configs
	if err := ad.discoverTenantConfigs(ctx, mimirNamespace, discovered); err != nil {
		logrus.Warnf("Failed to discover tenant configs: %v", err)
	}

	// Discover from all namespaces for tenant overrides
	if err := ad.discoverNamespaceConfigs(ctx, discovered); err != nil {
		logrus.Warnf("Failed to discover namespace configs: %v", err)
	}

	logrus.Infof("Auto-discovery completed: %d global limits, %d tenant limits from %d sources",
		len(discovered.GlobalLimits), len(discovered.TenantLimits), len(discovered.ConfigSources))

	return discovered, nil
}

// discoverRuntimeOverrides discovers limits from runtime override ConfigMaps
func (ad *AutoDiscovery) discoverRuntimeOverrides(ctx context.Context, namespace string, discovered *DiscoveredLimits) error {
	// Try multiple common names for runtime overrides
	overrideNames := []string{
		"runtime-overrides",
		"mimir-runtime-overrides",
		"cortex-runtime-overrides",
		"overrides",
		"mimir-overrides",
	}

	for _, name := range overrideNames {
		configMap, err := ad.k8sClient.GetConfigMap(ctx, namespace, name, metav1.GetOptions{})
		if err != nil {
			continue // Try next name
		}

		logrus.Infof("Found runtime overrides ConfigMap: %s", name)

		// Add to config sources
		discovered.ConfigSources = append(discovered.ConfigSources, ConfigSource{
			Name:      name,
			Namespace: namespace,
			Type:      "runtime-override",
			Keys:      getConfigMapKeys(configMap.Data),
			LastSeen:  time.Now(),
		})

		// Parse the overrides
		for key, value := range configMap.Data {
			if strings.HasSuffix(key, ".yaml") || strings.HasSuffix(key, ".yml") {
				ad.parseOverridesYAML(value, discovered)
			}
		}

		break // Stop after finding the first one
	}

	return nil
}

// discoverMimirConfig discovers limits from main Mimir configuration
func (ad *AutoDiscovery) discoverMimirConfig(ctx context.Context, namespace string, discovered *DiscoveredLimits) error {
	configNames := []string{
		"mimir-config",
		"cortex-config",
		"mimir",
		"cortex",
	}

	for _, name := range configNames {
		configMap, err := ad.k8sClient.GetConfigMap(ctx, namespace, name, metav1.GetOptions{})
		if err != nil {
			continue
		}

		logrus.Infof("Found Mimir config ConfigMap: %s", name)

		discovered.ConfigSources = append(discovered.ConfigSources, ConfigSource{
			Name:      name,
			Namespace: namespace,
			Type:      "configmap",
			Keys:      getConfigMapKeys(configMap.Data),
			LastSeen:  time.Now(),
		})

		// Parse the main config
		for key, value := range configMap.Data {
			if strings.HasSuffix(key, ".yaml") || strings.HasSuffix(key, ".yml") {
				ad.parseMimirConfigYAML(value, discovered)
			}
		}

		break
	}

	return nil
}

// discoverTenantConfigs discovers tenant-specific configurations
func (ad *AutoDiscovery) discoverTenantConfigs(ctx context.Context, namespace string, discovered *DiscoveredLimits) error {
	// Get all ConfigMaps in the Mimir namespace
	configMaps, err := ad.k8sClient.GetConfigMaps(ctx, namespace, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get ConfigMaps: %w", err)
	}

	for _, cm := range configMaps.Items {
		// Look for tenant-specific patterns
		if ad.isTenantConfigMap(cm.Name) {
			logrus.Infof("Found tenant config ConfigMap: %s", cm.Name)

			discovered.ConfigSources = append(discovered.ConfigSources, ConfigSource{
				Name:      cm.Name,
				Namespace: cm.Namespace,
				Type:      "configmap",
				Keys:      getConfigMapKeys(cm.Data),
				LastSeen:  time.Now(),
			})

			// Extract tenant ID from ConfigMap name
			tenantID := ad.extractTenantFromConfigMapName(cm.Name)
			ad.parseTenantConfigMap(cm.Data, tenantID, discovered)
		}
	}

	return nil
}

// discoverNamespaceConfigs discovers configurations across all namespaces
func (ad *AutoDiscovery) discoverNamespaceConfigs(ctx context.Context, discovered *DiscoveredLimits) error {
	// Get all namespaces
	namespaces, err := ad.k8sClient.GetNamespaces(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get namespaces: %w", err)
	}

	for _, namespace := range namespaces.Items {
		// Skip system namespaces
		if ad.isSystemNamespace(namespace.Name) {
			continue
		}

		// Look for monitoring configurations in tenant namespaces
		if err := ad.scanNamespaceForConfigs(ctx, namespace.Name, discovered); err != nil {
			logrus.Warnf("Failed to scan namespace %s: %v", namespace.Name, err)
		}
	}

	return nil
}

// scanNamespaceForConfigs scans a namespace for relevant configurations
func (ad *AutoDiscovery) scanNamespaceForConfigs(ctx context.Context, namespace string, discovered *DiscoveredLimits) error {
	configMaps, err := ad.k8sClient.GetConfigMaps(ctx, namespace, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, cm := range configMaps.Items {
		// Look for Alloy/Grafana Agent configs that might contain tenant info
		if ad.isMonitoringConfigMap(cm.Name) {
			discovered.ConfigSources = append(discovered.ConfigSources, ConfigSource{
				Name:      cm.Name,
				Namespace: cm.Namespace,
				Type:      "configmap",
				Keys:      getConfigMapKeys(cm.Data),
				LastSeen:  time.Now(),
			})

			// Parse for tenant information and potential limit overrides
			ad.parseMonitoringConfig(cm.Data, namespace, discovered)
		}
	}

	return nil
}

// parseOverridesYAML parses runtime overrides YAML content
func (ad *AutoDiscovery) parseOverridesYAML(content string, discovered *DiscoveredLimits) {
	var overrides map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &overrides); err != nil {
		logrus.Warnf("Failed to parse overrides YAML: %v", err)
		return
	}

	// Extract overrides
	if overridesSection, exists := overrides["overrides"]; exists {
		if overridesMap, ok := overridesSection.(map[interface{}]interface{}); ok {
			for tenantKey, tenantLimits := range overridesMap {
				tenantID := fmt.Sprintf("%v", tenantKey)

				if limitsMap, ok := tenantLimits.(map[interface{}]interface{}); ok {
					tenantLimit := TenantLimit{
						TenantID:    tenantID,
						Limits:      convertInterfaceMap(limitsMap),
						Source:      "runtime-override",
						LastUpdated: time.Now(),
					}
					discovered.TenantLimits[tenantID] = tenantLimit
				}
			}
		}
	}

	// Extract global defaults
	if limits, exists := overrides["limits"]; exists {
		if limitsMap, ok := limits.(map[interface{}]interface{}); ok {
			for key, value := range limitsMap {
				discovered.GlobalLimits[fmt.Sprintf("%v", key)] = value
			}
		}
	}
}

// parseMimirConfigYAML parses main Mimir configuration
func (ad *AutoDiscovery) parseMimirConfigYAML(content string, discovered *DiscoveredLimits) {
	var config map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		logrus.Warnf("Failed to parse Mimir config YAML: %v", err)
		return
	}

	// Look for limits section
	if limits, exists := config["limits"]; exists {
		if limitsMap, ok := limits.(map[interface{}]interface{}); ok {
			for key, value := range limitsMap {
				discovered.GlobalLimits[fmt.Sprintf("%v", key)] = value
			}
		}
	}

	// Look for distributor limits
	if distributor, exists := config["distributor"]; exists {
		if distributorMap, ok := distributor.(map[interface{}]interface{}); ok {
			ad.extractLimitsFromSection(distributorMap, "distributor", discovered.GlobalLimits)
		}
	}

	// Look for ingester limits
	if ingester, exists := config["ingester"]; exists {
		if ingesterMap, ok := ingester.(map[interface{}]interface{}); ok {
			ad.extractLimitsFromSection(ingesterMap, "ingester", discovered.GlobalLimits)
		}
	}

	// Look for querier limits
	if querier, exists := config["querier"]; exists {
		if querierMap, ok := querier.(map[interface{}]interface{}); ok {
			ad.extractLimitsFromSection(querierMap, "querier", discovered.GlobalLimits)
		}
	}
}

// parseTenantConfigMap parses tenant-specific configuration
func (ad *AutoDiscovery) parseTenantConfigMap(data map[string]string, tenantID string, discovered *DiscoveredLimits) {
	limits := make(map[string]interface{})

	for key, value := range data {
		// Parse different formats
		if strings.HasSuffix(key, ".yaml") || strings.HasSuffix(key, ".yml") {
			var yamlData map[string]interface{}
			if err := yaml.Unmarshal([]byte(value), &yamlData); err == nil {
				for k, v := range yamlData {
					limits[k] = v
				}
			}
		} else {
			// Try to parse as key=value
			if ad.isLimitKey(key) {
				if numValue, err := ad.parseNumericValue(value); err == nil {
					limits[key] = numValue
				} else {
					limits[key] = value
				}
			}
		}
	}

	if len(limits) > 0 {
		tenantLimit := TenantLimit{
			TenantID:    tenantID,
			Limits:      limits,
			Source:      "configmap",
			LastUpdated: time.Now(),
		}
		discovered.TenantLimits[tenantID] = tenantLimit
	}
}

// parseMonitoringConfig parses monitoring configuration for tenant info
func (ad *AutoDiscovery) parseMonitoringConfig(data map[string]string, namespace string, discovered *DiscoveredLimits) {
	for _, content := range data {
		// Look for X-Scope-OrgID headers that indicate tenant mappings
		orgID := ad.extractOrgIDFromContent(content)
		if orgID != "" {
			// Create a tenant entry if we found an org ID
			if existing, exists := discovered.TenantLimits[orgID]; !exists {
				tenantLimit := TenantLimit{
					TenantID:    orgID,
					Limits:      make(map[string]interface{}),
					Source:      fmt.Sprintf("namespace:%s", namespace),
					LastUpdated: time.Now(),
				}
				discovered.TenantLimits[orgID] = tenantLimit
			} else {
				// Update source if we found it in a different namespace
				existing.Source = fmt.Sprintf("%s,namespace:%s", existing.Source, namespace)
				discovered.TenantLimits[orgID] = existing
			}
		}
	}
}

// Helper functions
func (ad *AutoDiscovery) isTenantConfigMap(name string) bool {
	tenantPatterns := []string{
		"tenant-",
		"-tenant",
		"-limits",
		"-overrides",
		"user-",
		"org-",
	}

	name = strings.ToLower(name)
	for _, pattern := range tenantPatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}

	return false
}

func (ad *AutoDiscovery) isMonitoringConfigMap(name string) bool {
	monitoringPatterns := []string{
		"alloy",
		"grafana-agent",
		"prometheus",
		"agent-config",
		"monitoring",
		"scrape",
	}

	name = strings.ToLower(name)
	for _, pattern := range monitoringPatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}

	return false
}

func (ad *AutoDiscovery) isSystemNamespace(name string) bool {
	systemNamespaces := []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
		"default",
		"mimir-insights",
	}

	for _, sysNS := range systemNamespaces {
		if name == sysNS {
			return true
		}
	}

	return false
}

func (ad *AutoDiscovery) extractTenantFromConfigMapName(name string) string {
	// Try to extract tenant ID from ConfigMap name
	patterns := []string{
		`tenant-(.+)`,
		`(.+)-tenant`,
		`(.+)-limits`,
		`(.+)-overrides`,
		`user-(.+)`,
		`org-(.+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(name); len(matches) > 1 {
			return matches[1]
		}
	}

	return name
}

func (ad *AutoDiscovery) extractOrgIDFromContent(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(strings.ToLower(line), "x-scope-orgid") {
			// Extract the value
			if strings.Contains(line, ":") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					orgID := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
					return orgID
				}
			}
		}
	}
	return ""
}

func (ad *AutoDiscovery) isLimitKey(key string) bool {
	limitKeywords := []string{
		"limit",
		"max",
		"rate",
		"burst",
		"series",
		"ingestion",
		"query",
		"retention",
	}

	keyLower := strings.ToLower(key)
	for _, keyword := range limitKeywords {
		if strings.Contains(keyLower, keyword) {
			return true
		}
	}

	return false
}

func (ad *AutoDiscovery) parseNumericValue(value string) (interface{}, error) {
	// Try to parse as integer first
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal, nil
	}

	// Try to parse as float
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal, nil
	}

	// Try to parse duration strings (e.g., "5m", "1h")
	if strings.HasSuffix(value, "s") || strings.HasSuffix(value, "m") ||
		strings.HasSuffix(value, "h") || strings.HasSuffix(value, "d") {
		return value, nil
	}

	return nil, fmt.Errorf("not a numeric value")
}

func (ad *AutoDiscovery) extractLimitsFromSection(section map[interface{}]interface{}, prefix string, globalLimits map[string]interface{}) {
	for key, value := range section {
		keyStr := fmt.Sprintf("%v", key)
		if ad.isLimitKey(keyStr) {
			fullKey := fmt.Sprintf("%s.%s", prefix, keyStr)
			globalLimits[fullKey] = value
		}
	}
}

func getConfigMapKeys(data map[string]string) []string {
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	return keys
}

func convertInterfaceMap(input map[interface{}]interface{}) map[string]interface{} {
	output := make(map[string]interface{})
	for key, value := range input {
		strKey := fmt.Sprintf("%v", key)
		output[strKey] = value
	}
	return output
}
