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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Engine handles auto-discovery of Mimir components and tenant namespaces
type Engine struct {
	k8sClient           *k8s.Client
	config              *config.Config
	environmentDetector *EnvironmentDetector
	// Compiled regex patterns for performance
	compiledPatterns *CompiledPatterns
	// Multi-strategy tenant discovery
	multiStrategyDiscovery *MultiStrategyTenantDiscovery
	// Multi-strategy Mimir discovery
	multiStrategyMimirDiscovery *MultiStrategyMimirDiscovery
}

// CompiledPatterns holds pre-compiled regex patterns for efficient matching
type CompiledPatterns struct {
	NamespacePatterns   []*regexp.Regexp
	ComponentPatterns   map[string][]*regexp.Regexp
	ServicePatterns     []*regexp.Regexp
	ConfigMapPatterns   []*regexp.Regexp
	MetricsPathPatterns []*regexp.Regexp
}

// ValidationResult represents the confidence level of a discovery match
type ValidationResult struct {
	ConfidenceScore float64                `json:"confidence_score"`
	MatchedBy       []string               `json:"matched_by"`
	ValidationInfo  map[string]interface{} `json:"validation_info"`
}

// MimirComponent represents a discovered Mimir component
type MimirComponent struct {
	Name             string            `json:"name"`
	Type             string            `json:"type"`
	Namespace        string            `json:"namespace"`
	Status           string            `json:"status"`
	Replicas         int32             `json:"replicas"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`
	Image            string            `json:"image"`
	Version          string            `json:"version"`
	ServiceEndpoints []string          `json:"service_endpoints"`
	MetricsEndpoints []string          `json:"metrics_endpoints"`
	ConfigMaps       []string          `json:"config_maps"`
	Validation       ValidationResult  `json:"validation"`
}

// TenantNamespace represents a discovered tenant namespace
type TenantNamespace struct {
	Name           string                 `json:"name"`
	Labels         map[string]string      `json:"labels"`
	Annotations    map[string]string      `json:"annotations"`
	AlloyConfig    *AlloyConfig           `json:"alloy_config"`
	ConsulConfig   *ConsulConfig          `json:"consul_config"`
	NginxConfig    *NginxConfig           `json:"nginx_config"`
	MimirLimits    map[string]interface{} `json:"mimir_limits"`
	ComponentCount int                    `json:"component_count"`
	Status         string                 `json:"status"`
	Validation     ValidationResult       `json:"validation"`
}

