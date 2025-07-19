package discovery

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/akshaydubey29/mimirInsights/pkg/config"
	"github.com/akshaydubey29/mimirInsights/pkg/k8s"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TenantDiscoveryStrategy represents different approaches to discover tenants
type TenantDiscoveryStrategy string

const (
	StrategyNamespaceLabels    TenantDiscoveryStrategy = "namespace_labels"
	StrategyMimirMetrics       TenantDiscoveryStrategy = "mimir_metrics"
	StrategyMimirConfig        TenantDiscoveryStrategy = "mimir_config"
	StrategyKubernetesLabels   TenantDiscoveryStrategy = "kubernetes_labels"
	StrategyServiceDiscovery   TenantDiscoveryStrategy = "service_discovery"
	StrategyConfigMapPatterns  TenantDiscoveryStrategy = "configmap_patterns"
	StrategyIngressAnnotations TenantDiscoveryStrategy = "ingress_annotations"
	StrategyPodLabels          TenantDiscoveryStrategy = "pod_labels"
	StrategySecretPatterns     TenantDiscoveryStrategy = "secret_patterns"
	StrategyNetworkPolicies    TenantDiscoveryStrategy = "network_policies"
	StrategyRBACBindings       TenantDiscoveryStrategy = "rbac_bindings"
)

// TenantDiscoveryResult represents the result of a single discovery strategy
type TenantDiscoveryResult struct {
	Strategy    TenantDiscoveryStrategy `json:"strategy"`
	Tenants     []TenantInfo            `json:"tenants"`
	Confidence  float64                 `json:"confidence"`
	Errors      []string                `json:"errors"`
	Duration    time.Duration           `json:"duration"`
	LastUpdated time.Time               `json:"last_updated"`
}

// TenantInfo represents a discovered tenant with comprehensive information
type TenantInfo struct {
	Name             string                  `json:"name"`
	Namespace        string                  `json:"namespace"`
	OrgID            string                  `json:"org_id"`
	Source           TenantDiscoveryStrategy `json:"source"`
	Confidence       float64                 `json:"confidence"`
	Labels           map[string]string       `json:"labels"`
	Annotations      map[string]string       `json:"annotations"`
	Resources        TenantResources         `json:"resources"`
	Metrics          TenantMetrics           `json:"metrics"`
	Configuration    TenantConfiguration     `json:"configuration"`
	NetworkPolicies  []string                `json:"network_policies"`
	RBACBindings     []string                `json:"rbac_bindings"`
	LastSeen         time.Time               `json:"last_seen"`
	DiscoveryMethods []string                `json:"discovery_methods"`
}

// TenantResources represents tenant-specific resources
type TenantResources struct {
	Pods            []string `json:"pods"`
	Services        []string `json:"services"`
	ConfigMaps      []string `json:"config_maps"`
	Secrets         []string `json:"secrets"`
	Deployments     []string `json:"deployments"`
	StatefulSets    []string `json:"stateful_sets"`
	DaemonSets      []string `json:"daemon_sets"`
	Ingresses       []string `json:"ingresses"`
	NetworkPolicies []string `json:"network_policies"`
}

// TenantMetrics represents tenant-specific metrics
type TenantMetrics struct {
	IngestionRate     float64   `json:"ingestion_rate"`
	QueryRate         float64   `json:"query_rate"`
	SeriesCount       int64     `json:"series_count"`
	SamplesPerSecond  float64   `json:"samples_per_second"`
	StorageUsageGB    float64   `json:"storage_usage_gb"`
	CPUUsage          float64   `json:"cpu_usage"`
	MemoryUsage       float64   `json:"memory_usage"`
	ErrorRate         float64   `json:"error_rate"`
	LastMetricsUpdate time.Time `json:"last_metrics_update"`
}

// TenantConfiguration represents tenant-specific configuration
type TenantConfiguration struct {
	MimirLimits    map[string]interface{} `json:"mimir_limits"`
	AlloyConfig    *AlloyConfig           `json:"alloy_config"`
	ConsulConfig   *ConsulConfig          `json:"consul_config"`
	NginxConfig    *NginxConfig           `json:"nginx_config"`
	ScrapeConfigs  []ScrapeConfig         `json:"scrape_configs"`
	AlertingRules  []string               `json:"alerting_rules"`
	RecordingRules []string               `json:"recording_rules"`
}

