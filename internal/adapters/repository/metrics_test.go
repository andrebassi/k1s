package repository

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsfake "k8s.io/metrics/pkg/client/clientset/versioned/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestGetPodMetrics(t *testing.T) {
	// Create fake metrics clientset with reactor to intercept Get calls
	metricsClient := metricsfake.NewSimpleClientset()

	// Use PrependReactor to return our test data
	metricsClient.PrependReactor("get", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &metricsv1beta1.PodMetrics{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
			Containers: []metricsv1beta1.ContainerMetrics{
				{
					Name: "main",
					Usage: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("128Mi"),
					},
				},
			},
		}, nil
	})

	ctx := context.Background()
	metrics, err := GetPodMetrics(ctx, metricsClient, "default", "test-pod")
	if err != nil {
		t.Fatalf("GetPodMetrics() error = %v", err)
	}

	if metrics == nil {
		t.Fatal("GetPodMetrics() returned nil")
	}

	if metrics.Name != "test-pod" {
		t.Errorf("Name = %q, want 'test-pod'", metrics.Name)
	}

	if len(metrics.Containers) != 1 {
		t.Fatalf("len(Containers) = %d, want 1", len(metrics.Containers))
	}

	if metrics.Containers[0].Name != "main" {
		t.Errorf("Container name = %q, want 'main'", metrics.Containers[0].Name)
	}
}

func TestGetPodMetrics_NilClient(t *testing.T) {
	ctx := context.Background()
	_, err := GetPodMetrics(ctx, nil, "default", "test-pod")
	if err == nil {
		t.Error("GetPodMetrics() should return error for nil client")
	}
}

func TestGetPodMetrics_NotFound(t *testing.T) {
	metricsClient := metricsfake.NewSimpleClientset()

	ctx := context.Background()
	_, err := GetPodMetrics(ctx, metricsClient, "default", "nonexistent")
	if err == nil {
		t.Error("GetPodMetrics() should return error for nonexistent pod")
	}
}

func TestGetNamespaceMetrics(t *testing.T) {
	metricsClient := metricsfake.NewSimpleClientset()

	// Use PrependReactor to return our test data for List
	metricsClient.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &metricsv1beta1.PodMetricsList{
			Items: []metricsv1beta1.PodMetrics{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: "default",
					},
					Containers: []metricsv1beta1.ContainerMetrics{
						{
							Name: "main",
							Usage: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("200m"),
								corev1.ResourceMemory: resource.MustParse("256Mi"),
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-2",
						Namespace: "default",
					},
					Containers: []metricsv1beta1.ContainerMetrics{
						{
							Name: "app",
							Usage: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("512Mi"),
							},
						},
					},
				},
			},
		}, nil
	})

	ctx := context.Background()
	metrics, err := GetNamespaceMetrics(ctx, metricsClient, "default")
	if err != nil {
		t.Fatalf("GetNamespaceMetrics() error = %v", err)
	}

	if len(metrics) != 2 {
		t.Errorf("GetNamespaceMetrics() returned %d metrics, want 2", len(metrics))
	}
}

func TestGetNamespaceMetrics_NilClient(t *testing.T) {
	ctx := context.Background()
	_, err := GetNamespaceMetrics(ctx, nil, "default")
	if err == nil {
		t.Error("GetNamespaceMetrics() should return error for nil client")
	}
}

func TestFormatCPU(t *testing.T) {
	tests := []struct {
		name       string
		milliCores int64
		expected   string
	}{
		{"zero", 0, "0m"},
		{"small value", 100, "100m"},
		{"500 millicores", 500, "500m"},
		{"just under 1 core", 999, "999m"},
		{"exactly 1 core", 1000, "1.00"},
		{"1.5 cores", 1500, "1.50"},
		{"2 cores", 2000, "2.00"},
		{"large value", 8000, "8.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCPU(tt.milliCores)
			if result != tt.expected {
				t.Errorf("formatCPU(%d) = %q, want %q", tt.milliCores, result, tt.expected)
			}
		})
	}
}

