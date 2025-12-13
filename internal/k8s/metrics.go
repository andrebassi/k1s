package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

type PodMetrics struct {
	Name       string
	Namespace  string
	Containers []ContainerMetrics
}

type ContainerMetrics struct {
	Name        string
	CPUUsage    string
	MemoryUsage string
	CPUPercent  float64
	MemPercent  float64
}

func GetPodMetrics(ctx context.Context, metricsClient *metricsv.Clientset, namespace, podName string) (*PodMetrics, error) {
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

func GetNamespaceMetrics(ctx context.Context, metricsClient *metricsv.Clientset, namespace string) ([]PodMetrics, error) {
	if metricsClient == nil {
		return nil, fmt.Errorf("metrics server not available")
	}

	metricsList, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
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

func formatCPU(milliCores int64) string {
	if milliCores < 1000 {
		return fmt.Sprintf("%dm", milliCores)
	}
	return fmt.Sprintf("%.2f", float64(milliCores)/1000)
}

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

type ResourceUsageSummary struct {
	CPUUsed     string
	CPUPercent  float64
	MemUsed     string
	MemPercent  float64
	IsThrottled bool
	IsOOM       bool
}

func CalculateResourceUsage(metrics *PodMetrics, pod *PodInfo) *ResourceUsageSummary {
	if metrics == nil || pod == nil {
		return nil
	}

	summary := &ResourceUsageSummary{}

	var totalCPU int64
	var totalMem int64

	for _, cm := range metrics.Containers {
		_ = cm
	}

	summary.CPUUsed = formatCPU(totalCPU)
	summary.MemUsed = formatMemory(totalMem)

	return summary
}