// MultiStrategyTenantDiscovery handles comprehensive tenant discovery using multiple strategies
type MultiStrategyTenantDiscovery struct {
	k8sClient *k8s.Client
	config    *config.Config
	engine    *Engine
}

// NewMultiStrategyTenantDiscovery creates a new multi-strategy tenant discovery instance
func NewMultiStrategyTenantDiscovery(engine *Engine) *MultiStrategyTenantDiscovery {
	return &MultiStrategyTenantDiscovery{
		k8sClient: engine.GetK8sClient(),
		config:    engine.GetConfig(),
		engine:    engine,
	}
}

// DiscoverTenantsComprehensive performs comprehensive tenant discovery using all available strategies
func (m *MultiStrategyTenantDiscovery) DiscoverTenantsComprehensive(ctx context.Context) (*ComprehensiveTenantDiscoveryResult, error) {
	logrus.Info("🔍 Starting comprehensive tenant discovery using multiple strategies")

	start := time.Now()
	results := make(map[TenantDiscoveryStrategy]*TenantDiscoveryResult)
	errors := []string{}

	// Strategy 1: Namespace Labels Discovery
	if result, err := m.discoverByNamespaceLabels(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("Namespace labels discovery failed: %v", err))
	} else {
		results[StrategyNamespaceLabels] = result
	}

	// Strategy 2: Mimir Metrics Discovery
	if result, err := m.discoverByMimirMetrics(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("Mimir metrics discovery failed: %v", err))
	} else {
		results[StrategyMimirMetrics] = result
	}

	// Strategy 3: Mimir Configuration Discovery
	if result, err := m.discoverByMimirConfig(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("Mimir config discovery failed: %v", err))
	} else {
		results[StrategyMimirConfig] = result
	}

	// Strategy 4: Kubernetes Labels Discovery
	if result, err := m.discoverByKubernetesLabels(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("Kubernetes labels discovery failed: %v", err))
	} else {
		results[StrategyKubernetesLabels] = result
	}

	// Strategy 5: Service Discovery
	if result, err := m.discoverByServiceDiscovery(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("Service discovery failed: %v", err))
	} else {
		results[StrategyServiceDiscovery] = result
	}

	// Strategy 6: ConfigMap Patterns Discovery
	if result, err := m.discoverByConfigMapPatterns(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("ConfigMap patterns discovery failed: %v", err))
	} else {
		results[StrategyConfigMapPatterns] = result
	}

	// Strategy 7: Ingress Annotations Discovery
	if result, err := m.discoverByIngressAnnotations(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("Ingress annotations discovery failed: %v", err))
	} else {
		results[StrategyIngressAnnotations] = result
	}

	// Strategy 8: Pod Labels Discovery
	if result, err := m.discoverByPodLabels(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("Pod labels discovery failed: %v", err))
	} else {
		results[StrategyPodLabels] = result
	}

	// Strategy 9: Secret Patterns Discovery
	if result, err := m.discoverBySecretPatterns(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("Secret patterns discovery failed: %v", err))
	} else {
		results[StrategySecretPatterns] = result
	}

	// Strategy 10: Network Policies Discovery
	if result, err := m.discoverByNetworkPolicies(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("Network policies discovery failed: %v", err))
	} else {
		results[StrategyNetworkPolicies] = result
	}

	// Strategy 11: RBAC Bindings Discovery
	if result, err := m.discoverByRBACBindings(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("RBAC bindings discovery failed: %v", err))
	} else {
		results[StrategyRBACBindings] = result
	}

	// Consolidate and deduplicate results
	consolidatedTenants := m.consolidateTenantResults(results)

	// Perform cross-validation and confidence scoring
	validatedTenants := m.crossValidateTenants(ctx, consolidatedTenants)

	comprehensiveResult := &ComprehensiveTenantDiscoveryResult{
		Strategies:           results,
		ConsolidatedTenants:  validatedTenants,
		TotalStrategies:      len(results),
		SuccessfulStrategies: len(results),
		Errors:               errors,
		Duration:             time.Since(start),
		LastUpdated:          time.Now(),
	}

	logrus.Infof("✅ Comprehensive tenant discovery completed in %v", comprehensiveResult.Duration)
	logrus.Infof("📊 Discovered %d tenants using %d strategies", len(validatedTenants), len(results))

	return comprehensiveResult, nil
}