// WorkloadInfo represents any type of Kubernetes workload (Deployment, StatefulSet, DaemonSet, etc.)
type WorkloadInfo struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"` // "Deployment", "StatefulSet", "DaemonSet", "ReplicaSet"
	Namespace   string            `json:"namespace"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Image       string            `json:"image"`
	Version     string            `json:"version"`
	Replicas    int32             `json:"replicas"`
	Status      string            `json:"status"`
}

// AlloyConfig represents Alloy configuration
type AlloyConfig struct {
	Workloads     []WorkloadInfo `json:"workloads"` // Can be Deployment, StatefulSet, DaemonSet
	Replicas      int32          `json:"replicas"`
	ScrapeConfigs []string       `json:"scrape_configs"`
	Targets       []string       `json:"targets"`
	Image         string         `json:"image"`
	Version       string         `json:"version"`
}

// ConsulConfig represents Consul configuration
type ConsulConfig struct {
	Workloads []WorkloadInfo `json:"workloads"` // Can be Deployment, StatefulSet
	Replicas  int32          `json:"replicas"`
	Endpoints []string       `json:"endpoints"`
	Services  []string       `json:"services"`
	Image     string         `json:"image"`
	Version   string         `json:"version"`
}

// NginxConfig represents NGINX configuration
type NginxConfig struct {
	Workloads []WorkloadInfo `json:"workloads"` // Can be Deployment, DaemonSet
	Replicas  int32          `json:"replicas"`
	Upstreams []string       `json:"upstreams"`
	Routes    []string       `json:"routes"`
	Image     string         `json:"image"`
	Version   string         `json:"version"`
}

// ScrapeConfig represents a scrape configuration
type ScrapeConfig struct {
	JobName     string            `json:"job_name"`
	Targets     []string          `json:"targets"`
	Interval    string            `json:"interval"`
	MetricsPath string            `json:"metrics_path"`
	Labels      map[string]string `json:"labels"`
}

// DiscoveryResult holds the complete discovery results
type DiscoveryResult struct {
	MimirComponents  []MimirComponent  `json:"mimir_components"`
	TenantNamespaces []TenantNamespace `json:"tenant_namespaces"`
	ConfigMaps       []ConfigMapInfo   `json:"config_maps"`
	Environment      *EnvironmentInfo  `json:"environment"`
	AutoDiscoveredNS string            `json:"auto_discovered_namespace"`
	LastUpdated      time.Time         `json:"last_updated"`
}

// ConfigMapInfo represents discovered ConfigMap information
type ConfigMapInfo struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Data      map[string]string `json:"data"`
	Labels    map[string]string `json:"labels"`
}

// NewEngine creates a new discovery engine
func NewEngine() *Engine {
	k8sClient, err := k8s.NewClient()
	if err != nil {
		logrus.Fatalf("Failed to create k8s client: %v", err)
	}

	engine := &Engine{
		k8sClient:           k8sClient,
		config:              config.Get(),
		environmentDetector: NewEnvironmentDetector(k8sClient),
	}

	// Compile regex patterns for performance
	engine.compilePatterns()

	// Initialize multi-strategy tenant discovery
	engine.multiStrategyDiscovery = NewMultiStrategyTenantDiscovery(engine)

	// Initialize multi-strategy Mimir discovery
	engine.multiStrategyMimirDiscovery = NewMultiStrategyMimirDiscovery(engine)

	return engine
}

// compilePatterns pre-compiles all regex patterns for efficient matching
func (e *Engine) compilePatterns() {
	e.compiledPatterns = &CompiledPatterns{
		ComponentPatterns: make(map[string][]*regexp.Regexp),
	}

	// Get discovery config from mimir config
	discoveryConfig := e.config.Mimir.Discovery

	// Compile namespace patterns
	for _, pattern := range discoveryConfig.NamespacePatterns {
		if regex, err := regexp.Compile(pattern); err == nil {
			e.compiledPatterns.NamespacePatterns = append(e.compiledPatterns.NamespacePatterns, regex)
		} else {
			logrus.Warnf("Invalid namespace pattern: %s", pattern)
		}
	}

	// Compile component patterns
	for componentType, patterns := range discoveryConfig.ComponentPatterns {
		for _, pattern := range patterns {
			if regex, err := regexp.Compile(pattern); err == nil {
				e.compiledPatterns.ComponentPatterns[componentType] = append(
					e.compiledPatterns.ComponentPatterns[componentType], regex)
			} else {
				logrus.Warnf("Invalid component pattern for %s: %s", componentType, pattern)
			}
		}
	}

	// Compile service patterns
	for _, pattern := range discoveryConfig.ServicePatterns {
		if regex, err := regexp.Compile(pattern); err == nil {
			e.compiledPatterns.ServicePatterns = append(e.compiledPatterns.ServicePatterns, regex)
		} else {
			logrus.Warnf("Invalid service pattern: %s", pattern)
		}
	}

	// Compile ConfigMap patterns
	for _, pattern := range discoveryConfig.ConfigMapPatterns {
		if regex, err := regexp.Compile(pattern); err == nil {
			e.compiledPatterns.ConfigMapPatterns = append(e.compiledPatterns.ConfigMapPatterns, regex)
		} else {
			logrus.Warnf("Invalid ConfigMap pattern: %s", pattern)
		}
	}

	// Compile metrics path patterns
	for _, path := range e.config.Mimir.API.MetricsPaths {
		if regex, err := regexp.Compile(path); err == nil {
			e.compiledPatterns.MetricsPathPatterns = append(e.compiledPatterns.MetricsPathPatterns, regex)
		}
	}
}

// GetK8sClient returns the Kubernetes client
func (e *Engine) GetK8sClient() *k8s.Client {
	return e.k8sClient
}

// GetConfig returns the configuration
func (e *Engine) GetConfig() *config.Config {
	return e.config
}

// AutoDiscoverMimirNamespace automatically discovers the Mimir namespace
func (e *Engine) AutoDiscoverMimirNamespace(ctx context.Context) (string, error) {
	logrus.Info("Auto-discovering Mimir namespace...")

	// Get all namespaces
	namespaces, err := e.k8sClient.GetNamespaces(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get namespaces: %w", err)
	}

	var candidates []NamespaceCandidate

	for _, ns := range namespaces.Items {
		candidate := e.evaluateNamespaceForMimir(ctx, &ns)
		if candidate.Score > 0 {
			candidates = append(candidates, candidate)
		}
	}

	// Sort by confidence score and return the best match
	if len(candidates) == 0 {
		return "", fmt.Errorf("no Mimir namespace found")
	}

	// Sort candidates by score (highest first)
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].Score > candidates[i].Score {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	bestCandidate := candidates[0]
	logrus.Infof("Auto-discovered Mimir namespace: %s (confidence: %.2f)",
		bestCandidate.Name, bestCandidate.Score)

	return bestCandidate.Name, nil
}

// NamespaceCandidate represents a potential Mimir namespace with confidence score
type NamespaceCandidate struct {
	Name    string   `json:"name"`
	Score   float64  `json:"score"`
	Reasons []string `json:"reasons"`
}

// evaluateNamespaceForMimir scores a namespace based on multiple criteria
func (e *Engine) evaluateNamespaceForMimir(ctx context.Context, ns *corev1.Namespace) NamespaceCandidate {
	candidate := NamespaceCandidate{
		Name:    ns.Name,
		Score:   0,
		Reasons: []string{},
	}

	// 1. Check namespace name patterns
	for _, pattern := range e.compiledPatterns.NamespacePatterns {
		if pattern.MatchString(ns.Name) {
			candidate.Score += 20
			candidate.Reasons = append(candidate.Reasons, fmt.Sprintf("namespace name matches pattern: %s", pattern.String()))
			break
		}
	}

	// 2. Check namespace labels
	discoveryConfig := e.config.Mimir.Discovery
	for _, labelSelector := range discoveryConfig.NamespaceLabels {
		if value, exists := ns.Labels[labelSelector.Key]; exists {
			for _, expectedValue := range labelSelector.Values {
				if value == expectedValue {
					candidate.Score += 15
					candidate.Reasons = append(candidate.Reasons, fmt.Sprintf("namespace label %s=%s", labelSelector.Key, value))
				}
			}
		}
	}

	// 3. Check for Mimir deployments in this namespace
	deployments, err := e.k8sClient.GetDeployments(ctx, ns.Name, metav1.ListOptions{})
	if err == nil {
		mimirDeployments := 0
		for _, deployment := range deployments.Items {
			if e.isMimirComponentAdvanced(deployment.Name, deployment.Labels, deployment.Annotations) {
				mimirDeployments++
				candidate.Score += 10
			}
		}
		if mimirDeployments > 0 {
			candidate.Reasons = append(candidate.Reasons, fmt.Sprintf("found %d Mimir deployments", mimirDeployments))
		}
	}

	// 4. Check for Mimir services
	services, err := e.k8sClient.GetServices(ctx, ns.Name, metav1.ListOptions{})
	if err == nil {
		mimirServices := 0
		for _, service := range services.Items {
			if e.isMimirServiceAdvanced(service.Name, service.Labels, service.Annotations) {
				mimirServices++
				candidate.Score += 8
			}
		}
		if mimirServices > 0 {
			candidate.Reasons = append(candidate.Reasons, fmt.Sprintf("found %d Mimir services", mimirServices))
		}
	}

	// 5. Check for Mimir ConfigMaps
	configMaps, err := e.k8sClient.GetConfigMaps(ctx, ns.Name, metav1.ListOptions{})
	if err == nil {
		mimirConfigMaps := 0
		for _, cm := range configMaps.Items {
			if e.isMimirConfigMapAdvanced(cm.Name, cm.Labels, cm.Data) {
				mimirConfigMaps++
				candidate.Score += 5
			}
		}
		if mimirConfigMaps > 0 {
			candidate.Reasons = append(candidate.Reasons, fmt.Sprintf("found %d Mimir ConfigMaps", mimirConfigMaps))
		}
	}

	return candidate
}

// DiscoverAll performs complete discovery of Mimir and tenant components
func (e *Engine) DiscoverAll(ctx context.Context) (*DiscoveryResult, error) {
	logrus.Info("🚀 Starting intelligent auto-discovery of Mimir components and tenant namespaces")
	logrus.Infof("📋 [DISCOVERY] Configuration - Namespace: %s, Auto-detect: %v", e.config.Mimir.Namespace, e.config.Mimir.Discovery.AutoDetect)

	result := &DiscoveryResult{
		LastUpdated: time.Now(),
	}

	// Auto-discover Mimir namespace if not configured
	mimirNamespace := e.config.Mimir.Namespace
	if mimirNamespace == "" || mimirNamespace == "auto" {
		logrus.Infof("🔍 [DISCOVERY] Auto-discovering Mimir namespace...")
		autoNS, err := e.AutoDiscoverMimirNamespace(ctx)
		if err != nil {
			logrus.Warnf("⚠️ [DISCOVERY] Failed to auto-discover Mimir namespace: %v", err)
			mimirNamespace = "mimir" // fallback
			logrus.Infof("📋 [DISCOVERY] Using fallback namespace: %s", mimirNamespace)
		} else {
			mimirNamespace = autoNS
			result.AutoDiscoveredNS = autoNS
			logrus.Infof("✅ [DISCOVERY] Auto-discovered Mimir namespace: %s", mimirNamespace)
		}
	}

	// Update config with discovered namespace
	e.config.Mimir.Namespace = mimirNamespace
	logrus.Infof("📋 [DISCOVERY] Using namespace for discovery: %s", mimirNamespace)

	// Discover Mimir components with enhanced validation
	logrus.Infof("🔍 [DISCOVERY] Starting Mimir component discovery in namespace: %s", mimirNamespace)
	mimirComponents, err := e.discoverMimirComponentsAdvanced(ctx, mimirNamespace)
	if err != nil {
		logrus.Errorf("❌ [DISCOVERY] Failed to discover Mimir components: %v", err)
		return nil, fmt.Errorf("failed to discover Mimir components: %w", err)
	}
	result.MimirComponents = mimirComponents
	logrus.Infof("✅ [DISCOVERY] Mimir component discovery completed: %d components found", len(mimirComponents))

	// Log details about discovered components
	for i, component := range mimirComponents {
		logrus.Infof("📋 [DISCOVERY] Component %d/%d: %s (type: %s, confidence: %.1f%%, status: %s)",
			i+1, len(mimirComponents), component.Name, component.Type, component.Validation.ConfidenceScore, component.Status)
	}

	// Discover tenant namespaces
	logrus.Infof("🔍 [DISCOVERY] Starting tenant namespace discovery...")
	tenantNamespaces, err := e.discoverTenantNamespaces(ctx)
	if err != nil {
		logrus.Errorf("❌ [DISCOVERY] Failed to discover tenant namespaces: %v", err)
		return nil, fmt.Errorf("failed to discover tenant namespaces: %w", err)
	}
	result.TenantNamespaces = tenantNamespaces
	logrus.Infof("✅ [DISCOVERY] Tenant namespace discovery completed: %d tenants found", len(tenantNamespaces))

	// Log details about discovered tenants
	for i, tenant := range tenantNamespaces {
		logrus.Infof("📋 [DISCOVERY] Tenant %d/%d: %s (status: %s, components: %d, confidence: %.1f%%)",
			i+1, len(tenantNamespaces), tenant.Name, tenant.Status, tenant.ComponentCount, tenant.Validation.ConfidenceScore)
	}

	// Discover ConfigMaps
	logrus.Infof("🔍 [DISCOVERY] Starting ConfigMap discovery...")
	configMaps, err := e.discoverConfigMaps(ctx)
	if err != nil {
		logrus.Errorf("❌ [DISCOVERY] Failed to discover ConfigMaps: %v", err)
		return nil, fmt.Errorf("failed to discover ConfigMaps: %w", err)
	}
	result.ConfigMaps = configMaps
	logrus.Infof("✅ [DISCOVERY] ConfigMap discovery completed: %d ConfigMaps found", len(configMaps))

	// Log details about discovered ConfigMaps
	for i, configMap := range configMaps {
		logrus.Infof("📋 [DISCOVERY] ConfigMap %d/%d: %s (namespace: %s, data keys: %d)",
			i+1, len(configMaps), configMap.Name, configMap.Namespace, len(configMap.Data))
	}

	// Detect environment information
	logrus.Infof("🔍 [DISCOVERY] Starting environment detection...")
	environment, err := e.environmentDetector.DetectEnvironment(ctx, mimirNamespace)
	if err != nil {
		logrus.Warnf("⚠️ [DISCOVERY] Failed to detect environment: %v", err)
		// Don't fail the entire discovery if environment detection fails
		environment = &EnvironmentInfo{
			DataSource:      "unknown",
			IsProduction:    false,
			DetectedTenants: []DetectedTenant{},
			LastUpdated:     time.Now(),
		}
		logrus.Infof("📋 [DISCOVERY] Using fallback environment configuration")
	}
	result.Environment = environment
	logrus.Infof("✅ [DISCOVERY] Environment detection completed - Production: %v, Data source: %s",
		environment.IsProduction, environment.DataSource)

	// Generate discovery summary and recommendations
	logrus.Infof("📊 [DISCOVERY] Discovery Summary:")
	logrus.Infof("   - Mimir Components: %d", len(mimirComponents))
	logrus.Infof("   - Tenant Namespaces: %d", len(tenantNamespaces))
	logrus.Infof("   - ConfigMaps: %d", len(configMaps))
	logrus.Infof("   - Environment: %s (Production: %v)", environment.DataSource, environment.IsProduction)
	logrus.Infof("   - Namespace: %s", mimirNamespace)

	// Provide recommendations based on discovery results
	if len(mimirComponents) == 0 {
		logrus.Warnf("⚠️ [DISCOVERY] No Mimir components found - this may indicate:")
		logrus.Warnf("   - Mimir is not deployed in the cluster")
		logrus.Warnf("   - Mimir is deployed in a different namespace")
		logrus.Warnf("   - Discovery patterns need adjustment")
	} else {
		// Check confidence scores
		lowConfidenceCount := 0
		for _, component := range mimirComponents {
			if component.Validation.ConfidenceScore < 50 {
				lowConfidenceCount++
			}
		}
		if lowConfidenceCount > 0 {
			logrus.Warnf("⚠️ [DISCOVERY] %d components have low confidence scores - review discovery patterns", lowConfidenceCount)
		}
	}

	if len(tenantNamespaces) == 0 {
		logrus.Infof("ℹ️ [DISCOVERY] No tenant namespaces found - this is normal if no multi-tenant setup exists")
	}

	logrus.Infof("✅ [DISCOVERY] Enhanced discovery completed successfully in %v", time.Since(result.LastUpdated))

	return result, nil
}

// discoverMimirComponentsAdvanced discovers all Mimir components in the configured namespace
func (e *Engine) discoverMimirComponentsAdvanced(ctx context.Context, namespace string) ([]MimirComponent, error) {
	logrus.Infof("Discovering Mimir components in namespace: %s", namespace)

	var components []MimirComponent

	// Discover Deployments
	deployments, err := e.k8sClient.GetDeployments(ctx, namespace, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments: %w", err)
	}

	for _, deployment := range deployments.Items {
		// Check if this is a Mimir component
		if e.isMimirComponentAdvanced(deployment.Name, deployment.Labels, deployment.Annotations) {
			component := MimirComponent{
				Name:        deployment.Name,
				Type:        getComponentType(deployment.Name),
				Namespace:   deployment.Namespace,
				Status:      getDeploymentStatus(&deployment),
				Replicas:    *deployment.Spec.Replicas,
				Labels:      deployment.Labels,
				Annotations: deployment.Annotations,
			}

			// Extract image and version
			if len(deployment.Spec.Template.Spec.Containers) > 0 {
				container := deployment.Spec.Template.Spec.Containers[0]
				component.Image = container.Image
				component.Version = extractVersion(container.Image)
			}

			// Discover services for this component
			services, err := e.k8sClient.GetServices(ctx, namespace, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app.kubernetes.io/name=%s", deployment.Name),
			})
			if err == nil {
				for _, service := range services.Items {
					for _, port := range service.Spec.Ports {
						endpoint := fmt.Sprintf("%s.%s.svc.cluster.local:%d", service.Name, service.Namespace, port.Port)
						component.ServiceEndpoints = append(component.ServiceEndpoints, endpoint)
					}
				}
			}

			// Discover metrics endpoints for this component
			metricsEndpoints := []string{}
			if len(deployment.Spec.Template.Spec.Containers) > 0 && deployment.Spec.Template.Spec.Containers[0].Ports != nil {
				for _, port := range deployment.Spec.Template.Spec.Containers[0].Ports {
					if e.isMetricsPath(port.Name) {
						metricsEndpoints = append(metricsEndpoints, fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", deployment.Name, namespace, port.ContainerPort))
					}
				}
			}
			component.MetricsEndpoints = metricsEndpoints

			// Perform cross-validation
			component.Validation = e.validateMimirComponentCrossReference(ctx, &component, namespace)

			components = append(components, component)
		}
	}

	// Discover StatefulSets
	statefulSets, err := e.k8sClient.GetStatefulSets(ctx, namespace, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get statefulsets: %w", err)
	}

	for _, statefulSet := range statefulSets.Items {
		if e.isMimirComponentAdvanced(statefulSet.Name, statefulSet.Labels, statefulSet.Annotations) {
			component := MimirComponent{
				Name:        statefulSet.Name,
				Type:        getComponentType(statefulSet.Name),
				Namespace:   statefulSet.Namespace,
				Status:      getStatefulSetStatus(&statefulSet),
				Replicas:    *statefulSet.Spec.Replicas,
				Labels:      statefulSet.Labels,
				Annotations: statefulSet.Annotations,
			}

			if len(statefulSet.Spec.Template.Spec.Containers) > 0 {
				container := statefulSet.Spec.Template.Spec.Containers[0]
				component.Image = container.Image
				component.Version = extractVersion(container.Image)
			}

			// Discover services for this component
			services, err := e.k8sClient.GetServices(ctx, namespace, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app.kubernetes.io/name=%s", statefulSet.Name),
			})
			if err == nil {
				for _, service := range services.Items {
					for _, port := range service.Spec.Ports {
						endpoint := fmt.Sprintf("%s.%s.svc.cluster.local:%d", service.Name, service.Namespace, port.Port)
						component.ServiceEndpoints = append(component.ServiceEndpoints, endpoint)
					}
				}
			}

			// Discover metrics endpoints for this component
			metricsEndpoints := []string{}
			if len(statefulSet.Spec.Template.Spec.Containers) > 0 && statefulSet.Spec.Template.Spec.Containers[0].Ports != nil {
				for _, port := range statefulSet.Spec.Template.Spec.Containers[0].Ports {
					if e.isMetricsPath(port.Name) {
						metricsEndpoints = append(metricsEndpoints, fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", statefulSet.Name, namespace, port.ContainerPort))
					}
				}
			}
			component.MetricsEndpoints = metricsEndpoints

			// Perform cross-validation
			component.Validation = e.validateMimirComponentCrossReference(ctx, &component, namespace)

			components = append(components, component)
		}
	}

	return components, nil
}

// discoverTenantNamespaces discovers all tenant namespaces and their components
func (e *Engine) discoverTenantNamespaces(ctx context.Context) ([]TenantNamespace, error) {
	logrus.Info("Discovering tenant namespaces")

	var tenants []TenantNamespace

	// Get all namespaces
	namespaces, err := e.k8sClient.GetNamespaces(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get namespaces: %w", err)
	}

	for _, namespace := range namespaces.Items {
		// Check if this is a tenant namespace
		if isTenantNamespace(namespace.Labels, e.config.K8s) {
			tenant, err := e.discoverTenantNamespace(ctx, &namespace)
			if err != nil {
				logrus.Warnf("Failed to discover tenant namespace %s: %v", namespace.Name, err)
				continue
			}
			tenants = append(tenants, *tenant)
		}
	}

	return tenants, nil
}

// discoverTenantNamespace discovers components within a specific tenant namespace
func (e *Engine) discoverTenantNamespace(ctx context.Context, namespace *corev1.Namespace) (*TenantNamespace, error) {
	tenant := &TenantNamespace{
		Name:        namespace.Name,
		Labels:      namespace.Labels,
		Annotations: namespace.Annotations,
		Status:      string(namespace.Status.Phase),
	}

	// Discover Alloy configuration
	alloyConfig, err := e.discoverAlloyConfig(ctx, namespace.Name)
	if err != nil {
		logrus.Warnf("Failed to discover Alloy config for %s: %v", namespace.Name, err)
	} else {
		tenant.AlloyConfig = alloyConfig
	}

	// Discover Consul configuration
	consulConfig, err := e.discoverConsulConfig(ctx, namespace.Name)
	if err != nil {
		logrus.Warnf("Failed to discover Consul config for %s: %v", namespace.Name, err)
	} else {
		tenant.ConsulConfig = consulConfig
	}

	// Discover NGINX configuration
	nginxConfig, err := e.discoverNginxConfig(ctx, namespace.Name)
	if err != nil {
		logrus.Warnf("Failed to discover NGINX config for %s: %v", namespace.Name, err)
	} else {
		tenant.NginxConfig = nginxConfig
	}

	// Get Mimir limits for this tenant
	mimirLimits, err := e.getTenantMimirLimits(ctx, namespace.Name)
	if err != nil {
		logrus.Warnf("Failed to get Mimir limits for %s: %v", namespace.Name, err)
	} else {
		tenant.MimirLimits = mimirLimits
	}

	// Count components
	tenant.ComponentCount = e.countTenantComponents(ctx, namespace.Name)

	return tenant, nil
}

// discoverConfigMaps discovers relevant ConfigMaps
func (e *Engine) discoverConfigMaps(ctx context.Context) ([]ConfigMapInfo, error) {
	var configMaps []ConfigMapInfo

	// Discover ConfigMaps in Mimir namespace
	mimirConfigMaps, err := e.k8sClient.GetConfigMaps(ctx, e.config.Mimir.Namespace, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Mimir ConfigMaps: %w", err)
	}

	for _, cm := range mimirConfigMaps.Items {
		if e.isMimirConfigMapAdvanced(cm.Name, cm.Labels, cm.Data) {
			configMaps = append(configMaps, ConfigMapInfo{
				Name:      cm.Name,
				Namespace: cm.Namespace,
				Data:      cm.Data,
				Labels:    cm.Labels,
			})
		}
	}

	return configMaps, nil
}

// Helper functions
func isMimirComponent(labels map[string]string) bool {
	// Check for Mimir-specific labels
	for key, value := range labels {
		if strings.Contains(strings.ToLower(key), "mimir") ||
			strings.Contains(strings.ToLower(value), "mimir") ||
			strings.Contains(strings.ToLower(key), "cortex") ||
			strings.Contains(strings.ToLower(value), "cortex") {
			return true
		}
	}
	return false
}

func isTenantNamespace(labels map[string]string, k8sConfig config.K8sConfig) bool {
	// Check if namespace has tenant label
	if value, exists := labels[k8sConfig.TenantLabel]; exists {
		return strings.HasPrefix(value, k8sConfig.TenantPrefix)
	}
	return false
}

func getComponentType(name string) string {
	name = strings.ToLower(name)
	switch {
	case strings.Contains(name, "distributor"):
		return "distributor"
	case strings.Contains(name, "ingester"):
		return "ingester"
	case strings.Contains(name, "querier"):
		return "querier"
	case strings.Contains(name, "compactor"):
		return "compactor"
	case strings.Contains(name, "ruler"):
		return "ruler"
	case strings.Contains(name, "alertmanager"):
		return "alertmanager"
	default:
		return "unknown"
	}
}

func extractVersion(image string) string {
	parts := strings.Split(image, ":")
	if len(parts) > 1 {
		return parts[1]
	}
	return "latest"
}

func isRelevantConfigMap(name string) bool {
	relevantNames := []string{
		"runtime-overrides",
		"mimir-config",
		"cortex-config",
		"limits-config",
	}

	name = strings.ToLower(name)
	for _, relevant := range relevantNames {
		if strings.Contains(name, relevant) {
			return true
		}
	}
	return false
}

// isAlloyConfigMap checks if a ConfigMap contains Alloy configuration
func isAlloyConfigMap(name string) bool {
	alloyNames := []string{
		"alloy",
		"grafana-agent",
		"agent-config",
		"monitoring-config",
		"scrape-config",
	}

	name = strings.ToLower(name)
	for _, alloyName := range alloyNames {
		if strings.Contains(name, alloyName) {
			return true
		}
	}
	return false
}

// isNginxConfigMap checks if a ConfigMap contains NGINX configuration
func isNginxConfigMap(name string) bool {
	nginxNames := []string{
		"nginx",
		"nginx-config",
		"nginx-upstream",
		"nginx-routes",
	}

	name = strings.ToLower(name)
	for _, nginxName := range nginxNames {
		if strings.Contains(name, nginxName) {
			return true
		}
	}
	return false
}

// parseAlloyConfig parses Alloy configuration and extracts tenant information
func (e *Engine) parseAlloyConfig(configData map[string]string) []string {
	var scrapeConfigs []string

	for key, value := range configData {
		// Skip non-config files
		if !strings.HasSuffix(key, ".yaml") && !strings.HasSuffix(key, ".yml") && !strings.HasSuffix(key, ".river") {
			continue
		}

		// Parse the configuration content for tenant information
		tenantInfo := e.extractTenantFromConfig(value)
		if tenantInfo != "" {
			scrapeConfigs = append(scrapeConfigs, tenantInfo)
		}
	}

	return scrapeConfigs
}

// extractTenantFromConfig extracts tenant information from Alloy/Grafana Agent config
func (e *Engine) extractTenantFromConfig(config string) string {
	lines := strings.Split(config, "\n")
	var tenant, orgID string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for X-Scope-OrgID header configuration
		if strings.Contains(strings.ToLower(line), "x-scope-orgid") {
			// Extract the tenant/org ID value
			if strings.Contains(line, ":") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					orgID = strings.Trim(strings.TrimSpace(parts[1]), `"'`)
				}
			}
		}

		// Look for tenant label or configuration
		if strings.Contains(strings.ToLower(line), "tenant") && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				tenant = strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			}
		}

		// Look for external_labels that might contain tenant info
		if strings.Contains(strings.ToLower(line), "external_labels") {
			// This indicates the start of external labels section
			continue
		}

		// Look for job_name which might indicate the tenant
		if strings.Contains(strings.ToLower(line), "job_name") && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				jobName := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
				if tenant == "" && jobName != "" {
					tenant = jobName
				}
			}
		}
	}

	// Return the most specific tenant identifier we found
	if orgID != "" {
		return fmt.Sprintf("orgid:%s", orgID)
	}
	if tenant != "" {
		return fmt.Sprintf("tenant:%s", tenant)
	}

	return ""
}

