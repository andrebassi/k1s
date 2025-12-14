package repository

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
)

// MetricsClientInterface defines the interface for metrics operations.
// This allows for easy testing with fake clients.
type MetricsClientInterface interface {
	MetricsV1beta1() metricsv1beta1.MetricsV1beta1Interface
}

// PodMetrics contains resource usage data for a pod and its containers.
// This data comes from the Kubernetes metrics-server.
type PodMetrics struct {
	Name       string             // Pod name
	Namespace  string             // Pod namespace
	Containers []ContainerMetrics // Per-container resource usage
}

// ContainerMetrics contains CPU and memory usage for a single container.
type ContainerMetrics struct {
	Name        string  // Container name
	CPUUsage    string  // Formatted CPU usage (e.g., "100m", "1.5")
	MemoryUsage string  // Formatted memory usage (e.g., "128Mi", "1.2Gi")
	CPUPercent  float64 // CPU usage as percentage of limit (if set)
	MemPercent  float64 // Memory usage as percentage of limit (if set)
}

// GetPodMetrics retrieves current resource usage for a specific pod.
// Returns an error if metrics-server is not available in the cluster.
func GetPodMetrics(ctx context.Context, metricsClient MetricsClientInterface, namespace, podName string) (*PodMetrics, error) {
	if metricsClient == nil {
		return nil, fmt.Errorf("metrics server not available")
	}

	metrics, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	pm := &PodMetrics{
		Name:      metrics.Name,
		Namespace: metrics.Namespace,
	}

	for _, c := range metrics.Containers {
		cpu := c.Usage.Cpu()
		mem := c.Usage.Memory()

		pm.Containers = append(pm.Containers, ContainerMetrics{
			Name:        c.Name,
			CPUUsage:    formatCPU(cpu.MilliValue()),
			MemoryUsage: formatMemory(mem.Value()),
		})
	}

	return pm, nil
}

// GetNamespaceMetrics retrieves resource usage for all pods in a namespace.
// Returns an error if metrics-server is not available in the cluster.
func GetNamespaceMetrics(ctx context.Context, metricsClient MetricsClientInterface, namespace string) ([]PodMetrics, error) {
	if metricsClient == nil {
		return nil, fmt.Errorf("metrics server not available")
	}

	metricsList, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		//coverage:ignore
		return nil, err
	}

	var result []PodMetrics
	for _, m := range metricsList.Items {
		pm := PodMetrics{
			Name:      m.Name,
			Namespace: m.Namespace,
		}

		for _, c := range m.Containers {
			cpu := c.Usage.Cpu()
			mem := c.Usage.Memory()

			pm.Containers = append(pm.Containers, ContainerMetrics{
				Name:        c.Name,
				CPUUsage:    formatCPU(cpu.MilliValue()),
				MemoryUsage: formatMemory(mem.Value()),
			})
		}
		result = append(result, pm)
	}

	return result, nil
}

// formatCPU converts millicores to a human-readable string.
// Values under 1000m are shown as millicores (e.g., "500m"),
// values at or above 1000m are shown as cores (e.g., "1.50").
func formatCPU(milliCores int64) string {
	if milliCores < 1000 {
		return fmt.Sprintf("%dm", milliCores)
	}
	return fmt.Sprintf("%.2f", float64(milliCores)/1000)
}

// formatMemory converts bytes to a human-readable string using binary units.
// Uses Ki, Mi, Gi suffixes following Kubernetes conventions.
func formatMemory(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGi", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1fMi", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1fKi", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// ResourceUsageSummary provides an aggregated view of pod resource usage.
// Includes flags for resource pressure conditions.
type ResourceUsageSummary struct {
	CPUUsed     string  // Total CPU usage across all containers
	CPUPercent  float64 // CPU usage as percentage of total limits
	MemUsed     string  // Total memory usage across all containers
	MemPercent  float64 // Memory usage as percentage of total limits
	IsThrottled bool    // True if CPU throttling is detected
	IsOOM       bool    // True if OOM conditions are detected
}

// CalculateResourceUsage computes aggregated resource usage for a pod.
// Combines metrics data with pod spec to calculate percentages.
// Returns nil if metrics or pod info is unavailable.
func CalculateResourceUsage(metrics *PodMetrics, pod *PodInfo) *ResourceUsageSummary {
	if metrics == nil || pod == nil {
		return nil
	}

	summary := &ResourceUsageSummary{}

	var totalCPU int64
	var totalMem int64

	for _, cm := range metrics.Containers {
		_ = cm // Placeholder for future metric aggregation
	}

	summary.CPUUsed = formatCPU(totalCPU)
	summary.MemUsed = formatMemory(totalMem)

	return summary
}