// ComprehensiveTenantDiscoveryResult represents the complete result of multi-strategy discovery
type ComprehensiveTenantDiscoveryResult struct {
	Strategies           map[TenantDiscoveryStrategy]*TenantDiscoveryResult `json:"strategies"`
	ConsolidatedTenants  []TenantInfo                                       `json:"consolidated_tenants"`
	TotalStrategies      int                                                `json:"total_strategies"`
	SuccessfulStrategies int                                                `json:"successful_strategies"`
	Errors               []string                                           `json:"errors"`
	Duration             time.Duration                                      `json:"duration"`
	LastUpdated          time.Time                                          `json:"last_updated"`
}

// discoverByNamespaceLabels discovers tenants by analyzing namespace labels
func (m *MultiStrategyTenantDiscovery) discoverByNamespaceLabels(ctx context.Context) (*TenantDiscoveryResult, error) {
	start := time.Now()
	logrus.Info("🔍 Strategy 1: Discovering tenants by namespace labels")

	namespaces, err := m.k8sClient.GetNamespaces(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get namespaces: %v", err)
	}

	tenants := []TenantInfo{}
	errors := []string{}

	// Tenant label patterns to look for
	tenantLabelPatterns := []string{
		"tenant",
		"team",
		"org",
		"organization",
		"project",
		"environment",
		"namespace",
		"partition",
		"division",
		"department",
	}

	for _, ns := range namespaces.Items {
		tenantInfo := m.extractTenantFromNamespaceLabels(&ns, tenantLabelPatterns)
		if tenantInfo != nil {
			tenants = append(tenants, *tenantInfo)
		}
	}

	confidence := m.calculateStrategyConfidence(len(tenants), len(namespaces.Items), 0.8)

	return &TenantDiscoveryResult{
		Strategy:    StrategyNamespaceLabels,
		Tenants:     tenants,
		Confidence:  confidence,
		Errors:      errors,
		Duration:    time.Since(start),
		LastUpdated: time.Now(),
	}, nil
}

// discoverByMimirMetrics discovers tenants by analyzing Mimir metrics
func (m *MultiStrategyTenantDiscovery) discoverByMimirMetrics(ctx context.Context) (*TenantDiscoveryResult, error) {
	start := time.Now()
	logrus.Info("🔍 Strategy 2: Discovering tenants by Mimir metrics")

	// This would require querying Mimir's metrics API
	// For now, we'll implement a placeholder that looks for tenant-related metrics
	tenants := []TenantInfo{}
	errors := []string{}

	// Query Mimir for tenant-related metrics
	// Example metrics to look for:
	// - cortex_ingester_series_created_total{user="tenant1"}
	// - cortex_querier_request_duration_seconds{user="tenant2"}
	// - cortex_distributor_received_samples_total{user="tenant3"}

	// Placeholder implementation - in real implementation, this would query Mimir API
	logrus.Info("📊 Querying Mimir metrics for tenant discovery (placeholder)")

	confidence := m.calculateStrategyConfidence(len(tenants), 0, 0.9)

	return &TenantDiscoveryResult{
		Strategy:    StrategyMimirMetrics,
		Tenants:     tenants,
		Confidence:  confidence,
		Errors:      errors,
		Duration:    time.Since(start),
		LastUpdated: time.Now(),
	}, nil
}

// discoverByMimirConfig discovers tenants by analyzing Mimir configuration
func (m *MultiStrategyTenantDiscovery) discoverByMimirConfig(ctx context.Context) (*TenantDiscoveryResult, error) {
	start := time.Now()
	logrus.Info("🔍 Strategy 3: Discovering tenants by Mimir configuration")

	tenants := []TenantInfo{}
	errors := []string{}

	// Look for Mimir configuration in ConfigMaps
	configMaps, err := m.k8sClient.GetConfigMaps(ctx, "", metav1.ListOptions{})
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to get ConfigMaps: %v", err))
	} else {
		for _, cm := range configMaps.Items {
			tenantInfo := m.extractTenantFromMimirConfig(&cm)
			if tenantInfo != nil {
				tenants = append(tenants, *tenantInfo)
			}
		}
	}

	confidence := m.calculateStrategyConfidence(len(tenants), len(configMaps.Items), 0.85)

	return &TenantDiscoveryResult{
		Strategy:    StrategyMimirConfig,
		Tenants:     tenants,
		Confidence:  confidence,
		Errors:      errors,
		Duration:    time.Since(start),
		LastUpdated: time.Now(),
	}, nil
}