// parseNginxConfig parses NGINX configuration from ConfigMap data
func (e *Engine) parseNginxConfig(configData map[string]string) ([]string, []string) {
	var upstreams []string
	var routes []string

	for key, value := range configData {
		// Skip non-config files
		if !strings.HasSuffix(key, ".yaml") && !strings.HasSuffix(key, ".yml") && !strings.HasSuffix(key, ".river") {
			continue
		}

		// Look for upstreams
		if strings.Contains(strings.ToLower(key), "upstream") {
			upstreams = append(upstreams, value)
		}

		// Look for routes
		if strings.Contains(strings.ToLower(key), "route") {
			routes = append(routes, value)
		}
	}

	return upstreams, routes
}

// parseMimirOverrides parses Mimir runtime overrides for tenant-specific limits
func (e *Engine) parseMimirOverrides(config string, tenantName string) map[string]interface{} {
	limits := make(map[string]interface{})

	lines := strings.Split(config, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for tenant-specific overrides
		if strings.Contains(strings.ToLower(line), fmt.Sprintf("tenant:%s", tenantName)) {
			// This line is a tenant override
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				overrideKey := strings.TrimSpace(parts[0])
				overrideValue := strings.TrimSpace(parts[1])
				limits[overrideKey] = overrideValue
			}
		}
	}

	return limits
}