func TestFormatMemory(t *testing.T) {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero bytes", 0, "0B"},
		{"small bytes", 500, "500B"},
		{"just under 1KB", 1023, "1023B"},
		{"exactly 1KB", KB, "1.0Ki"},
		{"100KB", 100 * KB, "100.0Ki"},
		{"just under 1MB", MB - 1, "1024.0Ki"},
		{"exactly 1MB", MB, "1.0Mi"},
		{"128MB", 128 * MB, "128.0Mi"},
		{"512MB", 512 * MB, "512.0Mi"},
		{"just under 1GB", GB - 1, "1024.0Mi"},
		{"exactly 1GB", GB, "1.0Gi"},
		{"2GB", 2 * GB, "2.0Gi"},
		{"8GB", 8 * GB, "8.0Gi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMemory(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatMemory(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestCalculateResourceUsage_NilInputs(t *testing.T) {
	// Test with nil metrics
	result := CalculateResourceUsage(nil, &PodInfo{})
	if result != nil {
		t.Error("CalculateResourceUsage(nil, pod) should return nil")
	}

	// Test with nil pod
	result = CalculateResourceUsage(&PodMetrics{}, nil)
	if result != nil {
		t.Error("CalculateResourceUsage(metrics, nil) should return nil")
	}

	// Test with both nil
	result = CalculateResourceUsage(nil, nil)
	if result != nil {
		t.Error("CalculateResourceUsage(nil, nil) should return nil")
	}
}

func TestCalculateResourceUsage_ValidInputs(t *testing.T) {
	metrics := &PodMetrics{
		Name:      "test-pod",
		Namespace: "default",
		Containers: []ContainerMetrics{
			{
				Name:        "app",
				CPUUsage:    "100m",
				MemoryUsage: "128Mi",
				CPUPercent:  50.0,
				MemPercent:  25.0,
			},
		},
	}

	pod := &PodInfo{
		Name:      "test-pod",
		Namespace: "default",
	}

	result := CalculateResourceUsage(metrics, pod)
	if result == nil {
		t.Fatal("CalculateResourceUsage should not return nil for valid inputs")
	}

	// Check that summary fields are populated
	if result.CPUUsed == "" {
		t.Error("CPUUsed should not be empty")
	}
	if result.MemUsed == "" {
		t.Error("MemUsed should not be empty")
	}
}

func TestContainerMetricsStruct(t *testing.T) {
	cm := ContainerMetrics{
		Name:        "test-container",
		CPUUsage:    "100m",
		MemoryUsage: "256Mi",
		CPUPercent:  25.5,
		MemPercent:  50.0,
	}

	if cm.Name != "test-container" {
		t.Errorf("Expected Name 'test-container', got %q", cm.Name)
	}
	if cm.CPUUsage != "100m" {
		t.Errorf("Expected CPUUsage '100m', got %q", cm.CPUUsage)
	}
	if cm.MemoryUsage != "256Mi" {
		t.Errorf("Expected MemoryUsage '256Mi', got %q", cm.MemoryUsage)
	}
}

func TestPodMetricsStruct(t *testing.T) {
	pm := PodMetrics{
		Name:      "my-pod",
		Namespace: "production",
		Containers: []ContainerMetrics{
			{Name: "container1"},
			{Name: "container2"},
		},
	}

	if pm.Name != "my-pod" {
		t.Errorf("Expected Name 'my-pod', got %q", pm.Name)
	}
	if pm.Namespace != "production" {
		t.Errorf("Expected Namespace 'production', got %q", pm.Namespace)
	}
	if len(pm.Containers) != 2 {
		t.Errorf("Expected 2 containers, got %d", len(pm.Containers))
	}
}

func TestResourceUsageSummaryStruct(t *testing.T) {
	summary := ResourceUsageSummary{
		CPUUsed:     "500m",
		CPUPercent:  50.0,
		MemUsed:     "1Gi",
		MemPercent:  75.0,
		IsThrottled: true,
		IsOOM:       false,
	}

	if summary.CPUUsed != "500m" {
		t.Errorf("Expected CPUUsed '500m', got %q", summary.CPUUsed)
	}
	if !summary.IsThrottled {
		t.Error("Expected IsThrottled to be true")
	}
	if summary.IsOOM {
		t.Error("Expected IsOOM to be false")
	}
}
