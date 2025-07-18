package tuning

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/akshaydubey29/mimirInsights/pkg/k8s"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// AlloyTuner handles Alloy replica tuning operations
type AlloyTuner struct {
	k8sClient *k8s.Client
}

// AlloyReplicaRequest represents a request to adjust Alloy replicas
type AlloyReplicaRequest struct {
	Namespace      string `json:"namespace" binding:"required"`
	DeploymentName string `json:"deployment_name" binding:"required"`
	Replicas       int32  `json:"replicas" binding:"required,min=1,max=10"`
	Reason         string `json:"reason"`
}

// AlloyReplicaResponse represents the response after replica adjustment
type AlloyReplicaResponse struct {
	Namespace      string    `json:"namespace"`
	DeploymentName string    `json:"deployment_name"`
	OldReplicas    int32     `json:"old_replicas"`
	NewReplicas    int32     `json:"new_replicas"`
	Status         string    `json:"status"`
	Message        string    `json:"message"`
	UpdatedAt      time.Time `json:"updated_at"`
	Recommendation string    `json:"recommendation"`
}

// AlloyScalingRecommendation represents scaling recommendations
type AlloyScalingRecommendation struct {
	Namespace           string       `json:"namespace"`
	DeploymentName      string       `json:"deployment_name"`
	CurrentReplicas     int32        `json:"current_replicas"`
	RecommendedReplicas int32        `json:"recommended_replicas"`
	Reason              string       `json:"reason"`
	ConfidenceScore     float64      `json:"confidence_score"`
	MetricsData         AlloyMetrics `json:"metrics_data"`
	Priority            string       `json:"priority"` // "low", "medium", "high", "critical"
}

// AlloyMetrics represents metrics used for scaling decisions
type AlloyMetrics struct {
	CPUUtilization     float64 `json:"cpu_utilization"`
	MemoryUtilization  float64 `json:"memory_utilization"`
	ScrapeTargets      int     `json:"scrape_targets"`
	ScrapeFailures     int     `json:"scrape_failures"`
	QueueSize          int     `json:"queue_size"`
	LastScrapeDuration float64 `json:"last_scrape_duration"`
}

// NewAlloyTuner creates a new Alloy tuner
func NewAlloyTuner(k8sClient *k8s.Client) *AlloyTuner {
	return &AlloyTuner{
		k8sClient: k8sClient,
	}
}