// discoverByKubernetesLabels discovers tenants by analyzing Kubernetes resource labels
func (m *MultiStrategyTenantDiscovery) discoverByKubernetesLabels(ctx context.Context) (*TenantDiscoveryResult, error) {
	start := time.Now()
	logrus.Info("🔍 Strategy 4: Discovering tenants by Kubernetes labels")

	tenants := []TenantInfo{}
	errors := []string{}

	// Analyze labels from various Kubernetes resources
	resources := []struct {
		name   string
		getter func(context.Context) ([]interface{}, error)
	}{
		{"Pods", m.getPodsForLabelAnalysis},
		{"Services", m.getServicesForLabelAnalysis},
		{"Deployments", m.getDeploymentsForLabelAnalysis},
		{"StatefulSets", m.getStatefulSetsForLabelAnalysis},
	}

	for _, resource := range resources {
		items, err := resource.getter(ctx)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to get %s: %v", resource.name, err))
			continue
		}

		for _, item := range items {
			tenantInfo := m.extractTenantFromResourceLabels(item)
			if tenantInfo != nil {
				tenants = append(tenants, *tenantInfo)
			}
		}
	}

	confidence := m.calculateStrategyConfidence(len(tenants), 0, 0.75)

	return &TenantDiscoveryResult{
		Strategy:    StrategyKubernetesLabels,
		Tenants:     tenants,
		Confidence:  confidence,
		Errors:      errors,
		Duration:    time.Since(start),
		LastUpdated: time.Now(),
	}, nil
}

// discoverByServiceDiscovery discovers tenants by analyzing service patterns
func (m *MultiStrategyTenantDiscovery) discoverByServiceDiscovery(ctx context.Context) (*TenantDiscoveryResult, error) {
	start := time.Now()
	logrus.Info("🔍 Strategy 5: Discovering tenants by service discovery")

	tenants := []TenantInfo{}
	errors := []string{}

	services, err := m.k8sClient.GetServices(ctx, "", metav1.ListOptions{})
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to get services: %v", err))
	} else {
		for _, svc := range services.Items {
			tenantInfo := m.extractTenantFromService(&svc)
			if tenantInfo != nil {
				tenants = append(tenants, *tenantInfo)
			}
		}
	}

	confidence := m.calculateStrategyConfidence(len(tenants), len(services.Items), 0.7)

	return &TenantDiscoveryResult{
		Strategy:    StrategyServiceDiscovery,
		Tenants:     tenants,
		Confidence:  confidence,
		Errors:      errors,
		Duration:    time.Since(start),
		LastUpdated: time.Now(),
	}, nil
}

// discoverByConfigMapPatterns discovers tenants by analyzing ConfigMap patterns
func (m *MultiStrategyTenantDiscovery) discoverByConfigMapPatterns(ctx context.Context) (*TenantDiscoveryResult, error) {
	start := time.Now()
	logrus.Info("🔍 Strategy 6: Discovering tenants by ConfigMap patterns")

	tenants := []TenantInfo{}
	errors := []string{}

	configMaps, err := m.k8sClient.GetConfigMaps(ctx, "")
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to get ConfigMaps: %v", err))
	} else {
		for _, cm := range configMaps {
			tenantInfo := m.extractTenantFromConfigMapPatterns(cm)
			if tenantInfo != nil {
				tenants = append(tenants, *tenantInfo)
			}
		}
	}

	confidence := m.calculateStrategyConfidence(len(tenants), len(configMaps), 0.8)

	return &TenantDiscoveryResult{
		Strategy:    StrategyConfigMapPatterns,
		Tenants:     tenants,
		Confidence:  confidence,
		Errors:      errors,
		Duration:    time.Since(start),
		LastUpdated: time.Now(),
	}, nil
}

// discoverByIngressAnnotations discovers tenants by analyzing Ingress annotations
func (m *MultiStrategyTenantDiscovery) discoverByIngressAnnotations(ctx context.Context) (*TenantDiscoveryResult, error) {
	start := time.Now()
	logrus.Info("🔍 Strategy 7: Discovering tenants by Ingress annotations")

	tenants := []TenantInfo{}
	errors := []string{}

	// This would require getting Ingress resources
	// For now, placeholder implementation
	logrus.Info("📊 Analyzing Ingress annotations for tenant discovery (placeholder)")

	confidence := m.calculateStrategyConfidence(len(tenants), 0, 0.6)

	return &TenantDiscoveryResult{
		Strategy:    StrategyIngressAnnotations,
		Tenants:     tenants,
		Confidence:  confidence,
		Errors:      errors,
		Duration:    time.Since(start),
		LastUpdated: time.Now(),
	}, nil
}

