package drift

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/akshaydubey29/mimirInsights/pkg/k8s"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Detector handles configuration drift detection
type Detector struct {
	k8sClient     *k8s.Client
	baselineStore *BaselineStore
}

// DriftStatus represents the drift status of a configuration
type DriftStatus struct {
	Resource     string                 `json:"resource"`
	Namespace    string                 `json:"namespace"`
	Name         string                 `json:"name"`
	Status       string                 `json:"status"` // "no_drift", "drifted", "new", "deleted"
	LastChecked  time.Time              `json:"last_checked"`
	Changes      []ConfigChange         `json:"changes"`
	RiskLevel    string                 `json:"risk_level"`
	BaselineHash string                 `json:"baseline_hash"`
	CurrentHash  string                 `json:"current_hash"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ConfigChange represents a specific configuration change
type ConfigChange struct {
	Type        string `json:"type"` // "added", "modified", "deleted"
	Key         string `json:"key"`
	OldValue    string `json:"old_value"`
	NewValue    string `json:"new_value"`
	Impact      string `json:"impact"` // "low", "medium", "high", "critical"
	Description string `json:"description"`
}

// BaselineConfig represents a baseline configuration
type BaselineConfig struct {
	Resource     string            `json:"resource"`
	Namespace    string            `json:"namespace"`
	Name         string            `json:"name"`
	Hash         string            `json:"hash"`
	Data         map[string]string `json:"data"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	CreatedAt    time.Time         `json:"created_at"`
	LastModified time.Time         `json:"last_modified"`
}

// DriftReport represents a comprehensive drift report
type DriftReport struct {
	GeneratedAt    time.Time     `json:"generated_at"`
	TotalResources int           `json:"total_resources"`
	DriftedCount   int           `json:"drifted_count"`
	NewCount       int           `json:"new_count"`
	DeletedCount   int           `json:"deleted_count"`
	DriftStatuses  []DriftStatus `json:"drift_statuses"`
	Summary        DriftSummary  `json:"summary"`
}

// DriftSummary provides a summary of drift detection results
type DriftSummary struct {
	LowRisk      int `json:"low_risk"`
	MediumRisk   int `json:"medium_risk"`
	HighRisk     int `json:"high_risk"`
	CriticalRisk int `json:"critical_risk"`
}

// NewDetector creates a new drift detector
func NewDetector(k8sClient *k8s.Client) *Detector {
	return &Detector{
		k8sClient:     k8sClient,
		baselineStore: NewBaselineStore(),
	}
}

// DetectDrift performs drift detection across all monitored resources
func (d *Detector) DetectDrift(ctx context.Context, namespaces []string) (*DriftReport, error) {
	logrus.Info("Starting configuration drift detection")

	report := &DriftReport{
		GeneratedAt:   time.Now(),
		DriftStatuses: []DriftStatus{},
	}

	for _, namespace := range namespaces {
		// Detect ConfigMap drift
		configMapStatuses, err := d.detectConfigMapDrift(ctx, namespace)
		if err != nil {
			logrus.Warnf("Failed to detect ConfigMap drift in %s: %v", namespace, err)
			continue
		}
		report.DriftStatuses = append(report.DriftStatuses, configMapStatuses...)

		// Future: Add other resource types (Secrets, Deployments, etc.)
	}

	// Calculate summary statistics
	d.calculateSummary(report)

	logrus.Infof("Drift detection completed. Found %d drifted resources out of %d total",
		report.DriftedCount, report.TotalResources)

	return report, nil
}

// detectConfigMapDrift detects drift in ConfigMaps
func (d *Detector) detectConfigMapDrift(ctx context.Context, namespace string) ([]DriftStatus, error) {
	var statuses []DriftStatus

	// Get current ConfigMaps
	configMaps, err := d.k8sClient.GetConfigMaps(ctx, namespace, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ConfigMaps: %w", err)
	}

	// Check each ConfigMap against baseline
	for _, cm := range configMaps.Items {
		if d.shouldMonitorConfigMap(&cm) {
			status := d.analyzeConfigMapDrift(&cm)
			statuses = append(statuses, status)
		}
	}

	// Check for deleted ConfigMaps
	deletedStatuses := d.checkForDeletedConfigMaps(namespace, configMaps.Items)
	statuses = append(statuses, deletedStatuses...)

	return statuses, nil
}

