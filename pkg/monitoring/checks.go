package monitoring

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"time"

	"github.com/akshaydubey29/mimirInsights/pkg/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubernetesHealthCheck checks Kubernetes API connectivity
type KubernetesHealthCheck struct {
	k8sClient *k8s.Client
}

// NewKubernetesHealthCheck creates a new Kubernetes health check
func NewKubernetesHealthCheck(k8sClient *k8s.Client) *KubernetesHealthCheck {
	return &KubernetesHealthCheck{k8sClient: k8sClient}
}

func (k *KubernetesHealthCheck) Name() string           { return "kubernetes" }
func (k *KubernetesHealthCheck) Critical() bool         { return true }
func (k *KubernetesHealthCheck) Timeout() time.Duration { return 10 * time.Second }

func (k *KubernetesHealthCheck) Check(ctx context.Context) HealthResult {
	// Test basic API connectivity
	_, err := k.k8sClient.GetNamespaces(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		return HealthResult{
			Status:  StatusUnhealthy,
			Message: fmt.Sprintf("Kubernetes API unreachable: %v", err),
			Error:   err,
		}
	}

	return HealthResult{
		Status:  StatusHealthy,
		Message: "Kubernetes API connectivity normal",
	}
}

// DatabaseHealthCheck checks database connectivity (placeholder)
type DatabaseHealthCheck struct{}

func NewDatabaseHealthCheck() *DatabaseHealthCheck {
	return &DatabaseHealthCheck{}
}

func (d *DatabaseHealthCheck) Name() string           { return "database" }
func (d *DatabaseHealthCheck) Critical() bool         { return false }
func (d *DatabaseHealthCheck) Timeout() time.Duration { return 5 * time.Second }

func (d *DatabaseHealthCheck) Check(ctx context.Context) HealthResult {
	// For now, this is a placeholder since we don't have a persistent database
	// In production, you would check actual database connectivity
	return HealthResult{
		Status:  StatusHealthy,
		Message: "No persistent database configured",
		Metadata: map[string]interface{}{
			"type": "in-memory",
		},
	}
}

// MemoryHealthCheck checks memory usage
type MemoryHealthCheck struct{}

func NewMemoryHealthCheck() *MemoryHealthCheck {
	return &MemoryHealthCheck{}
}

func (m *MemoryHealthCheck) Name() string           { return "memory" }
func (m *MemoryHealthCheck) Critical() bool         { return true }
func (m *MemoryHealthCheck) Timeout() time.Duration { return 2 * time.Second }

func (m *MemoryHealthCheck) Check(ctx context.Context) HealthResult {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	// Convert to MB
	allocMB := float64(stats.Alloc) / 1024 / 1024
	sysMB := float64(stats.Sys) / 1024 / 1024

	// Simple threshold checking
	status := StatusHealthy
	message := fmt.Sprintf("Memory usage normal: %.1f MB allocated, %.1f MB system", allocMB, sysMB)

	if allocMB > 1024 { // 1GB threshold
		status = StatusDegraded
		message = fmt.Sprintf("High memory usage: %.1f MB allocated", allocMB)
	}

	if allocMB > 2048 { // 2GB threshold
		status = StatusUnhealthy
		message = fmt.Sprintf("Critical memory usage: %.1f MB allocated", allocMB)
	}

	return HealthResult{
		Status:  status,
		Message: message,
		Metadata: map[string]interface{}{
			"alloc_mb":   allocMB,
			"sys_mb":     sysMB,
			"gc_cycles":  stats.NumGC,
			"goroutines": runtime.NumGoroutine(),
		},
	}
}

// DiskHealthCheck checks disk usage
type DiskHealthCheck struct{}

func NewDiskHealthCheck() *DiskHealthCheck {
	return &DiskHealthCheck{}
}

func (d *DiskHealthCheck) Name() string           { return "disk" }
func (d *DiskHealthCheck) Critical() bool         { return true }
func (d *DiskHealthCheck) Timeout() time.Duration { return 3 * time.Second }

func (d *DiskHealthCheck) Check(ctx context.Context) HealthResult {
	// Get disk usage for current working directory
	wd, err := os.Getwd()
	if err != nil {
		return HealthResult{
			Status:  StatusUnknown,
			Message: fmt.Sprintf("Could not get working directory: %v", err),
			Error:   err,
		}
	}

	var stat syscall.Statfs_t
	err = syscall.Statfs(wd, &stat)
	if err != nil {
		return HealthResult{
			Status:  StatusUnknown,
			Message: fmt.Sprintf("Could not get disk stats: %v", err),
			Error:   err,
		}
	}

	// Calculate disk usage
	totalBytes := stat.Blocks * uint64(stat.Bsize)
	freeBytes := stat.Bavail * uint64(stat.Bsize)
	usedBytes := totalBytes - freeBytes
	usagePercent := float64(usedBytes) / float64(totalBytes) * 100

	status := StatusHealthy
	message := fmt.Sprintf("Disk usage normal: %.1f%% used", usagePercent)

	if usagePercent > 80 {
		status = StatusDegraded
		message = fmt.Sprintf("High disk usage: %.1f%% used", usagePercent)
	}

	if usagePercent > 95 {
		status = StatusUnhealthy
		message = fmt.Sprintf("Critical disk usage: %.1f%% used", usagePercent)
	}

	return HealthResult{
		Status:  status,
		Message: message,
		Metadata: map[string]interface{}{
			"usage_percent": usagePercent,
			"total_gb":      float64(totalBytes) / 1024 / 1024 / 1024,
			"free_gb":       float64(freeBytes) / 1024 / 1024 / 1024,
			"used_gb":       float64(usedBytes) / 1024 / 1024 / 1024,
		},
	}
}

// NetworkHealthCheck checks network connectivity
type NetworkHealthCheck struct{}

func NewNetworkHealthCheck() *NetworkHealthCheck {
	return &NetworkHealthCheck{}
}

func (n *NetworkHealthCheck) Name() string           { return "network" }
func (n *NetworkHealthCheck) Critical() bool         { return false }
func (n *NetworkHealthCheck) Timeout() time.Duration { return 5 * time.Second }

func (n *NetworkHealthCheck) Check(ctx context.Context) HealthResult {
	// For production, you might want to check connectivity to external services
	// For now, this is a simple placeholder
	return HealthResult{
		Status:  StatusHealthy,
		Message: "Network connectivity assumed healthy",
		Metadata: map[string]interface{}{
			"check_type": "placeholder",
		},
	}
}