// discoverByPodLabels discovers tenants by analyzing Pod labels
func (m *MultiStrategyTenantDiscovery) discoverByPodLabels(ctx context.Context) (*TenantDiscoveryResult, error) {
	start := time.Now()
	logrus.Info("🔍 Strategy 8: Discovering tenants by Pod labels")

	tenants := []TenantInfo{}
	errors := []string{}

	pods, err := m.k8sClient.GetPods(ctx, "")
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to get pods: %v", err))
	} else {
		for _, pod := range pods {
			tenantInfo := m.extractTenantFromPodLabels(pod)
			if tenantInfo != nil {
				tenants = append(tenants, *tenantInfo)
			}
		}
	}

	confidence := m.calculateStrategyConfidence(len(tenants), len(pods), 0.7)

	return &TenantDiscoveryResult{
		Strategy:    StrategyPodLabels,
		Tenants:     tenants,
		Confidence:  confidence,
		Errors:      errors,
		Duration:    time.Since(start),
		LastUpdated: time.Now(),
	}, nil
}

// discoverBySecretPatterns discovers tenants by analyzing Secret patterns
func (m *MultiStrategyTenantDiscovery) discoverBySecretPatterns(ctx context.Context) (*TenantDiscoveryResult, error) {
	start := time.Now()
	logrus.Info("🔍 Strategy 9: Discovering tenants by Secret patterns")

	tenants := []TenantInfo{}
	errors := []string{}

	// This would require getting Secret resources
	// For now, placeholder implementation
	logrus.Info("📊 Analyzing Secret patterns for tenant discovery (placeholder)")

	confidence := m.calculateStrategyConfidence(len(tenants), 0, 0.65)

	return &TenantDiscoveryResult{
		Strategy:    StrategySecretPatterns,
		Tenants:     tenants,
		Confidence:  confidence,
		Errors:      errors,
		Duration:    time.Since(start),
		LastUpdated: time.Now(),
	}, nil
}

// discoverByNetworkPolicies discovers tenants by analyzing Network Policies
func (m *MultiStrategyTenantDiscovery) discoverByNetworkPolicies(ctx context.Context) (*TenantDiscoveryResult, error) {
	start := time.Now()
	logrus.Info("🔍 Strategy 10: Discovering tenants by Network Policies")

	tenants := []TenantInfo{}
	errors := []string{}

	// This would require getting NetworkPolicy resources
	// For now, placeholder implementation
	logrus.Info("📊 Analyzing Network Policies for tenant discovery (placeholder)")

	confidence := m.calculateStrategyConfidence(len(tenants), 0, 0.7)

	return &TenantDiscoveryResult{
		Strategy:    StrategyNetworkPolicies,
		Tenants:     tenants,
		Confidence:  confidence,
		Errors:      errors,
		Duration:    time.Since(start),
		LastUpdated: time.Now(),
	}, nil
}

// discoverByRBACBindings discovers tenants by analyzing RBAC bindings
func (m *MultiStrategyTenantDiscovery) discoverByRBACBindings(ctx context.Context) (*TenantDiscoveryResult, error) {
	start := time.Now()
	logrus.Info("🔍 Strategy 11: Discovering tenants by RBAC bindings")

	tenants := []TenantInfo{}
	errors := []string{}

	// This would require getting RBAC resources (RoleBindings, ClusterRoleBindings)
	// For now, placeholder implementation
	logrus.Info("📊 Analyzing RBAC bindings for tenant discovery (placeholder)")

	confidence := m.calculateStrategyConfidence(len(tenants), 0, 0.6)

	return &TenantDiscoveryResult{
		Strategy:    StrategyRBACBindings,
		Tenants:     tenants,
		Confidence:  confidence,
		Errors:      errors,
		Duration:    time.Since(start),
		LastUpdated: time.Now(),
	}, nil
}