// Placeholder implementations for methods that will be implemented in separate files
func (e *Engine) discoverAlloyConfig(ctx context.Context, namespace string) (*AlloyConfig, error) {
	logrus.Infof("Discovering Alloy configuration in namespace: %s", namespace)

	alloyConfig := &AlloyConfig{
		Workloads:     []WorkloadInfo{},
		ScrapeConfigs: []string{},
		Targets:       []string{},
	}

	// Search for Alloy workloads using comprehensive discovery
	alloySelectors := []string{
		"app.kubernetes.io/name=alloy",
		"app.kubernetes.io/name=grafana-agent",
		"app=alloy",
		"app=grafana-agent",
		"component=alloy",
	}

	workloads, err := e.DiscoverWorkloadsByLabels(ctx, namespace, alloySelectors...)
	if err != nil {
		return nil, fmt.Errorf("failed to discover Alloy workloads: %w", err)
	}

	alloyConfig.Workloads = workloads

	// Set primary config from first workload found
	if len(workloads) > 0 {
		primary := workloads[0]
		alloyConfig.Replicas = primary.Replicas
		alloyConfig.Image = primary.Image
		alloyConfig.Version = primary.Version
	}

	// Look for Alloy configuration in ConfigMaps
	configMaps, err := e.k8sClient.GetConfigMaps(ctx, namespace, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ConfigMaps: %w", err)
	}

	for _, cm := range configMaps.Items {
		if isAlloyConfigMap(cm.Name) {
			scrapeConfigs := e.parseAlloyConfig(cm.Data)
			alloyConfig.ScrapeConfigs = append(alloyConfig.ScrapeConfigs, scrapeConfigs...)
		}
	}

	return alloyConfig, nil
}

