package discovery

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/akshaydubey29/mimirInsights/pkg/k8s"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnvironmentInfo represents the detected environment information
type EnvironmentInfo struct {
	IsProduction       bool                   `json:"is_production"`
	ClusterName        string                 `json:"cluster_name"`
	ClusterVersion     string                 `json:"cluster_version"`
	MimirNamespace     string                 `json:"mimir_namespace"`
	TotalNamespaces    int                    `json:"total_namespaces"`
	TotalNodes         int                    `json:"total_nodes"`
	DataSource         string                 `json:"data_source"` // "production", "mock", "mixed"
	DetectedTenants    []DetectedTenant       `json:"detected_tenants"`
	MimirComponents    []string               `json:"mimir_components"`
	LastUpdated        time.Time              `json:"last_updated"`
	EnvironmentDetails map[string]interface{} `json:"environment_details"`
}

// DetectedTenant represents an auto-discovered tenant
type DetectedTenant struct {
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	Source          string            `json:"source"` // "alloy", "grafana-agent", "label", "configmap"
	OrgID           string            `json:"org_id"`
	HasRealData     bool              `json:"has_real_data"`
	LastSeen        time.Time         `json:"last_seen"`
	MetricsVolume   int64             `json:"metrics_volume"`
	ComponentStatus map[string]string `json:"component_status"`
}

// EnvironmentDetector handles environment detection and analysis
type EnvironmentDetector struct {
	k8sClient *k8s.Client
}

// NewEnvironmentDetector creates a new environment detector
func NewEnvironmentDetector(k8sClient *k8s.Client) *EnvironmentDetector {
	return &EnvironmentDetector{
		k8sClient: k8sClient,
	}
}

// DetectEnvironment performs comprehensive environment detection
func (ed *EnvironmentDetector) DetectEnvironment(ctx context.Context, mimirNamespace string) (*EnvironmentInfo, error) {
	logrus.Info("Starting comprehensive environment detection")

	env := &EnvironmentInfo{
		LastUpdated:        time.Now(),
		MimirNamespace:     mimirNamespace,
		DetectedTenants:    []DetectedTenant{},
		MimirComponents:    []string{},
		EnvironmentDetails: make(map[string]interface{}),
	}

	// Detect cluster information
	if err := ed.detectClusterInfo(ctx, env); err != nil {
		logrus.Warnf("Failed to detect cluster info: %v", err)
	}

	// Detect Mimir components
	if err := ed.detectMimirComponents(ctx, env); err != nil {
		logrus.Warnf("Failed to detect Mimir components: %v", err)
	}

	// Auto-discover tenants from all sources
	if err := ed.autoDiscoverTenants(ctx, env); err != nil {
		logrus.Warnf("Failed to auto-discover tenants: %v", err)
	}

	// Determine if this is a production environment
	ed.determineProductionStatus(env)

	// Analyze data sources
	ed.analyzeDataSources(env)

	logrus.Infof("Environment detection completed: %s environment with %d tenants",
		env.DataSource, len(env.DetectedTenants))

	return env, nil
}

// detectClusterInfo detects basic cluster information
func (ed *EnvironmentDetector) detectClusterInfo(ctx context.Context, env *EnvironmentInfo) error {
	// Test cluster connection
	err := ed.k8sClient.TestConnection(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}

	// Get cluster info
	clusterInfo, err := ed.k8sClient.GetClusterInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get cluster info: %w", err)
	}

	if versionInfo, ok := clusterInfo["version"]; ok {
		env.ClusterVersion = fmt.Sprintf("%v", versionInfo)
	}

	if nodeCount, ok := clusterInfo["nodeCount"]; ok {
		env.TotalNodes = nodeCount.(int)
	}

	if namespaceCount, ok := clusterInfo["namespaceCount"]; ok {
		env.TotalNamespaces = namespaceCount.(int)
	}

	// Try to detect cluster name from various sources
	env.ClusterName = ed.detectClusterName(ctx)

	return nil
}

// detectClusterName attempts to detect the cluster name from various sources
func (ed *EnvironmentDetector) detectClusterName(ctx context.Context) string {
	// Try to get cluster name from kube-system namespace annotations
	kubeSystemNS, err := ed.k8sClient.GetNamespace(ctx, "kube-system", metav1.GetOptions{})
	if err == nil {
		if clusterName, exists := kubeSystemNS.Annotations["cluster-name"]; exists {
			return clusterName
		}
	}

	// Try to get from cluster-info ConfigMap
	clusterInfo, err := ed.k8sClient.GetConfigMap(ctx, "kube-public", "cluster-info", metav1.GetOptions{})
	if err == nil {
		if kubeconfig, exists := clusterInfo.Data["kubeconfig"]; exists {
			if name := ed.extractClusterNameFromKubeconfig(kubeconfig); name != "" {
				return name
			}
		}
	}

	// Default fallback
	return "unknown-cluster"
}