// ScaleAlloyReplicas scales Alloy deployment replicas
func (a *AlloyTuner) ScaleAlloyReplicas(ctx context.Context, req AlloyReplicaRequest) (*AlloyReplicaResponse, error) {
	logrus.Infof("Scaling Alloy deployment %s/%s to %d replicas", req.Namespace, req.DeploymentName, req.Replicas)

	// Get current deployment
	deployment, err := a.k8sClient.GetDeployment(ctx, req.Namespace, req.DeploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	// Validate that this is an Alloy deployment
	if !a.isAlloyDeployment(deployment) {
		return nil, fmt.Errorf("deployment %s does not appear to be an Alloy deployment", req.DeploymentName)
	}

	oldReplicas := *deployment.Spec.Replicas

	// Validate scaling request
	if err := a.validateScalingRequest(req, oldReplicas); err != nil {
		return nil, fmt.Errorf("scaling validation failed: %w", err)
	}

	// Create patch for replica count
	patch := []byte(fmt.Sprintf(`{"spec":{"replicas":%d}}`, req.Replicas))

	// Apply patch
	updatedDeployment, err := a.k8sClient.PatchDeployment(
		ctx,
		req.Namespace,
		req.DeploymentName,
		types.MergePatchType,
		patch,
		metav1.PatchOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to patch deployment: %w", err)
	}

	// Generate recommendation for future
	recommendation := a.generateScalingRecommendation(req.Replicas, oldReplicas)

	response := &AlloyReplicaResponse{
		Namespace:      req.Namespace,
		DeploymentName: req.DeploymentName,
		OldReplicas:    oldReplicas,
		NewReplicas:    *updatedDeployment.Spec.Replicas,
		Status:         "success",
		Message:        fmt.Sprintf("Successfully scaled from %d to %d replicas", oldReplicas, req.Replicas),
		UpdatedAt:      time.Now(),
		Recommendation: recommendation,
	}

	logrus.Infof("Successfully scaled Alloy deployment %s/%s from %d to %d replicas",
		req.Namespace, req.DeploymentName, oldReplicas, req.Replicas)

	return response, nil
}

// GetAlloyScalingRecommendations gets scaling recommendations for all Alloy deployments
func (a *AlloyTuner) GetAlloyScalingRecommendations(ctx context.Context, namespaces []string) ([]AlloyScalingRecommendation, error) {
	var recommendations []AlloyScalingRecommendation

	for _, namespace := range namespaces {
		deployments, err := a.k8sClient.GetDeployments(ctx, namespace, metav1.ListOptions{})
		if err != nil {
			logrus.Warnf("Failed to get deployments in namespace %s: %v", namespace, err)
			continue
		}

		for _, deployment := range deployments.Items {
			if a.isAlloyDeployment(&deployment) {
				recommendation := a.analyzeAlloyDeployment(ctx, &deployment)
				if recommendation != nil {
					recommendations = append(recommendations, *recommendation)
				}
			}
		}
	}

	return recommendations, nil
}

// GetAlloyWorkloads gets all Alloy workloads (Deployments, StatefulSets, DaemonSets) across namespaces
func (a *AlloyTuner) GetAlloyWorkloads(ctx context.Context, namespaces []string) ([]map[string]interface{}, error) {
	var alloyWorkloads []map[string]interface{}

	for _, namespace := range namespaces {
		// Search Deployments
		deployments, err := a.k8sClient.GetDeployments(ctx, namespace, metav1.ListOptions{})
		if err == nil {
			for _, deployment := range deployments.Items {
				if a.isAlloyDeployment(&deployment) {
					workloadInfo := map[string]interface{}{
						"namespace":       deployment.Namespace,
						"name":            deployment.Name,
						"type":            "Deployment",
						"replicas":        *deployment.Spec.Replicas,
						"ready_replicas":  deployment.Status.ReadyReplicas,
						"image":           a.getContainerImage(&deployment),
						"labels":          deployment.Labels,
						"created_at":      deployment.CreationTimestamp,
						"resource_limits": a.getResourceLimits(&deployment),
					}
					alloyWorkloads = append(alloyWorkloads, workloadInfo)
				}
			}
		}

		// Search StatefulSets
		statefulSets, err := a.k8sClient.GetStatefulSets(ctx, namespace, metav1.ListOptions{})
		if err == nil {
			for _, statefulSet := range statefulSets.Items {
				if a.isAlloyStatefulSet(&statefulSet) {
					workloadInfo := map[string]interface{}{
						"namespace":       statefulSet.Namespace,
						"name":            statefulSet.Name,
						"type":            "StatefulSet",
						"replicas":        *statefulSet.Spec.Replicas,
						"ready_replicas":  statefulSet.Status.ReadyReplicas,
						"image":           a.getContainerImageFromStatefulSet(&statefulSet),
						"labels":          statefulSet.Labels,
						"created_at":      statefulSet.CreationTimestamp,
						"resource_limits": a.getResourceLimitsFromStatefulSet(&statefulSet),
					}
					alloyWorkloads = append(alloyWorkloads, workloadInfo)
				}
			}
		}

		// Search DaemonSets
		daemonSets, err := a.k8sClient.GetDaemonSets(ctx, namespace, metav1.ListOptions{})
		if err == nil {
			for _, daemonSet := range daemonSets.Items {
				if a.isAlloyDaemonSet(&daemonSet) {
					workloadInfo := map[string]interface{}{
						"namespace":         daemonSet.Namespace,
						"name":              daemonSet.Name,
						"type":              "DaemonSet",
						"desired_scheduled": daemonSet.Status.DesiredNumberScheduled,
						"ready_replicas":    daemonSet.Status.NumberReady,
						"image":             a.getContainerImageFromDaemonSet(&daemonSet),
						"labels":            daemonSet.Labels,
						"created_at":        daemonSet.CreationTimestamp,
						"resource_limits":   a.getResourceLimitsFromDaemonSet(&daemonSet),
					}
					alloyWorkloads = append(alloyWorkloads, workloadInfo)
				}
			}
		}
	}

	return alloyWorkloads, nil
}

// GetAlloyDeployments gets all Alloy deployments across namespaces (backwards compatibility)
func (a *AlloyTuner) GetAlloyDeployments(ctx context.Context, namespaces []string) ([]map[string]interface{}, error) {
	var alloyDeployments []map[string]interface{}

	for _, namespace := range namespaces {
		deployments, err := a.k8sClient.GetDeployments(ctx, namespace, metav1.ListOptions{})
		if err != nil {
			logrus.Warnf("Failed to get deployments in namespace %s: %v", namespace, err)
			continue
		}

		for _, deployment := range deployments.Items {
			if a.isAlloyDeployment(&deployment) {
				alloyInfo := map[string]interface{}{
					"namespace":       deployment.Namespace,
					"name":            deployment.Name,
					"type":            "Deployment",
					"replicas":        *deployment.Spec.Replicas,
					"ready_replicas":  deployment.Status.ReadyReplicas,
					"image":           a.getContainerImage(&deployment),
					"labels":          deployment.Labels,
					"created_at":      deployment.CreationTimestamp,
					"resource_limits": a.getResourceLimits(&deployment),
				}
				alloyDeployments = append(alloyDeployments, alloyInfo)
			}
		}
	}

	return alloyDeployments, nil
}

// isAlloyDeployment checks if a deployment is an Alloy deployment
func (a *AlloyTuner) isAlloyDeployment(deployment *appsv1.Deployment) bool {
	// Check deployment name patterns
	alloyPatterns := []string{"alloy", "grafana-alloy", "agent"}
	deploymentName := deployment.Name

	for _, pattern := range alloyPatterns {
		if len(deploymentName) >= len(pattern) {
			for i := 0; i <= len(deploymentName)-len(pattern); i++ {
				if deploymentName[i:i+len(pattern)] == pattern {
					return true
				}
			}
		}
	}

	// Check labels
	if deployment.Labels != nil {
		if app, exists := deployment.Labels["app"]; exists {
			if app == "alloy" || app == "grafana-alloy" {
				return true
			}
		}
		if component, exists := deployment.Labels["app.kubernetes.io/name"]; exists {
			if component == "alloy" || component == "grafana-alloy" {
				return true
			}
		}
	}

	// Check container images
	for _, container := range deployment.Spec.Template.Spec.Containers {
		if len(container.Image) > 0 {
			image := container.Image
			if len(image) >= 5 && image[len(image)-5:] == "alloy" {
				return true
			}
			if len(image) >= 12 && image[:12] == "grafana/alloy" {
				return true
			}
		}
	}

	return false
}

// isAlloyStatefulSet checks if a StatefulSet is an Alloy StatefulSet
func (a *AlloyTuner) isAlloyStatefulSet(statefulSet *appsv1.StatefulSet) bool {
	// Check labels
	if name, exists := statefulSet.Labels["app.kubernetes.io/name"]; exists {
		if name == "alloy" || name == "grafana-agent" {
			return true
		}
	}

	if app, exists := statefulSet.Labels["app"]; exists {
		if app == "alloy" || app == "grafana-agent" {
			return true
		}
	}

	if component, exists := statefulSet.Labels["component"]; exists {
		if component == "alloy" || component == "grafana-agent" {
			return true
		}
	}

	// Check StatefulSet name
	name := strings.ToLower(statefulSet.Name)
	return strings.Contains(name, "alloy") || strings.Contains(name, "grafana-agent")
}

// isAlloyDaemonSet checks if a DaemonSet is an Alloy DaemonSet
func (a *AlloyTuner) isAlloyDaemonSet(daemonSet *appsv1.DaemonSet) bool {
	// Check labels
	if name, exists := daemonSet.Labels["app.kubernetes.io/name"]; exists {
		if name == "alloy" || name == "grafana-agent" {
			return true
		}
	}

	if app, exists := daemonSet.Labels["app"]; exists {
		if app == "alloy" || app == "grafana-agent" {
			return true
		}
	}

	if component, exists := daemonSet.Labels["component"]; exists {
		if component == "alloy" || component == "grafana-agent" {
			return true
		}
	}

	// Check DaemonSet name
	name := strings.ToLower(daemonSet.Name)
	return strings.Contains(name, "alloy") || strings.Contains(name, "grafana-agent")
}

// validateScalingRequest validates the scaling request
func (a *AlloyTuner) validateScalingRequest(req AlloyReplicaRequest, currentReplicas int32) error {
	// Check replica bounds
	if req.Replicas < 1 {
		return fmt.Errorf("replica count must be at least 1")
	}
	if req.Replicas > 10 {
		return fmt.Errorf("replica count cannot exceed 10 for safety")
	}

	// Check for dramatic scaling changes
	scalingFactor := float64(req.Replicas) / float64(currentReplicas)
	if scalingFactor > 3.0 {
		return fmt.Errorf("scaling up by more than 3x requires manual approval (current: %d, requested: %d)",
			currentReplicas, req.Replicas)
	}

	// Check if scaling down to zero (not allowed)
	if req.Replicas == 0 {
		return fmt.Errorf("scaling to zero replicas is not allowed")
	}

	return nil
}

// analyzeAlloyDeployment analyzes an Alloy deployment for scaling recommendations
func (a *AlloyTuner) analyzeAlloyDeployment(ctx context.Context, deployment *appsv1.Deployment) *AlloyScalingRecommendation {
	currentReplicas := *deployment.Spec.Replicas

	// Get metrics for this deployment (simulated for now)
	metrics := a.getDeploymentMetrics(ctx, deployment)

	// Calculate recommended replicas based on metrics
	recommendedReplicas, reason, confidence := a.calculateRecommendedReplicas(metrics, currentReplicas)

	if recommendedReplicas == currentReplicas {
		return nil // No scaling needed
	}

	priority := a.calculatePriority(metrics, currentReplicas, recommendedReplicas)

	return &AlloyScalingRecommendation{
		Namespace:           deployment.Namespace,
		DeploymentName:      deployment.Name,
		CurrentReplicas:     currentReplicas,
		RecommendedReplicas: recommendedReplicas,
		Reason:              reason,
		ConfidenceScore:     confidence,
		MetricsData:         metrics,
		Priority:            priority,
	}
}

// getDeploymentMetrics gets metrics for a deployment (simulated)
func (a *AlloyTuner) getDeploymentMetrics(ctx context.Context, deployment *appsv1.Deployment) AlloyMetrics {
	// In a real implementation, this would query Prometheus metrics
	// For now, return simulated metrics based on replica count
	replicas := *deployment.Spec.Replicas

	// Simulate higher utilization with fewer replicas
	cpuUtil := 50.0 + float64(10.0/float64(replicas))
	memUtil := 40.0 + float64(15.0/float64(replicas))

	return AlloyMetrics{
		CPUUtilization:     cpuUtil,
		MemoryUtilization:  memUtil,
		ScrapeTargets:      int(replicas * 50), // 50 targets per replica
		ScrapeFailures:     int(replicas * 2),  // 2 failures per replica
		QueueSize:          int(replicas * 10), // 10 queued items per replica
		LastScrapeDuration: 2.5,
	}
}

// calculateRecommendedReplicas calculates recommended replica count
func (a *AlloyTuner) calculateRecommendedReplicas(metrics AlloyMetrics, current int32) (int32, string, float64) {
	// High CPU utilization - scale up
	if metrics.CPUUtilization > 80 {
		return current + 1, "High CPU utilization detected", 0.9
	}

	// High memory utilization - scale up
	if metrics.MemoryUtilization > 85 {
		return current + 1, "High memory utilization detected", 0.85
	}

	// Many scrape failures - scale up
	failureRate := float64(metrics.ScrapeFailures) / float64(metrics.ScrapeTargets) * 100
	if failureRate > 5 {
		return current + 1, "High scrape failure rate detected", 0.8
	}

	// Low utilization - scale down
	if metrics.CPUUtilization < 30 && metrics.MemoryUtilization < 40 && current > 1 {
		return current - 1, "Low resource utilization detected", 0.7
	}

	// Large queue size - scale up
	if metrics.QueueSize > int(current)*20 {
		return current + 1, "Large metrics queue detected", 0.75
	}

	return current, "No scaling needed", 1.0
}

// calculatePriority calculates the priority of the scaling recommendation
func (a *AlloyTuner) calculatePriority(metrics AlloyMetrics, current, recommended int32) string {
	if metrics.CPUUtilization > 90 || metrics.MemoryUtilization > 95 {
		return "critical"
	}
	if metrics.CPUUtilization > 80 || metrics.MemoryUtilization > 85 {
		return "high"
	}
	if abs(recommended-current) > 1 {
		return "medium"
	}
	return "low"
}

// generateScalingRecommendation generates a recommendation message
func (a *AlloyTuner) generateScalingRecommendation(newReplicas, oldReplicas int32) string {
	if newReplicas > oldReplicas {
		return fmt.Sprintf("Monitor CPU and memory usage. Consider HPA if scaling frequently.")
	} else if newReplicas < oldReplicas {
		return fmt.Sprintf("Monitor for increased latency or scrape failures after scale-down.")
	}
	return "Monitor deployment performance after scaling."
}

// getContainerImage gets the primary container image
func (a *AlloyTuner) getContainerImage(deployment *appsv1.Deployment) string {
	if len(deployment.Spec.Template.Spec.Containers) > 0 {
		return deployment.Spec.Template.Spec.Containers[0].Image
	}
	return ""
}

// getContainerImageFromStatefulSet gets the primary container image from StatefulSet
func (a *AlloyTuner) getContainerImageFromStatefulSet(statefulSet *appsv1.StatefulSet) string {
	if len(statefulSet.Spec.Template.Spec.Containers) > 0 {
		return statefulSet.Spec.Template.Spec.Containers[0].Image
	}
	return ""
}

// getContainerImageFromDaemonSet gets the primary container image from DaemonSet
func (a *AlloyTuner) getContainerImageFromDaemonSet(daemonSet *appsv1.DaemonSet) string {
	if len(daemonSet.Spec.Template.Spec.Containers) > 0 {
		return daemonSet.Spec.Template.Spec.Containers[0].Image
	}
	return ""
}

// getResourceLimits gets resource limits for the deployment
func (a *AlloyTuner) getResourceLimits(deployment *appsv1.Deployment) map[string]interface{} {
	limits := make(map[string]interface{})

	if len(deployment.Spec.Template.Spec.Containers) > 0 {
		container := deployment.Spec.Template.Spec.Containers[0]
		if container.Resources.Limits != nil {
			if cpu := container.Resources.Limits.Cpu(); cpu != nil {
				limits["cpu"] = cpu.String()
			}
			if memory := container.Resources.Limits.Memory(); memory != nil {
				limits["memory"] = memory.String()
			}
		}
		if container.Resources.Requests != nil {
			requests := make(map[string]interface{})
			if cpu := container.Resources.Requests.Cpu(); cpu != nil {
				requests["cpu"] = cpu.String()
			}
			if memory := container.Resources.Requests.Memory(); memory != nil {
				requests["memory"] = memory.String()
			}
			limits["requests"] = requests
		}
	}

	return limits
}

// getResourceLimitsFromDeployment gets resource limits for the Deployment
func (a *AlloyTuner) getResourceLimitsFromDeployment(deployment *appsv1.Deployment) map[string]interface{} {
	limits := make(map[string]interface{})
	if len(deployment.Spec.Template.Spec.Containers) > 0 {
		container := deployment.Spec.Template.Spec.Containers[0]
		if container.Resources.Limits != nil {
			if cpu := container.Resources.Limits.Cpu(); cpu != nil {
				limits["cpu"] = cpu.String()
			}
			if memory := container.Resources.Limits.Memory(); memory != nil {
				limits["memory"] = memory.String()
			}
		}
		if container.Resources.Requests != nil {
			if cpu := container.Resources.Requests.Cpu(); cpu != nil {
				limits["cpu_request"] = cpu.String()
			}
			if memory := container.Resources.Requests.Memory(); memory != nil {
				limits["memory_request"] = memory.String()
			}
		}
	}
	return limits
}

// getResourceLimitsFromStatefulSet gets resource limits for the StatefulSet
func (a *AlloyTuner) getResourceLimitsFromStatefulSet(statefulSet *appsv1.StatefulSet) map[string]interface{} {
	limits := make(map[string]interface{})
	if len(statefulSet.Spec.Template.Spec.Containers) > 0 {
		container := statefulSet.Spec.Template.Spec.Containers[0]
		if container.Resources.Limits != nil {
			if cpu := container.Resources.Limits.Cpu(); cpu != nil {
				limits["cpu"] = cpu.String()
			}
			if memory := container.Resources.Limits.Memory(); memory != nil {
				limits["memory"] = memory.String()
			}
		}
		if container.Resources.Requests != nil {
			if cpu := container.Resources.Requests.Cpu(); cpu != nil {
				limits["cpu_request"] = cpu.String()
			}
			if memory := container.Resources.Requests.Memory(); memory != nil {
				limits["memory_request"] = memory.String()
			}
		}
	}
	return limits
}

// getResourceLimitsFromDaemonSet gets resource limits for the DaemonSet
func (a *AlloyTuner) getResourceLimitsFromDaemonSet(daemonSet *appsv1.DaemonSet) map[string]interface{} {
	limits := make(map[string]interface{})
	if len(daemonSet.Spec.Template.Spec.Containers) > 0 {
		container := daemonSet.Spec.Template.Spec.Containers[0]
		if container.Resources.Limits != nil {
			if cpu := container.Resources.Limits.Cpu(); cpu != nil {
				limits["cpu"] = cpu.String()
			}
			if memory := container.Resources.Limits.Memory(); memory != nil {
				limits["memory"] = memory.String()
			}
		}
		if container.Resources.Requests != nil {
			if cpu := container.Resources.Requests.Cpu(); cpu != nil {
				limits["cpu_request"] = cpu.String()
			}
			if memory := container.Resources.Requests.Memory(); memory != nil {
				limits["memory_request"] = memory.String()
			}
		}
	}
	return limits
}

// abs returns the absolute value of an int32
func abs(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}