func (e *Engine) discoverConsulConfig(ctx context.Context, namespace string) (*ConsulConfig, error) {
	logrus.Infof("Discovering Consul configuration in namespace: %s", namespace)

	consulConfig := &ConsulConfig{
		Workloads: []WorkloadInfo{},
		Endpoints: []string{},
		Services:  []string{},
	}

	// Search for Consul workloads using comprehensive discovery
	consulSelectors := []string{
		"app.kubernetes.io/name=consul",
		"app=consul",
		"component=consul",
	}

	workloads, err := e.DiscoverWorkloadsByLabels(ctx, namespace, consulSelectors...)
	if err != nil {
		return nil, fmt.Errorf("failed to discover Consul workloads: %w", err)
	}

	consulConfig.Workloads = workloads

	// Set primary config from first workload found
	if len(workloads) > 0 {
		primary := workloads[0]
		consulConfig.Replicas = primary.Replicas
		consulConfig.Image = primary.Image
		consulConfig.Version = primary.Version
	}

	// Look for Consul services to get endpoints
	services, err := e.k8sClient.GetServices(ctx, namespace, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=consul",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get Consul services: %w", err)
	}

	for _, service := range services.Items {
		for _, port := range service.Spec.Ports {
			endpoint := fmt.Sprintf("%s.%s.svc.cluster.local:%d", service.Name, service.Namespace, port.Port)
			consulConfig.Endpoints = append(consulConfig.Endpoints, endpoint)
		}
	}

	return consulConfig, nil
}

func (e *Engine) discoverNginxConfig(ctx context.Context, namespace string) (*NginxConfig, error) {
	logrus.Infof("Discovering NGINX configuration in namespace: %s", namespace)

	nginxConfig := &NginxConfig{
		Workloads: []WorkloadInfo{},
		Upstreams: []string{},
		Routes:    []string{},
	}

	// Search for NGINX workloads using comprehensive discovery
	nginxSelectors := []string{
		"app.kubernetes.io/name=nginx",
		"app=nginx",
		"component=nginx",
		"app.kubernetes.io/name=nginx-ingress",
		"app=nginx-ingress",
	}

	workloads, err := e.DiscoverWorkloadsByLabels(ctx, namespace, nginxSelectors...)
	if err != nil {
		return nil, fmt.Errorf("failed to discover NGINX workloads: %w", err)
	}

	nginxConfig.Workloads = workloads

	// Set primary config from first workload found
	if len(workloads) > 0 {
		primary := workloads[0]
		nginxConfig.Replicas = primary.Replicas
		nginxConfig.Image = primary.Image
		nginxConfig.Version = primary.Version
	}

	// Look for NGINX configuration in ConfigMaps
	configMaps, err := e.k8sClient.GetConfigMaps(ctx, namespace, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ConfigMaps: %w", err)
	}

	for _, cm := range configMaps.Items {
		if isNginxConfigMap(cm.Name) {
			upstreams, routes := e.parseNginxConfig(cm.Data)
			nginxConfig.Upstreams = append(nginxConfig.Upstreams, upstreams...)
			nginxConfig.Routes = append(nginxConfig.Routes, routes...)
		}
	}

	return nginxConfig, nil
}

func (e *Engine) getTenantMimirLimits(ctx context.Context, tenantName string) (map[string]interface{}, error) {
	logrus.Infof("Getting Mimir limits for tenant: %s", tenantName)

	limits := make(map[string]interface{})

	// Get Mimir runtime overrides ConfigMap
	runtimeOverrides, err := e.k8sClient.GetConfigMap(ctx, e.config.Mimir.Namespace, "runtime-overrides", metav1.GetOptions{})
	if err != nil {
		logrus.Warnf("Failed to get runtime overrides: %v", err)
		// Try alternative names
		runtimeOverrides, err = e.k8sClient.GetConfigMap(ctx, e.config.Mimir.Namespace, "mimir-runtime-overrides", metav1.GetOptions{})
		if err != nil {
			return limits, nil // Return empty limits if not found
		}
	}

	// Parse the runtime overrides for tenant-specific limits
	if overridesData, exists := runtimeOverrides.Data["overrides.yaml"]; exists {
		tenantLimits := e.parseMimirOverrides(overridesData, tenantName)
		for key, value := range tenantLimits {
			limits[key] = value
		}
	}

	// Also check for tenant-specific ConfigMaps
	tenantConfigMapName := fmt.Sprintf("%s-limits", tenantName)
	tenantConfigMap, err := e.k8sClient.GetConfigMap(ctx, e.config.Mimir.Namespace, tenantConfigMapName, metav1.GetOptions{})
	if err == nil {
		for key, value := range tenantConfigMap.Data {
			limits[key] = value
		}
	}

	return limits, nil
}