// extractClusterNameFromKubeconfig extracts cluster name from kubeconfig
func (ed *EnvironmentDetector) extractClusterNameFromKubeconfig(kubeconfig string) string {
	lines := strings.Split(kubeconfig, "\n")
	for _, line := range lines {
		if strings.Contains(line, "name:") && strings.Contains(line, "cluster") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// detectMimirComponents detects running Mimir components
func (ed *EnvironmentDetector) detectMimirComponents(ctx context.Context, env *EnvironmentInfo) error {
	// Get all deployments in Mimir namespace
	deployments, err := ed.k8sClient.GetDeployments(ctx, env.MimirNamespace, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get Mimir deployments: %w", err)
	}

	for _, deployment := range deployments.Items {
		if isMimirComponent(deployment.Labels) {
			componentType := getComponentType(deployment.Name)
			env.MimirComponents = append(env.MimirComponents, componentType)
		}
	}

	// Get all statefulsets in Mimir namespace
	statefulSets, err := ed.k8sClient.GetStatefulSets(ctx, env.MimirNamespace, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get Mimir statefulsets: %w", err)
	}

	for _, statefulSet := range statefulSets.Items {
		if isMimirComponent(statefulSet.Labels) {
			componentType := getComponentType(statefulSet.Name)
			env.MimirComponents = append(env.MimirComponents, componentType)
		}
	}

	return nil
}

// autoDiscoverTenants discovers tenants from all possible sources
func (ed *EnvironmentDetector) autoDiscoverTenants(ctx context.Context, env *EnvironmentInfo) error {
	// Get all namespaces
	namespaces, err := ed.k8sClient.GetNamespaces(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get namespaces: %w", err)
	}

	for _, namespace := range namespaces.Items {
		// Skip system namespaces
		if ed.isSystemNamespace(namespace.Name) {
			continue
		}

		tenant := ed.analyzeTenantNamespace(ctx, &namespace)
		if tenant != nil {
			env.DetectedTenants = append(env.DetectedTenants, *tenant)
		}
	}

	return nil
}

// isSystemNamespace checks if a namespace is a system namespace
func (ed *EnvironmentDetector) isSystemNamespace(name string) bool {
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

// analyzeTenantNamespace analyzes a namespace to determine if it's a tenant namespace
func (ed *EnvironmentDetector) analyzeTenantNamespace(ctx context.Context, namespace *corev1.Namespace) *DetectedTenant {
	tenant := &DetectedTenant{
		Name:            namespace.Name,
		Namespace:       namespace.Name,
		LastSeen:        time.Now(),
		ComponentStatus: make(map[string]string),
	}

	// Check for tenant indicators in labels
	for key, value := range namespace.Labels {
		if strings.Contains(strings.ToLower(key), "tenant") ||
			strings.Contains(strings.ToLower(key), "team") ||
			strings.Contains(strings.ToLower(key), "org") {
			tenant.Source = "label"
			tenant.OrgID = value
			break
		}
	}

	// Look for Alloy/Grafana Agent configurations
	if orgID := ed.findTenantInAlloyConfig(ctx, namespace.Name); orgID != "" {
		tenant.Source = "alloy"
		tenant.OrgID = orgID
	}

	// Analyze component status
	ed.analyzeComponentStatus(ctx, tenant)

	// Determine if this has real data
	tenant.HasRealData = ed.hasRealData(ctx, tenant)

	// Only return tenant if we found evidence it's actually a tenant
	if tenant.Source != "" || tenant.OrgID != "" || ed.hasMonitoringComponents(ctx, namespace.Name) {
		return tenant
	}

	return nil
}

// findTenantInAlloyConfig looks for tenant information in Alloy configurations
func (ed *EnvironmentDetector) findTenantInAlloyConfig(ctx context.Context, namespace string) string {
	configMaps, err := ed.k8sClient.GetConfigMaps(ctx, namespace, metav1.ListOptions{})
	if err != nil {
		return ""
	}

	for _, cm := range configMaps.Items {
		if isAlloyConfigMap(cm.Name) {
			for _, data := range cm.Data {
				if orgID := ed.extractOrgIDFromConfig(data); orgID != "" {
					return orgID
				}
			}
		}
	}

	return ""
}

// extractOrgIDFromConfig extracts org ID from configuration content
func (ed *EnvironmentDetector) extractOrgIDFromConfig(config string) string {
	lines := strings.Split(config, "\n")
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

// analyzeComponentStatus analyzes the status of components in a tenant namespace
func (ed *EnvironmentDetector) analyzeComponentStatus(ctx context.Context, tenant *DetectedTenant) {
	// Check for Alloy
	if deployments, err := ed.k8sClient.GetDeployments(ctx, tenant.Namespace, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=alloy",
	}); err == nil && len(deployments.Items) > 0 {
		tenant.ComponentStatus["alloy"] = getDeploymentStatus(&deployments.Items[0])
	}

	// Check for Grafana Agent
	if deployments, err := ed.k8sClient.GetDeployments(ctx, tenant.Namespace, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=grafana-agent",
	}); err == nil && len(deployments.Items) > 0 {
		tenant.ComponentStatus["grafana-agent"] = getDeploymentStatus(&deployments.Items[0])
	}

	// Check for NGINX
	if deployments, err := ed.k8sClient.GetDeployments(ctx, tenant.Namespace, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=nginx",
	}); err == nil && len(deployments.Items) > 0 {
		tenant.ComponentStatus["nginx"] = getDeploymentStatus(&deployments.Items[0])
	}

	// Check for Consul
	if deployments, err := ed.k8sClient.GetDeployments(ctx, tenant.Namespace, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=consul",
	}); err == nil && len(deployments.Items) > 0 {
		tenant.ComponentStatus["consul"] = getDeploymentStatus(&deployments.Items[0])
	}
}