// Helper methods for resource analysis
func (m *MultiStrategyTenantDiscovery) extractTenantFromNamespaceLabels(ns *corev1.Namespace, patterns []string) *TenantInfo {
	// Check if namespace matches tenant patterns
	for _, pattern := range patterns {
		if value, exists := ns.Labels[pattern]; exists {
			// Found a tenant label
			tenantInfo := &TenantInfo{
				Name:             value,
				Namespace:        ns.Name,
				OrgID:            value,
				Source:           StrategyNamespaceLabels,
				Confidence:       0.9,
				Labels:           ns.Labels,
				Annotations:      ns.Annotations,
				LastSeen:         time.Now(),
				DiscoveryMethods: []string{fmt.Sprintf("namespace_label_%s", pattern)},
			}

			// Extract additional tenant information from labels
			if team, exists := ns.Labels["team"]; exists {
				tenantInfo.OrgID = team
			}
			if org, exists := ns.Labels["org"]; exists {
				tenantInfo.OrgID = org
			}

			logrus.Infof("🔍 Discovered tenant from namespace labels: %s (namespace: %s, org_id: %s)",
				tenantInfo.Name, tenantInfo.Namespace, tenantInfo.OrgID)

			return tenantInfo
		}
	}

	// Check for namespace name patterns (e.g., tenant-dev, tenant-prod)
	namespacePatterns := []string{
		`^tenant-(\w+)$`,
		`^(\w+)-tenant$`,
		`^(\w+)-dev$`,
		`^(\w+)-prod$`,
		`^(\w+)-staging$`,
	}

	for _, pattern := range namespacePatterns {
		if matched, tenantName := m.matchNamespacePattern(ns.Name, pattern); matched {
			tenantInfo := &TenantInfo{
				Name:             tenantName,
				Namespace:        ns.Name,
				OrgID:            tenantName,
				Source:           StrategyNamespaceLabels,
				Confidence:       0.8,
				Labels:           ns.Labels,
				Annotations:      ns.Annotations,
				LastSeen:         time.Now(),
				DiscoveryMethods: []string{fmt.Sprintf("namespace_pattern_%s", pattern)},
			}

			logrus.Infof("🔍 Discovered tenant from namespace pattern: %s (namespace: %s, org_id: %s)",
				tenantInfo.Name, tenantInfo.Namespace, tenantInfo.OrgID)

			return tenantInfo
		}
	}

	return nil
}

func (m *MultiStrategyTenantDiscovery) extractTenantFromMimirConfig(cm *corev1.ConfigMap) *TenantInfo {
	// Look for Mimir configuration patterns
	mimirConfigPatterns := []string{
		"runtime_config",
		"limits_config",
		"distributor_config",
		"ingester_config",
		"querier_config",
		"compactor_config",
		"ruler_config",
		"alertmanager_config",
	}

	for _, pattern := range mimirConfigPatterns {
		if strings.Contains(strings.ToLower(cm.Name), pattern) {
			// Extract tenant information from config data
			for key, value := range cm.Data {
				if tenantName := m.extractTenantFromConfigValue(key, value); tenantName != "" {
					tenantInfo := &TenantInfo{
						Name:             tenantName,
						Namespace:        cm.Namespace,
						OrgID:            tenantName,
						Source:           StrategyMimirConfig,
						Confidence:       0.85,
						Labels:           cm.Labels,
						Annotations:      cm.Annotations,
						LastSeen:         time.Now(),
						DiscoveryMethods: []string{fmt.Sprintf("mimir_config_%s", pattern)},
					}

					logrus.Infof("🔍 Discovered tenant from Mimir config: %s (namespace: %s, config: %s)",
						tenantInfo.Name, tenantInfo.Namespace, cm.Name)

					return tenantInfo
				}
			}
		}
	}

	return nil
}

func (m *MultiStrategyTenantDiscovery) extractTenantFromResourceLabels(resource interface{}) *TenantInfo {
	// This is a generic method that would be implemented based on resource type
	// For now, return nil as this would need specific implementations for each resource type
	return nil
}

func (m *MultiStrategyTenantDiscovery) extractTenantFromService(svc *corev1.Service) *TenantInfo {
	// Look for tenant-related service patterns
	servicePatterns := []string{
		"tenant",
		"team",
		"org",
		"project",
		"environment",
	}

	for _, pattern := range servicePatterns {
		if strings.Contains(strings.ToLower(svc.Name), pattern) {
			// Extract tenant name from service name or labels
			tenantName := m.extractTenantFromServiceName(svc.Name)
			if tenantName != "" {
				tenantInfo := &TenantInfo{
					Name:             tenantName,
					Namespace:        svc.Namespace,
					OrgID:            tenantName,
					Source:           StrategyServiceDiscovery,
					Confidence:       0.7,
					Labels:           svc.Labels,
					Annotations:      svc.Annotations,
					LastSeen:         time.Now(),
					DiscoveryMethods: []string{fmt.Sprintf("service_pattern_%s", pattern)},
				}

				logrus.Infof("🔍 Discovered tenant from service: %s (namespace: %s, service: %s)",
					tenantInfo.Name, tenantInfo.Namespace, svc.Name)

				return tenantInfo
			}
		}
	}

	return nil
}