func (e *Engine) countTenantComponents(ctx context.Context, namespace string) int {
	count := 0

	// Count deployments
	deployments, err := e.k8sClient.GetDeployments(ctx, namespace, metav1.ListOptions{})
	if err == nil {
		count += len(deployments.Items)
	}

	// Count statefulsets
	statefulSets, err := e.k8sClient.GetStatefulSets(ctx, namespace, metav1.ListOptions{})
	if err == nil {
		count += len(statefulSets.Items)
	}

	// Count services
	services, err := e.k8sClient.GetServices(ctx, namespace, metav1.ListOptions{})
	if err == nil {
		count += len(services.Items)
	}

	return count
}

func getDeploymentStatus(deployment *appsv1.Deployment) string {
	if deployment.Status.ReadyReplicas == deployment.Status.Replicas {
		return "running"
	}
	if deployment.Status.Replicas == 0 {
		return "stopped"
	}
	if deployment.Status.ReadyReplicas < deployment.Status.Replicas {
		return "degraded"
	}
	return "unknown"
}

// DiscoverWorkloadsByLabels discovers all types of workloads (Deployment, StatefulSet, DaemonSet) by label selectors
func (e *Engine) DiscoverWorkloadsByLabels(ctx context.Context, namespace string, labelSelectors ...string) ([]WorkloadInfo, error) {
	var allWorkloads []WorkloadInfo

	for _, labelSelector := range labelSelectors {
		// Search Deployments
		deployments, err := e.k8sClient.GetDeployments(ctx, namespace, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err == nil {
			for _, deployment := range deployments.Items {
				workload := WorkloadInfo{
					Name:        deployment.Name,
					Type:        "Deployment",
					Namespace:   deployment.Namespace,
					Labels:      deployment.Labels,
					Annotations: deployment.Annotations,
					Replicas:    *deployment.Spec.Replicas,
					Status:      getDeploymentStatus(&deployment),
				}
				if len(deployment.Spec.Template.Spec.Containers) > 0 {
					container := deployment.Spec.Template.Spec.Containers[0]
					workload.Image = container.Image
					workload.Version = extractVersion(container.Image)
				}
				allWorkloads = append(allWorkloads, workload)
			}
		}

		// Search StatefulSets
		statefulSets, err := e.k8sClient.GetStatefulSets(ctx, namespace, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err == nil {
			for _, statefulSet := range statefulSets.Items {
				workload := WorkloadInfo{
					Name:        statefulSet.Name,
					Type:        "StatefulSet",
					Namespace:   statefulSet.Namespace,
					Labels:      statefulSet.Labels,
					Annotations: statefulSet.Annotations,
					Replicas:    *statefulSet.Spec.Replicas,
					Status:      getStatefulSetStatus(&statefulSet),
				}
				if len(statefulSet.Spec.Template.Spec.Containers) > 0 {
					container := statefulSet.Spec.Template.Spec.Containers[0]
					workload.Image = container.Image
					workload.Version = extractVersion(container.Image)
				}
				allWorkloads = append(allWorkloads, workload)
			}
		}

		// Search DaemonSets
		daemonSets, err := e.k8sClient.GetDaemonSets(ctx, namespace, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err == nil {
			for _, daemonSet := range daemonSets.Items {
				workload := WorkloadInfo{
					Name:        daemonSet.Name,
					Type:        "DaemonSet",
					Namespace:   daemonSet.Namespace,
					Labels:      daemonSet.Labels,
					Annotations: daemonSet.Annotations,
					Replicas:    daemonSet.Status.DesiredNumberScheduled, // DaemonSets don't have replicas, use desired nodes
					Status:      getDaemonSetStatus(&daemonSet),
				}
				if len(daemonSet.Spec.Template.Spec.Containers) > 0 {
					container := daemonSet.Spec.Template.Spec.Containers[0]
					workload.Image = container.Image
					workload.Version = extractVersion(container.Image)
				}
				allWorkloads = append(allWorkloads, workload)
			}
		}
	}

	return allWorkloads, nil
}

// getStatefulSetStatus determines the status of a StatefulSet
func getStatefulSetStatus(statefulSet *appsv1.StatefulSet) string {
	if statefulSet.Status.ReadyReplicas == *statefulSet.Spec.Replicas {
		return "Running"
	} else if statefulSet.Status.ReadyReplicas > 0 {
		return "Degraded"
	}
	return "NotReady"
}

// getDaemonSetStatus determines the status of a DaemonSet
func getDaemonSetStatus(daemonSet *appsv1.DaemonSet) string {
	if daemonSet.Status.NumberReady == daemonSet.Status.DesiredNumberScheduled {
		return "Running"
	} else if daemonSet.Status.NumberReady > 0 {
		return "Degraded"
	}
	return "NotReady"
}

// isMimirComponentAdvanced checks if a Kubernetes resource (Deployment/StatefulSet) is a Mimir component
func (e *Engine) isMimirComponentAdvanced(name string, labels map[string]string, annotations map[string]string) bool {
	// Exclude our own components to avoid false positives
	if strings.Contains(strings.ToLower(name), "mimir-insights") ||
		strings.Contains(strings.ToLower(name), "mimirinsights") {
		logrus.Debugf("🔍 [DISCOVERY] Excluding own component: %s", name)
		return false
	}

	// Check if the name matches any compiled component pattern
	for componentType, patterns := range e.compiledPatterns.ComponentPatterns {
		for _, pattern := range patterns {
			if pattern.MatchString(name) {
				logrus.Debugf("🔍 [DISCOVERY] Component %s matched pattern for %s", name, componentType)
				return true
			}
		}
	}

	// Check if the name contains Mimir-specific keywords (but not our own components)
	if strings.Contains(strings.ToLower(name), "mimir") ||
		strings.Contains(strings.ToLower(name), "cortex") {
		logrus.Debugf("🔍 [DISCOVERY] Component %s matched Mimir/Cortex keyword", name)
		return true
	}

	// Check if the labels or annotations contain Mimir-specific keywords
	for key, value := range labels {
		if strings.Contains(strings.ToLower(key), "mimir") ||
			strings.Contains(strings.ToLower(value), "mimir") ||
			strings.Contains(strings.ToLower(key), "cortex") ||
			strings.Contains(strings.ToLower(value), "cortex") {
			logrus.Debugf("🔍 [DISCOVERY] Component %s matched Mimir/Cortex label: %s=%s", name, key, value)
			return true
		}
	}
	for key, value := range annotations {
		if strings.Contains(strings.ToLower(key), "mimir") ||
			strings.Contains(strings.ToLower(value), "mimir") ||
			strings.Contains(strings.ToLower(key), "cortex") ||
			strings.Contains(strings.ToLower(value), "cortex") {
			logrus.Debugf("🔍 [DISCOVERY] Component %s matched Mimir/Cortex annotation: %s=%s", name, key, value)
			return true
		}
	}

	logrus.Debugf("🔍 [DISCOVERY] Component %s did not match any Mimir patterns", name)
	return false
}

// isMimirServiceAdvanced checks if a Kubernetes service is a Mimir service
func (e *Engine) isMimirServiceAdvanced(name string, labels map[string]string, annotations map[string]string) bool {
	// Check if the name matches any compiled service pattern
	for _, pattern := range e.compiledPatterns.ServicePatterns {
		if pattern.MatchString(name) {
			return true
		}
	}

	// Check if the labels or annotations contain Mimir-specific keywords
	for key, value := range labels {
		if strings.Contains(strings.ToLower(key), "mimir") ||
			strings.Contains(strings.ToLower(value), "mimir") ||
			strings.Contains(strings.ToLower(key), "cortex") ||
			strings.Contains(strings.ToLower(value), "cortex") {
			return true
		}
	}
	for key, value := range annotations {
		if strings.Contains(strings.ToLower(key), "mimir") ||
			strings.Contains(strings.ToLower(value), "mimir") ||
			strings.Contains(strings.ToLower(key), "cortex") ||
			strings.Contains(strings.ToLower(value), "cortex") {
			return true
		}
	}

	return false
}

// isMimirConfigMapAdvanced checks if a Kubernetes ConfigMap is a Mimir ConfigMap
func (e *Engine) isMimirConfigMapAdvanced(name string, labels map[string]string, data map[string]string) bool {
	// Exclude our own ConfigMaps to avoid false positives
	if strings.Contains(strings.ToLower(name), "mimir-insights") ||
		strings.Contains(strings.ToLower(name), "mimirinsights") {
		logrus.Debugf("🔍 [DISCOVERY] Excluding own ConfigMap: %s", name)
		return false
	}

	// Check if the name matches any compiled ConfigMap pattern
	for _, pattern := range e.compiledPatterns.ConfigMapPatterns {
		if pattern.MatchString(name) {
			logrus.Debugf("🔍 [DISCOVERY] ConfigMap %s matched pattern", name)
			return true
		}
	}

	// Check if the name contains Mimir-specific keywords
	if strings.Contains(strings.ToLower(name), "mimir") ||
		strings.Contains(strings.ToLower(name), "cortex") {
		logrus.Debugf("🔍 [DISCOVERY] ConfigMap %s matched Mimir/Cortex keyword", name)
		return true
	}

	// Check if the labels or data contain Mimir-specific keywords
	for key, value := range labels {
		if strings.Contains(strings.ToLower(key), "mimir") ||
			strings.Contains(strings.ToLower(value), "mimir") ||
			strings.Contains(strings.ToLower(key), "cortex") ||
			strings.Contains(strings.ToLower(value), "cortex") {
			logrus.Debugf("🔍 [DISCOVERY] ConfigMap %s matched Mimir/Cortex label: %s=%s", name, key, value)
			return true
		}
	}
	for key, value := range data {
		if strings.Contains(strings.ToLower(key), "mimir") ||
			strings.Contains(strings.ToLower(value), "mimir") ||
			strings.Contains(strings.ToLower(key), "cortex") ||
			strings.Contains(strings.ToLower(value), "cortex") {
			logrus.Debugf("🔍 [DISCOVERY] ConfigMap %s matched Mimir/Cortex data: %s", name, key)
			return true
		}
	}

	logrus.Debugf("🔍 [DISCOVERY] ConfigMap %s did not match any Mimir patterns", name)
	return false
}

// isMetricsPath checks if a port name is a metrics path
func (e *Engine) isMetricsPath(portName string) bool {
	for _, pattern := range e.compiledPatterns.MetricsPathPatterns {
		if pattern.MatchString(portName) {
			return true
		}
	}
	// Check common metrics port names
	metricsPortNames := []string{"metrics", "http-metrics", "prometheus", "monitoring"}
	for _, name := range metricsPortNames {
		if strings.Contains(strings.ToLower(portName), name) {
			return true
		}
	}
	return false
}

// validateMimirComponentCrossReference performs cross-validation of Mimir components
func (e *Engine) validateMimirComponentCrossReference(ctx context.Context, component *MimirComponent, namespace string) ValidationResult {
	validation := ValidationResult{
		ConfidenceScore: 0,
		MatchedBy:       []string{},
		ValidationInfo:  make(map[string]interface{}),
	}

	// 1. Validate against known services
	services, err := e.k8sClient.GetServices(ctx, namespace, metav1.ListOptions{})
	if err == nil {
		for _, service := range services.Items {
			if e.isRelatedService(component.Name, service.Name) {
				validation.ConfidenceScore += 10
				validation.MatchedBy = append(validation.MatchedBy, "related_service")
				if validation.ValidationInfo["related_services"] == nil {
					validation.ValidationInfo["related_services"] = []string{}
				}
				validation.ValidationInfo["related_services"] = append(
					validation.ValidationInfo["related_services"].([]string), service.Name)
				break
			}
		}
	}

	// 2. Validate against ConfigMaps
	configMaps, err := e.k8sClient.GetConfigMaps(ctx, namespace, metav1.ListOptions{})
	if err == nil {
		for _, cm := range configMaps.Items {
			if e.isRelatedConfigMap(component.Name, cm.Name, cm.Data) {
				validation.ConfidenceScore += 8
				validation.MatchedBy = append(validation.MatchedBy, "related_configmap")
				if validation.ValidationInfo["related_configmaps"] == nil {
					validation.ValidationInfo["related_configmaps"] = []string{}
				}
				validation.ValidationInfo["related_configmaps"] = append(
					validation.ValidationInfo["related_configmaps"].([]string), cm.Name)
			}
		}
	}

	// 3. Validate metrics endpoints availability
	metricsAvailable := 0
	for _, endpoint := range component.MetricsEndpoints {
		if e.validateMetricsEndpoint(ctx, endpoint) {
			metricsAvailable++
		}
	}
	if metricsAvailable > 0 {
		validation.ConfidenceScore += float64(metricsAvailable * 5)
		validation.MatchedBy = append(validation.MatchedBy, "metrics_available")
		validation.ValidationInfo["available_metrics"] = metricsAvailable
	}

	// 4. Validate component type based on labels and annotations
	if e.validateComponentType(component) {
		validation.ConfidenceScore += 15
		validation.MatchedBy = append(validation.MatchedBy, "component_type_validated")
	}

	// 5. Check for expected ports
	if e.hasExpectedPorts(component) {
		validation.ConfidenceScore += 5
		validation.MatchedBy = append(validation.MatchedBy, "expected_ports")
	}

	return validation
}

// isRelatedService checks if a service is related to a component
func (e *Engine) isRelatedService(componentName, serviceName string) bool {
	// Direct name match
	if componentName == serviceName {
		return true
	}

	// Check if service name contains component name or vice versa
	if strings.Contains(serviceName, componentName) || strings.Contains(componentName, serviceName) {
		return true
	}

	// Check for common Mimir service naming patterns
	componentType := getComponentType(componentName)
	return strings.Contains(serviceName, componentType)
}

// isRelatedConfigMap checks if a ConfigMap is related to a component
func (e *Engine) isRelatedConfigMap(componentName, configMapName string, data map[string]string) bool {
	// Check if ConfigMap name relates to component
	if strings.Contains(configMapName, componentName) || strings.Contains(componentName, configMapName) {
		return true
	}

	// Check if ConfigMap data mentions the component
	for _, value := range data {
		if strings.Contains(strings.ToLower(value), strings.ToLower(componentName)) {
			return true
		}
	}

	// Check for component type mentions
	componentType := getComponentType(componentName)
	if strings.Contains(configMapName, componentType) {
		return true
	}

	return false
}

// validateMetricsEndpoint checks if a metrics endpoint is accessible
func (e *Engine) validateMetricsEndpoint(ctx context.Context, endpoint string) bool {
	// For now, we'll do a basic validation
	// In a real implementation, you might want to make HTTP requests to check availability

	// Check if endpoint follows expected patterns
	expectedPatterns := []string{
		"http://.*:9090",
		"http://.*:8080",
		"http://.*:3000",
		".*metrics.*",
	}

	for _, pattern := range expectedPatterns {
		matched, _ := regexp.MatchString(pattern, endpoint)
		if matched {
			return true
		}
	}

	return false
}

// validateComponentType validates the component type based on its characteristics
func (e *Engine) validateComponentType(component *MimirComponent) bool {
	componentType := component.Type
	name := strings.ToLower(component.Name)

	// Validate that the detected type matches the name
	switch componentType {
	case "distributor":
		return strings.Contains(name, "distributor") || strings.Contains(name, "dist")
	case "ingester":
		return strings.Contains(name, "ingester") || strings.Contains(name, "ingest")
	case "querier":
		return strings.Contains(name, "querier") || strings.Contains(name, "query") || strings.Contains(name, "frontend")
	case "compactor":
		return strings.Contains(name, "compactor") || strings.Contains(name, "compact")
	case "ruler":
		return strings.Contains(name, "ruler") || strings.Contains(name, "rule")
	case "alertmanager":
		return strings.Contains(name, "alertmanager") || strings.Contains(name, "alert")
	case "store_gateway":
		return strings.Contains(name, "store") || strings.Contains(name, "gateway")
	default:
		// If type is unknown but name contains mimir/cortex, it's likely valid
		return strings.Contains(name, "mimir") || strings.Contains(name, "cortex")
	}
}

// hasExpectedPorts checks if the component has expected ports for its type
func (e *Engine) hasExpectedPorts(component *MimirComponent) bool {
	// Common Mimir component ports
	expectedPorts := map[string][]int32{
		"distributor":   {9090, 8080, 3100},
		"ingester":      {9090, 8080, 3100},
		"querier":       {9090, 8080, 3100},
		"compactor":     {9090, 8080, 3100},
		"ruler":         {9090, 8080, 3100},
		"alertmanager":  {9093, 9094, 8080},
		"store_gateway": {9090, 8080, 3100},
	}

	expected, exists := expectedPorts[component.Type]
	if !exists {
		return true // If we don't have expected ports, assume it's valid
	}

	// Check if any service endpoints match expected ports
	for _, endpoint := range component.ServiceEndpoints {
		for _, expectedPort := range expected {
			portStr := fmt.Sprintf(":%d", expectedPort)
			if strings.Contains(endpoint, portStr) {
				return true
			}
		}
	}

	return false
}

// autoDiscoverMetricsEndpoints automatically discovers and validates metrics endpoints
func (e *Engine) autoDiscoverMetricsEndpoints(ctx context.Context, namespace string) ([]string, error) {
	logrus.Info("Auto-discovering metrics endpoints...")

	var metricsEndpoints []string

	// 1. Discover from services
	services, err := e.k8sClient.GetServices(ctx, namespace, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}

	for _, service := range services.Items {
		if e.isMimirServiceAdvanced(service.Name, service.Labels, service.Annotations) {
			for _, port := range service.Spec.Ports {
				// Check common metrics ports
				if e.isLikelyMetricsPort(port.Port, port.Name) {
					endpoint := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d",
						service.Name, service.Namespace, port.Port)
					metricsEndpoints = append(metricsEndpoints, endpoint)

					// Add specific metrics paths
					for _, path := range e.config.Mimir.API.MetricsPaths {
						if path != "/metrics" { // avoid duplicate
							fullEndpoint := endpoint + path
							metricsEndpoints = append(metricsEndpoints, fullEndpoint)
						}
					}
				}
			}
		}
	}

	// 2. Discover from ingresses
	ingresses, err := e.k8sClient.GetIngresses(ctx, namespace, metav1.ListOptions{})
	if err == nil {
		for _, ingress := range ingresses.Items {
			if e.isMimirIngress(ingress.Name, ingress.Labels, ingress.Annotations) {
				for _, rule := range ingress.Spec.Rules {
					if rule.HTTP != nil {
						for _, path := range rule.HTTP.Paths {
							if e.isMetricsPath(path.Path) {
								scheme := "https"
								if _, exists := ingress.Annotations["nginx.ingress.kubernetes.io/ssl-redirect"]; exists {
									scheme = "http"
								}
								endpoint := fmt.Sprintf("%s://%s%s", scheme, rule.Host, path.Path)
								metricsEndpoints = append(metricsEndpoints, endpoint)
							}
						}
					}
				}
			}
		}
	}

	logrus.Infof("Auto-discovered %d metrics endpoints", len(metricsEndpoints))
	return metricsEndpoints, nil
}

// isLikelyMetricsPort checks if a port is likely to serve metrics
func (e *Engine) isLikelyMetricsPort(port int32, name string) bool {
	// Common metrics ports
	commonMetricsPorts := []int32{9090, 8080, 3000, 9093, 9094, 3100}
	for _, commonPort := range commonMetricsPorts {
		if port == commonPort {
			return true
		}
	}

	// Check port name
	if name != "" {
		metricsNames := []string{"metrics", "http", "web", "prometheus", "monitoring"}
		for _, metricsName := range metricsNames {
			if strings.Contains(strings.ToLower(name), metricsName) {
				return true
			}
		}
	}

	return false
}

// isMimirIngress checks if an ingress is related to Mimir
func (e *Engine) isMimirIngress(name string, labels, annotations map[string]string) bool {
	// Check name
	if strings.Contains(strings.ToLower(name), "mimir") ||
		strings.Contains(strings.ToLower(name), "cortex") {
		return true
	}

	// Check labels and annotations
	for key, value := range labels {
		if strings.Contains(strings.ToLower(key), "mimir") ||
			strings.Contains(strings.ToLower(value), "mimir") ||
			strings.Contains(strings.ToLower(key), "cortex") ||
			strings.Contains(strings.ToLower(value), "cortex") {
			return true
		}
	}

	for key, value := range annotations {
		if strings.Contains(strings.ToLower(key), "mimir") ||
			strings.Contains(strings.ToLower(value), "mimir") ||
			strings.Contains(strings.ToLower(key), "cortex") ||
			strings.Contains(strings.ToLower(value), "cortex") {
			return true
		}
	}

	return false
}

// GetAutoDiscoveredMetrics returns auto-discovered metrics endpoints for the Mimir namespace
func (e *Engine) GetAutoDiscoveredMetrics(ctx context.Context) ([]string, error) {
	// Auto-discover namespace if needed
	namespace := e.config.Mimir.Namespace
	if namespace == "" || namespace == "auto" {
		autoNS, err := e.AutoDiscoverMimirNamespace(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to auto-discover namespace: %w", err)
		}
		namespace = autoNS
	}

	return e.autoDiscoverMetricsEndpoints(ctx, namespace)
}

// DiscoverTenantsComprehensive performs comprehensive tenant discovery using multiple strategies
func (e *Engine) DiscoverTenantsComprehensive(ctx context.Context) (*ComprehensiveTenantDiscoveryResult, error) {
	logrus.Info("🔍 Starting comprehensive tenant discovery using multiple strategies")

	if e.multiStrategyDiscovery == nil {
		return nil, fmt.Errorf("multi-strategy discovery not initialized")
	}

	return e.multiStrategyDiscovery.DiscoverTenantsComprehensive(ctx)
}

// DiscoverMimirComprehensive performs comprehensive Mimir discovery using multiple strategies
func (e *Engine) DiscoverMimirComprehensive(ctx context.Context) (*ComprehensiveMimirDiscoveryResult, error) {
	logrus.Info("🔍 Starting comprehensive Mimir discovery using multiple strategies")

	if e.multiStrategyMimirDiscovery == nil {
		return nil, fmt.Errorf("multi-strategy Mimir discovery not initialized")
	}

	return e.multiStrategyMimirDiscovery.DiscoverMimirComprehensive(ctx)
}