// hasMonitoringComponents checks if namespace has monitoring components
func (ed *EnvironmentDetector) hasMonitoringComponents(ctx context.Context, namespace string) bool {
	deployments, err := ed.k8sClient.GetDeployments(ctx, namespace, metav1.ListOptions{})
	if err != nil {
		return false
	}

	monitoringComponents := []string{"alloy", "grafana-agent", "prometheus", "nginx", "consul"}

	for _, deployment := range deployments.Items {
		for _, component := range monitoringComponents {
			if strings.Contains(strings.ToLower(deployment.Name), component) {
				return true
			}
		}
	}

	return false
}

// hasRealData determines if a tenant namespace has real production data
func (ed *EnvironmentDetector) hasRealData(ctx context.Context, tenant *DetectedTenant) bool {
	// Check for active pods with recent activity
	pods, err := ed.k8sClient.GetPods(ctx, tenant.Namespace, metav1.ListOptions{})
	if err != nil {
		return false
	}

	activePods := 0
	for _, pod := range pods.Items {
		if pod.Status.Phase == "Running" {
			activePods++
		}
	}

	// If there are multiple active pods, likely real data
	if activePods > 2 {
		return true
	}

	// Check for persistent volumes (indicates data storage)
	pvcs, err := ed.k8sClient.GetPersistentVolumeClaims(ctx, tenant.Namespace, metav1.ListOptions{})
	if err == nil && len(pvcs.Items) > 0 {
		return true
	}

	return false
}

// determineProductionStatus determines if this is a production environment
func (ed *EnvironmentDetector) determineProductionStatus(env *EnvironmentInfo) {
	productionIndicators := 0

	// Check cluster size
	if env.TotalNodes >= 3 {
		productionIndicators++
	}

	// Check number of namespaces
	if env.TotalNamespaces >= 10 {
		productionIndicators++
	}

	// Check for production Mimir components
	if len(env.MimirComponents) >= 4 {
		productionIndicators++
	}

	// Check for real tenant data
	realDataTenants := 0
	for _, tenant := range env.DetectedTenants {
		if tenant.HasRealData {
			realDataTenants++
		}
	}

	if realDataTenants >= 2 {
		productionIndicators++
	}

	// Determine production status
	env.IsProduction = productionIndicators >= 3

	// Store details
	env.EnvironmentDetails["production_indicators"] = productionIndicators
	env.EnvironmentDetails["real_data_tenants"] = realDataTenants
}

// analyzeDataSources analyzes the sources of data in the environment
func (ed *EnvironmentDetector) analyzeDataSources(env *EnvironmentInfo) {
	realDataCount := 0
	mockDataCount := 0

	for _, tenant := range env.DetectedTenants {
		if tenant.HasRealData {
			realDataCount++
		} else {
			mockDataCount++
		}
	}

	if realDataCount > 0 && mockDataCount == 0 {
		env.DataSource = "production"
	} else if realDataCount == 0 && mockDataCount > 0 {
		env.DataSource = "mock"
	} else if realDataCount > 0 && mockDataCount > 0 {
		env.DataSource = "mixed"
	} else {
		env.DataSource = "unknown"
	}

	env.EnvironmentDetails["real_data_count"] = realDataCount
	env.EnvironmentDetails["mock_data_count"] = mockDataCount
}