func (m *MultiStrategyTenantDiscovery) extractTenantFromConfigMapPatterns(cm *corev1.ConfigMap) *TenantInfo {
	// Look for tenant-related ConfigMap patterns
	configMapPatterns := []string{
		"tenant",
		"team",
		"org",
		"project",
		"environment",
		"alloy",
		"consul",
		"nginx",
	}

	for _, pattern := range configMapPatterns {
		if strings.Contains(strings.ToLower(cm.Name), pattern) {
			// Extract tenant information from ConfigMap data
			for key, value := range cm.Data {
				if tenantName := m.extractTenantFromConfigValue(key, value); tenantName != "" {
					tenantInfo := &TenantInfo{
						Name:             tenantName,
						Namespace:        cm.Namespace,
						OrgID:            tenantName,
						Source:           StrategyConfigMapPatterns,
						Confidence:       0.8,
						Labels:           cm.Labels,
						Annotations:      cm.Annotations,
						LastSeen:         time.Now(),
						DiscoveryMethods: []string{fmt.Sprintf("configmap_pattern_%s", pattern)},
					}

					logrus.Infof("🔍 Discovered tenant from ConfigMap patterns: %s (namespace: %s, configmap: %s)",
						tenantInfo.Name, tenantInfo.Namespace, cm.Name)

					return tenantInfo
				}
			}
		}
	}

	return nil
}

func (m *MultiStrategyTenantDiscovery) extractTenantFromPodLabels(pod *corev1.Pod) *TenantInfo {
	// Look for tenant-related pod labels
	tenantLabelPatterns := []string{
		"tenant",
		"team",
		"org",
		"project",
		"environment",
		"namespace",
	}

	for _, pattern := range tenantLabelPatterns {
		if value, exists := pod.Labels[pattern]; exists {
			tenantInfo := &TenantInfo{
				Name:             value,
				Namespace:        pod.Namespace,
				OrgID:            value,
				Source:           StrategyPodLabels,
				Confidence:       0.7,
				Labels:           pod.Labels,
				Annotations:      pod.Annotations,
				LastSeen:         time.Now(),
				DiscoveryMethods: []string{fmt.Sprintf("pod_label_%s", pattern)},
			}

			logrus.Infof("🔍 Discovered tenant from pod labels: %s (namespace: %s, pod: %s)",
				tenantInfo.Name, tenantInfo.Namespace, pod.Name)

			return tenantInfo
		}
	}

	return nil
}

// Utility methods for pattern matching and extraction
func (m *MultiStrategyTenantDiscovery) matchNamespacePattern(namespace, pattern string) (bool, string) {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(namespace)
	if len(matches) > 1 {
		return true, matches[1]
	}
	return false, ""
}