// analyzeConfigMapDrift analyzes drift for a specific ConfigMap
func (d *Detector) analyzeConfigMapDrift(cm *corev1.ConfigMap) DriftStatus {
	resourceKey := fmt.Sprintf("configmap/%s/%s", cm.Namespace, cm.Name)
	currentHash := d.calculateConfigMapHash(cm)

	status := DriftStatus{
		Resource:    "ConfigMap",
		Namespace:   cm.Namespace,
		Name:        cm.Name,
		LastChecked: time.Now(),
		CurrentHash: currentHash,
		Changes:     []ConfigChange{},
		Metadata: map[string]interface{}{
			"labels":      cm.Labels,
			"annotations": cm.Annotations,
		},
	}

	// Get baseline configuration
	baseline := d.baselineStore.GetBaseline(resourceKey)
	if baseline == nil {
		// No baseline exists - this is a new resource
		status.Status = "new"
		status.RiskLevel = "medium"

		// Store as new baseline
		newBaseline := &BaselineConfig{
			Resource:     "ConfigMap",
			Namespace:    cm.Namespace,
			Name:         cm.Name,
			Hash:         currentHash,
			Data:         cm.Data,
			Labels:       cm.Labels,
			Annotations:  cm.Annotations,
			CreatedAt:    time.Now(),
			LastModified: time.Now(),
		}
		d.baselineStore.StoreBaseline(resourceKey, newBaseline)

		return status
	}

	status.BaselineHash = baseline.Hash

	// Compare current state with baseline
	if currentHash == baseline.Hash {
		status.Status = "no_drift"
		status.RiskLevel = "low"
	} else {
		status.Status = "drifted"
		status.Changes = d.analyzeConfigMapChanges(baseline, cm)
		status.RiskLevel = d.calculateRiskLevel(status.Changes)

		// Update baseline with current state
		baseline.Hash = currentHash
		baseline.Data = cm.Data
		baseline.Labels = cm.Labels
		baseline.Annotations = cm.Annotations
		baseline.LastModified = time.Now()
		d.baselineStore.StoreBaseline(resourceKey, baseline)
	}

	return status
}

// analyzeConfigMapChanges analyzes specific changes between baseline and current
func (d *Detector) analyzeConfigMapChanges(baseline *BaselineConfig, current *corev1.ConfigMap) []ConfigChange {
	var changes []ConfigChange

	// Check for data changes
	for key, oldValue := range baseline.Data {
		if newValue, exists := current.Data[key]; exists {
			if oldValue != newValue {
				changes = append(changes, ConfigChange{
					Type:        "modified",
					Key:         fmt.Sprintf("data.%s", key),
					OldValue:    oldValue,
					NewValue:    newValue,
					Impact:      d.assessDataChangeImpact(key, oldValue, newValue),
					Description: fmt.Sprintf("Configuration value changed for key '%s'", key),
				})
			}
		} else {
			changes = append(changes, ConfigChange{
				Type:        "deleted",
				Key:         fmt.Sprintf("data.%s", key),
				OldValue:    oldValue,
				NewValue:    "",
				Impact:      d.assessDataChangeImpact(key, oldValue, ""),
				Description: fmt.Sprintf("Configuration key '%s' was deleted", key),
			})
		}
	}

	// Check for new data keys
	for key, newValue := range current.Data {
		if _, exists := baseline.Data[key]; !exists {
			changes = append(changes, ConfigChange{
				Type:        "added",
				Key:         fmt.Sprintf("data.%s", key),
				OldValue:    "",
				NewValue:    newValue,
				Impact:      d.assessDataChangeImpact(key, "", newValue),
				Description: fmt.Sprintf("New configuration key '%s' was added", key),
			})
		}
	}

	// Check label changes
	changes = append(changes, d.analyzeMapChanges("labels", baseline.Labels, current.Labels)...)

	// Check annotation changes
	changes = append(changes, d.analyzeMapChanges("annotations", baseline.Annotations, current.Annotations)...)

	return changes
}