func (m *MultiStrategyTenantDiscovery) extractTenantFromConfigValue(key, value string) string {
	// Look for tenant patterns in configuration values
	tenantPatterns := []string{
		`tenant[_-]?(\w+)`,
		`team[_-]?(\w+)`,
		`org[_-]?(\w+)`,
		`project[_-]?(\w+)`,
		`(\w+)[_-]?tenant`,
		`(\w+)[_-]?dev`,
		`(\w+)[_-]?prod`,
		`(\w+)[_-]?staging`,
	}

	for _, pattern := range tenantPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(value)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

func (m *MultiStrategyTenantDiscovery) extractTenantFromServiceName(serviceName string) string {
	// Extract tenant name from service name patterns
	servicePatterns := []string{
		`(\w+)[_-]?service`,
		`(\w+)[_-]?api`,
		`(\w+)[_-]?app`,
		`service[_-]?(\w+)`,
		`api[_-]?(\w+)`,
		`app[_-]?(\w+)`,
	}

	for _, pattern := range servicePatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(serviceName)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// Resource getter methods (implementations)
func (m *MultiStrategyTenantDiscovery) getPodsForLabelAnalysis(ctx context.Context) ([]interface{}, error) {
	pods, err := m.k8sClient.GetPods(ctx, "")
	if err != nil {
		return nil, err
	}

	interfaces := make([]interface{}, len(pods))
	for i, pod := range pods {
		interfaces[i] = pod
	}

	return interfaces, nil
}

func (m *MultiStrategyTenantDiscovery) getServicesForLabelAnalysis(ctx context.Context) ([]interface{}, error) {
	services, err := m.k8sClient.GetServices(ctx, "")
	if err != nil {
		return nil, err
	}

	interfaces := make([]interface{}, len(services))
	for i, svc := range services {
		interfaces[i] = svc
	}

	return interfaces, nil
}

func (m *MultiStrategyTenantDiscovery) getDeploymentsForLabelAnalysis(ctx context.Context) ([]interface{}, error) {
	deployments, err := m.k8sClient.GetDeployments(ctx, "")
	if err != nil {
		return nil, err
	}

	interfaces := make([]interface{}, len(deployments))
	for i, deployment := range deployments {
		interfaces[i] = deployment
	}

	return interfaces, nil
}

func (m *MultiStrategyTenantDiscovery) getStatefulSetsForLabelAnalysis(ctx context.Context) ([]interface{}, error) {
	statefulSets, err := m.k8sClient.GetStatefulSets(ctx, "")
	if err != nil {
		return nil, err
	}

	interfaces := make([]interface{}, len(statefulSets))
	for i, statefulSet := range statefulSets {
		interfaces[i] = statefulSet
	}

	return interfaces, nil
}

// Utility methods
func (m *MultiStrategyTenantDiscovery) calculateStrategyConfidence(tenantCount, totalResources int, baseConfidence float64) float64 {
	if totalResources == 0 {
		return baseConfidence
	}

	discoveryRatio := float64(tenantCount) / float64(totalResources)
	return baseConfidence * discoveryRatio
}

func (m *MultiStrategyTenantDiscovery) consolidateTenantResults(results map[TenantDiscoveryStrategy]*TenantDiscoveryResult) []TenantInfo {
	// Create a map to track unique tenants by namespace and name
	tenantMap := make(map[string]*TenantInfo)

	// Process results from all strategies
	for strategy, result := range results {
		for _, tenant := range result.Tenants {
			key := fmt.Sprintf("%s:%s", tenant.Namespace, tenant.Name)

			if existingTenant, exists := tenantMap[key]; exists {
				// Merge tenant information from multiple strategies
				existingTenant.Confidence = (existingTenant.Confidence + tenant.Confidence) / 2
				existingTenant.DiscoveryMethods = append(existingTenant.DiscoveryMethods, tenant.DiscoveryMethods...)

				// Update last seen if this discovery is more recent
				if tenant.LastSeen.After(existingTenant.LastSeen) {
					existingTenant.LastSeen = tenant.LastSeen
				}

				// Merge labels and annotations
				for k, v := range tenant.Labels {
					existingTenant.Labels[k] = v
				}
				for k, v := range tenant.Annotations {
					existingTenant.Annotations[k] = v
				}
			} else {
				// Create new tenant entry
				tenantMap[key] = &tenant
			}
		}
	}

	// Convert map back to slice
	consolidatedTenants := make([]TenantInfo, 0, len(tenantMap))
	for _, tenant := range tenantMap {
		consolidatedTenants = append(consolidatedTenants, *tenant)
	}

	logrus.Infof("📊 Consolidated %d unique tenants from %d strategies", len(consolidatedTenants), len(results))

	return consolidatedTenants
}

func (m *MultiStrategyTenantDiscovery) crossValidateTenants(ctx context.Context, tenants []TenantInfo) []TenantInfo {
	// Perform cross-validation to increase confidence
	validatedTenants := make([]TenantInfo, 0, len(tenants))

	for _, tenant := range tenants {
		// Increase confidence if tenant is discovered by multiple methods
		if len(tenant.DiscoveryMethods) > 1 {
			tenant.Confidence = tenant.Confidence * 1.2 // Boost confidence by 20%
			if tenant.Confidence > 1.0 {
				tenant.Confidence = 1.0
			}
		}

		// Validate tenant by checking if namespace exists
		if ns, err := m.k8sClient.GetNamespace(ctx, tenant.Namespace); err == nil && ns != nil {
			tenant.Confidence = tenant.Confidence * 1.1 // Boost confidence by 10%
		} else {
			tenant.Confidence = tenant.Confidence * 0.8 // Reduce confidence by 20%
		}

		// Only include tenants with reasonable confidence
		if tenant.Confidence >= 0.5 {
			validatedTenants = append(validatedTenants, tenant)
		}
	}

	logrus.Infof("✅ Cross-validation completed: %d tenants validated out of %d", len(validatedTenants), len(tenants))

	return validatedTenants
}