// analyzeMapChanges analyzes changes in labels or annotations
func (d *Detector) analyzeMapChanges(mapType string, baseline, current map[string]string) []ConfigChange {
	var changes []ConfigChange

	// Handle nil maps
	if baseline == nil {
		baseline = make(map[string]string)
	}
	if current == nil {
		current = make(map[string]string)
	}

	// Check for modifications and deletions
	for key, oldValue := range baseline {
		if newValue, exists := current[key]; exists {
			if oldValue != newValue {
				changes = append(changes, ConfigChange{
					Type:        "modified",
					Key:         fmt.Sprintf("%s.%s", mapType, key),
					OldValue:    oldValue,
					NewValue:    newValue,
					Impact:      "low",
					Description: fmt.Sprintf("%s value changed for key '%s'", mapType, key),
				})
			}
		} else {
			changes = append(changes, ConfigChange{
				Type:        "deleted",
				Key:         fmt.Sprintf("%s.%s", mapType, key),
				OldValue:    oldValue,
				NewValue:    "",
				Impact:      "low",
				Description: fmt.Sprintf("%s key '%s' was deleted", mapType, key),
			})
		}
	}

	// Check for additions
	for key, newValue := range current {
		if _, exists := baseline[key]; !exists {
			changes = append(changes, ConfigChange{
				Type:        "added",
				Key:         fmt.Sprintf("%s.%s", mapType, key),
				OldValue:    "",
				NewValue:    newValue,
				Impact:      "low",
				Description: fmt.Sprintf("New %s key '%s' was added", mapType, key),
			})
		}
	}

	return changes
}

// assessDataChangeImpact assesses the impact of a data change
func (d *Detector) assessDataChangeImpact(key, oldValue, newValue string) string {
	// Critical configuration keys
	criticalKeys := []string{
		"runtime-config.yaml", "overrides.yaml", "limits.yaml",
		"distributor.yaml", "ingester.yaml", "querier.yaml",
	}

	// High impact configuration keys
	highImpactKeys := []string{
		"nginx.conf", "prometheus.yml", "alertmanager.yml",
		"config.yaml", "config.yml",
	}

	keyLower := key
	for _, criticalKey := range criticalKeys {
		if keyLower == criticalKey {
			return "critical"
		}
	}

	for _, highKey := range highImpactKeys {
		if keyLower == highKey {
			return "high"
		}
	}

	// Check for limit-related changes
	if d.isLimitRelatedChange(key, oldValue, newValue) {
		return "high"
	}

	// Default to medium for data changes, low for metadata
	return "medium"
}

// isLimitRelatedChange checks if the change is related to limits
func (d *Detector) isLimitRelatedChange(key, oldValue, newValue string) bool {
	limitIndicators := []string{
		"limit", "max_", "rate", "timeout", "threshold",
		"ingestion", "series", "memory", "cpu",
	}

	keyLower := key
	for _, indicator := range limitIndicators {
		if len(keyLower) > 0 && len(indicator) > 0 {
			// Simple contains check
			for i := 0; i <= len(keyLower)-len(indicator); i++ {
				if keyLower[i:i+len(indicator)] == indicator {
					return true
				}
			}
		}
	}

	return false
}

// calculateRiskLevel calculates overall risk level based on changes
func (d *Detector) calculateRiskLevel(changes []ConfigChange) string {
	if len(changes) == 0 {
		return "low"
	}

	criticalCount := 0
	highCount := 0
	mediumCount := 0

	for _, change := range changes {
		switch change.Impact {
		case "critical":
			criticalCount++
		case "high":
			highCount++
		case "medium":
			mediumCount++
		}
	}

	if criticalCount > 0 {
		return "critical"
	}
	if highCount > 0 {
		return "high"
	}
	if mediumCount > 0 {
		return "medium"
	}

	return "low"
}

// calculateConfigMapHash calculates a hash of the ConfigMap data
func (d *Detector) calculateConfigMapHash(cm *corev1.ConfigMap) string {
	hasher := sha256.New()

	// Sort keys for consistent hashing
	var keys []string
	for k := range cm.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Hash data
	for _, k := range keys {
		hasher.Write([]byte(k))
		hasher.Write([]byte(cm.Data[k]))
	}

	// Include labels and annotations in hash
	var labelKeys []string
	for k := range cm.Labels {
		labelKeys = append(labelKeys, k)
	}
	sort.Strings(labelKeys)
	for _, k := range labelKeys {
		hasher.Write([]byte("label:" + k))
		hasher.Write([]byte(cm.Labels[k]))
	}

	var annotationKeys []string
	for k := range cm.Annotations {
		annotationKeys = append(annotationKeys, k)
	}
	sort.Strings(annotationKeys)
	for _, k := range annotationKeys {
		hasher.Write([]byte("annotation:" + k))
		hasher.Write([]byte(cm.Annotations[k]))
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

// shouldMonitorConfigMap determines if a ConfigMap should be monitored
func (d *Detector) shouldMonitorConfigMap(cm *corev1.ConfigMap) bool {
	// Skip system ConfigMaps
	systemNamespaces := []string{"kube-system", "kube-public", "kube-node-lease"}
	for _, ns := range systemNamespaces {
		if cm.Namespace == ns {
			return false
		}
	}

	// Monitor ConfigMaps with specific patterns
	monitorPatterns := []string{
		"mimir", "cortex", "runtime", "overrides", "limits",
		"config", "alloy", "nginx", "prometheus",
	}

	cmName := cm.Name
	for _, pattern := range monitorPatterns {
		if len(cmName) >= len(pattern) {
			for i := 0; i <= len(cmName)-len(pattern); i++ {
				if cmName[i:i+len(pattern)] == pattern {
					return true
				}
			}
		}
	}

	return false
}

// checkForDeletedConfigMaps checks for ConfigMaps that were deleted
func (d *Detector) checkForDeletedConfigMaps(namespace string, currentCMs []corev1.ConfigMap) []DriftStatus {
	var statuses []DriftStatus

	// Get all baselines for this namespace
	baselines := d.baselineStore.GetBaselinesForNamespace(namespace)

	// Create a map of current ConfigMaps
	currentCMMap := make(map[string]bool)
	for _, cm := range currentCMs {
		key := fmt.Sprintf("configmap/%s/%s", cm.Namespace, cm.Name)
		currentCMMap[key] = true
	}

	// Check for deleted ConfigMaps
	for key, baseline := range baselines {
		if baseline.Resource == "ConfigMap" && !currentCMMap[key] {
			statuses = append(statuses, DriftStatus{
				Resource:     "ConfigMap",
				Namespace:    baseline.Namespace,
				Name:         baseline.Name,
				Status:       "deleted",
				LastChecked:  time.Now(),
				RiskLevel:    "high",
				BaselineHash: baseline.Hash,
				CurrentHash:  "",
				Changes: []ConfigChange{{
					Type:        "deleted",
					Key:         "resource",
					OldValue:    "exists",
					NewValue:    "deleted",
					Impact:      "high",
					Description: "ConfigMap was deleted",
				}},
			})
		}
	}

	return statuses
}

// calculateSummary calculates summary statistics for the drift report
func (d *Detector) calculateSummary(report *DriftReport) {
	report.TotalResources = len(report.DriftStatuses)

	for _, status := range report.DriftStatuses {
		switch status.Status {
		case "drifted":
			report.DriftedCount++
		case "new":
			report.NewCount++
		case "deleted":
			report.DeletedCount++
		}

		switch status.RiskLevel {
		case "low":
			report.Summary.LowRisk++
		case "medium":
			report.Summary.MediumRisk++
		case "high":
			report.Summary.HighRisk++
		case "critical":
			report.Summary.CriticalRisk++
		}
	}
}

// GetDriftHistory returns historical drift information
func (d *Detector) GetDriftHistory(resourceKey string, days int) ([]DriftStatus, error) {
	// TODO: Implement drift history storage and retrieval
	return []DriftStatus{}, nil
}

// CreateBaseline creates a baseline for the current state
func (d *Detector) CreateBaseline(ctx context.Context, namespaces []string) error {
	logrus.Info("Creating configuration baselines")

	for _, namespace := range namespaces {
		configMaps, err := d.k8sClient.GetConfigMaps(ctx, namespace, metav1.ListOptions{})
		if err != nil {
			logrus.Warnf("Failed to get ConfigMaps for baseline in %s: %v", namespace, err)
			continue
		}

		for _, cm := range configMaps.Items {
			if d.shouldMonitorConfigMap(&cm) {
				resourceKey := fmt.Sprintf("configmap/%s/%s", cm.Namespace, cm.Name)
				baseline := &BaselineConfig{
					Resource:     "ConfigMap",
					Namespace:    cm.Namespace,
					Name:         cm.Name,
					Hash:         d.calculateConfigMapHash(&cm),
					Data:         cm.Data,
					Labels:       cm.Labels,
					Annotations:  cm.Annotations,
					CreatedAt:    time.Now(),
					LastModified: time.Now(),
				}
				d.baselineStore.StoreBaseline(resourceKey, baseline)
			}
		}
	}

	logrus.Info("Configuration baselines created successfully")
	return nil
}
