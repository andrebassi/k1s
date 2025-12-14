package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestListHPAs(t *testing.T) {
	minReplicas := int32(1)
	clientset := fake.NewSimpleClientset(
		&autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "web-hpa",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
					Kind: "Deployment",
					Name: "web",
				},
				MinReplicas: &minReplicas,
				MaxReplicas: 10,
			},
			Status: autoscalingv2.HorizontalPodAutoscalerStatus{
				CurrentReplicas: 3,
			},
		},
	)

	ctx := context.Background()
	hpas, err := ListHPAs(ctx, clientset, "default")
	if err != nil {
		t.Fatalf("ListHPAs() error = %v", err)
	}

	if len(hpas) != 1 {
		t.Fatalf("ListHPAs() returned %d hpas, want 1", len(hpas))
	}

	if hpas[0].Reference != "Deployment/web" {
		t.Errorf("Reference = %q, want 'Deployment/web'", hpas[0].Reference)
	}
}

func TestGetHPA(t *testing.T) {
	minReplicas := int32(2)
	avgUtilization := int32(80)
	clientset := fake.NewSimpleClientset(
		&autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "api-hpa",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
					Kind: "Deployment",
					Name: "api",
				},
				MinReplicas: &minReplicas,
				MaxReplicas: 20,
				Metrics: []autoscalingv2.MetricSpec{
					{
						Type: autoscalingv2.ResourceMetricSourceType,
						Resource: &autoscalingv2.ResourceMetricSource{
							Name: corev1.ResourceCPU,
							Target: autoscalingv2.MetricTarget{
								Type:               autoscalingv2.UtilizationMetricType,
								AverageUtilization: &avgUtilization,
							},
						},
					},
				},
			},
			Status: autoscalingv2.HorizontalPodAutoscalerStatus{
				CurrentReplicas: 5,
				DesiredReplicas: 6,
				Conditions: []autoscalingv2.HorizontalPodAutoscalerCondition{
					{
						Type:   autoscalingv2.ScalingActive,
						Status: corev1.ConditionTrue,
						Reason: "ValidMetricFound",
					},
				},
			},
		},
	)

	ctx := context.Background()
	hpa, err := GetHPA(ctx, clientset, "default", "api-hpa")
	if err != nil {
		t.Fatalf("GetHPA() error = %v", err)
	}

	if hpa.MinReplicas != 2 {
		t.Errorf("MinReplicas = %d, want 2", hpa.MinReplicas)
	}
	if len(hpa.Metrics) != 1 {
		t.Errorf("len(Metrics) = %d, want 1", len(hpa.Metrics))
	}
}

func TestGetHPA_Full(t *testing.T) {
	cpu := int32(80)
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-hpa",
			Namespace:         "default",
			CreationTimestamp: metav1.Time{Time: time.Now().Add(-24 * time.Hour)},
			Labels:            map[string]string{"app": "web"},
			Annotations:       map[string]string{"note": "test"},
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       "web-app",
				APIVersion: "apps/v1",
			},
			MinReplicas: int32Ptr(2),
			MaxReplicas: 10,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: corev1.ResourceCPU,
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &cpu,
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 3,
			DesiredReplicas: 4,
			CurrentMetrics: []autoscalingv2.MetricStatus{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricStatus{
						Name: corev1.ResourceCPU,
						Current: autoscalingv2.MetricValueStatus{
							AverageUtilization: int32Ptr(75),
						},
					},
				},
			},
			Conditions: []autoscalingv2.HorizontalPodAutoscalerCondition{
				{
					Type:    autoscalingv2.ScalingActive,
					Status:  corev1.ConditionTrue,
					Reason:  "ValidMetricFound",
					Message: "the HPA was able to successfully calculate a replica count",
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(hpa)

	ctx := context.Background()
	data, err := GetHPA(ctx, clientset, "default", "test-hpa")
	if err != nil {
		t.Fatalf("GetHPA() error = %v", err)
	}

	if data.Name != "test-hpa" {
		t.Errorf("Name = %q, want 'test-hpa'", data.Name)
	}
	if data.Reference != "Deployment/web-app" {
		t.Errorf("Reference = %q, want 'Deployment/web-app'", data.Reference)
	}
	if data.MinReplicas != 2 {
		t.Errorf("MinReplicas = %d, want 2", data.MinReplicas)
	}
	if data.MaxReplicas != 10 {
		t.Errorf("MaxReplicas = %d, want 10", data.MaxReplicas)
	}
	if data.CurrentReplicas != 3 {
		t.Errorf("CurrentReplicas = %d, want 3", data.CurrentReplicas)
	}
	if data.DesiredReplicas != 4 {
		t.Errorf("DesiredReplicas = %d, want 4", data.DesiredReplicas)
	}
	if len(data.Metrics) != 1 {
		t.Errorf("len(Metrics) = %d, want 1", len(data.Metrics))
	}
	if len(data.Conditions) != 1 {
		t.Errorf("len(Conditions) = %d, want 1", len(data.Conditions))
	}
}

func TestGetHPA_WithMemoryAndExternal(t *testing.T) {
	mem := int32(70)
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "complex-hpa",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: "api",
			},
			MinReplicas: int32Ptr(1),
			MaxReplicas: 20,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: corev1.ResourceMemory,
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &mem,
						},
					},
				},
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricSource{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "queue_length",
						},
						Target: autoscalingv2.MetricTarget{
							Type:  autoscalingv2.AverageValueMetricType,
							Value: resourceQuantityPtr("100"),
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 5,
			DesiredReplicas: 5,
			CurrentMetrics: []autoscalingv2.MetricStatus{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricStatus{
						Name: corev1.ResourceMemory,
						Current: autoscalingv2.MetricValueStatus{
							AverageUtilization: int32Ptr(65),
						},
					},
				},
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricStatus{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "queue_length",
						},
						Current: autoscalingv2.MetricValueStatus{
							Value: resourceQuantityPtr("85"),
						},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(hpa)

	ctx := context.Background()
	data, err := GetHPA(ctx, clientset, "default", "complex-hpa")
	if err != nil {
		t.Fatalf("GetHPA() error = %v", err)
	}

	if len(data.Metrics) != 2 {
		t.Errorf("len(Metrics) = %d, want 2", len(data.Metrics))
	}
}

func TestFormatHPATargets_ResourceMetrics(t *testing.T) {
	cpu := int32(80)
	mem := int32(70)

	hpa := autoscalingv2.HorizontalPodAutoscaler{
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: corev1.ResourceCPU,
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &cpu,
						},
					},
				},
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: corev1.ResourceMemory,
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &mem,
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentMetrics: []autoscalingv2.MetricStatus{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricStatus{
						Name: corev1.ResourceCPU,
						Current: autoscalingv2.MetricValueStatus{
							AverageUtilization: int32Ptr(60),
						},
					},
				},
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricStatus{
						Name: corev1.ResourceMemory,
						Current: autoscalingv2.MetricValueStatus{
							AverageUtilization: int32Ptr(50),
						},
					},
				},
			},
		},
	}

	result := formatHPATargets(hpa)
	if result == "" {
		t.Error("formatHPATargets() returned empty string")
	}
	if !strings.Contains(result, "cpu") {
		t.Errorf("formatHPATargets() should contain 'cpu', got %q", result)
	}
	if !strings.Contains(result, "memory") {
		t.Errorf("formatHPATargets() should contain 'memory', got %q", result)
	}
}

func TestFormatHPATargets_ExternalMetrics(t *testing.T) {
	hpa := autoscalingv2.HorizontalPodAutoscaler{
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricSource{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "queue_messages",
						},
						Target: autoscalingv2.MetricTarget{
							Type:  autoscalingv2.AverageValueMetricType,
							Value: resourceQuantityPtr("50"),
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentMetrics: []autoscalingv2.MetricStatus{
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricStatus{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "queue_messages",
						},
						Current: autoscalingv2.MetricValueStatus{
							Value: resourceQuantityPtr("30"),
						},
					},
				},
			},
		},
	}

	result := formatHPATargets(hpa)
	if result == "" {
		t.Error("formatHPATargets() returned empty string")
	}
	if !strings.Contains(result, "queue_messages") {
		t.Errorf("formatHPATargets() should contain 'queue_messages', got %q", result)
	}
}

func TestGetHPA_ExternalMetric(t *testing.T) {
	avgValue := resource.MustParse("10")
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keda-hpa",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: "worker",
			},
			MinReplicas: int32Ptr(1),
			MaxReplicas: 10,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricSource{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "rabbitmq_queue_length",
						},
						Target: autoscalingv2.MetricTarget{
							Type:         autoscalingv2.AverageValueMetricType,
							AverageValue: &avgValue,
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 3,
			DesiredReplicas: 5,
			CurrentMetrics: []autoscalingv2.MetricStatus{
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricStatus{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "rabbitmq_queue_length",
						},
						Current: autoscalingv2.MetricValueStatus{
							AverageValue: &avgValue,
						},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(hpa)

	ctx := context.Background()
	data, err := GetHPA(ctx, clientset, "default", "keda-hpa")
	if err != nil {
		t.Fatalf("GetHPA() error = %v", err)
	}

	if data.Name != "keda-hpa" {
		t.Errorf("Name = %q, want 'keda-hpa'", data.Name)
	}
	if len(data.Metrics) != 1 {
		t.Fatalf("len(Metrics) = %d, want 1", len(data.Metrics))
	}
	if data.Metrics[0].Type != "External" {
		t.Errorf("Metric type = %q, want 'External'", data.Metrics[0].Type)
	}
	if data.Metrics[0].Name != "rabbitmq_queue_length" {
		t.Errorf("Metric name = %q, want 'rabbitmq_queue_length'", data.Metrics[0].Name)
	}
}

func TestGetHPA_NotFound(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	ctx := context.Background()
	_, err := GetHPA(ctx, clientset, "default", "nonexistent")
	if err == nil {
		t.Error("GetHPA() should return error for nonexistent HPA")
	}
}

func TestListHPAs_Error(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "horizontalpodautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListHPAs(ctx, clientset, "default")
	if err == nil {
		t.Error("ListHPAs() should return error on API failure")
	}
}

func TestGetHPA_PodsMetric(t *testing.T) {
	avgValue := resource.MustParse("100")
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pods-metric-hpa",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: "worker",
			},
			MinReplicas: int32Ptr(1),
			MaxReplicas: 10,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.PodsMetricSourceType,
					Pods: &autoscalingv2.PodsMetricSource{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "packets-per-second",
						},
						Target: autoscalingv2.MetricTarget{
							Type:         autoscalingv2.AverageValueMetricType,
							AverageValue: &avgValue,
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 2,
			DesiredReplicas: 3,
		},
	}

	clientset := fake.NewSimpleClientset(hpa)
	ctx := context.Background()

	data, err := GetHPA(ctx, clientset, "default", "pods-metric-hpa")
	if err != nil {
		t.Fatalf("GetHPA() error = %v", err)
	}

	if len(data.Metrics) != 1 {
		t.Fatalf("len(Metrics) = %d, want 1", len(data.Metrics))
	}

	if data.Metrics[0].Type != "Pods" {
		t.Errorf("Metric type = %q, want 'Pods'", data.Metrics[0].Type)
	}
	if data.Metrics[0].Name != "packets-per-second" {
		t.Errorf("Metric name = %q, want 'packets-per-second'", data.Metrics[0].Name)
	}
}

// ============================================
// HPA with ObjectMetricSourceType
// ============================================

func TestGetHPA_ObjectMetric(t *testing.T) {
	targetValue := resource.MustParse("200")
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "object-metric-hpa",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: "api",
			},
			MinReplicas: int32Ptr(2),
			MaxReplicas: 20,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ObjectMetricSourceType,
					Object: &autoscalingv2.ObjectMetricSource{
						DescribedObject: autoscalingv2.CrossVersionObjectReference{
							Kind: "Service",
							Name: "api-service",
						},
						Metric: autoscalingv2.MetricIdentifier{
							Name: "requests-per-second",
						},
						Target: autoscalingv2.MetricTarget{
							Type:  autoscalingv2.ValueMetricType,
							Value: &targetValue,
						},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(hpa)
	ctx := context.Background()

	data, err := GetHPA(ctx, clientset, "default", "object-metric-hpa")
	if err != nil {
		t.Fatalf("GetHPA() error = %v", err)
	}

	if len(data.Metrics) != 1 {
		t.Fatalf("len(Metrics) = %d, want 1", len(data.Metrics))
	}

	if data.Metrics[0].Type != "Object" {
		t.Errorf("Metric type = %q, want 'Object'", data.Metrics[0].Type)
	}
	if data.Metrics[0].Name != "requests-per-second" {
		t.Errorf("Metric name = %q, want 'requests-per-second'", data.Metrics[0].Name)
	}
	if data.Metrics[0].Target != "200" {
		t.Errorf("Metric target = %q, want '200'", data.Metrics[0].Target)
	}
}

// ============================================
// HPA with ObjectMetricSourceType (AverageValue)
// ============================================

func TestGetHPA_ObjectMetric_AverageValue(t *testing.T) {
	avgValue := resource.MustParse("50")
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "object-avg-hpa",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: "web",
			},
			MinReplicas: int32Ptr(1),
			MaxReplicas: 10,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ObjectMetricSourceType,
					Object: &autoscalingv2.ObjectMetricSource{
						DescribedObject: autoscalingv2.CrossVersionObjectReference{
							Kind: "Ingress",
							Name: "main-ingress",
						},
						Metric: autoscalingv2.MetricIdentifier{
							Name: "hits-per-second",
						},
						Target: autoscalingv2.MetricTarget{
							Type:         autoscalingv2.AverageValueMetricType,
							AverageValue: &avgValue,
						},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(hpa)
	ctx := context.Background()

	data, err := GetHPA(ctx, clientset, "default", "object-avg-hpa")
	if err != nil {
		t.Fatalf("GetHPA() error = %v", err)
	}

	if len(data.Metrics) != 1 {
		t.Fatalf("len(Metrics) = %d, want 1", len(data.Metrics))
	}

	if data.Metrics[0].Type != "Object" {
		t.Errorf("Metric type = %q, want 'Object'", data.Metrics[0].Type)
	}
	if data.Metrics[0].Target != "50" {
		t.Errorf("Metric target = %q, want '50'", data.Metrics[0].Target)
	}
}

// ============================================
// HPA with External metric using Value (not AverageValue)
// ============================================

func TestGetHPA_ExternalMetric_Value(t *testing.T) {
	targetValue := resource.MustParse("1000")
	currentValue := resource.MustParse("500")
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "external-value-hpa",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: "processor",
			},
			MinReplicas: int32Ptr(1),
			MaxReplicas: 50,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricSource{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "sqs_queue_messages",
						},
						Target: autoscalingv2.MetricTarget{
							Type:  autoscalingv2.ValueMetricType,
							Value: &targetValue,
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 5,
			DesiredReplicas: 10,
			CurrentMetrics: []autoscalingv2.MetricStatus{
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricStatus{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "sqs_queue_messages",
						},
						Current: autoscalingv2.MetricValueStatus{
							Value: &currentValue,
						},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(hpa)
	ctx := context.Background()

	data, err := GetHPA(ctx, clientset, "default", "external-value-hpa")
	if err != nil {
		t.Fatalf("GetHPA() error = %v", err)
	}

	if len(data.Metrics) != 1 {
		t.Fatalf("len(Metrics) = %d, want 1", len(data.Metrics))
	}

	if data.Metrics[0].Type != "External" {
		t.Errorf("Metric type = %q, want 'External'", data.Metrics[0].Type)
	}
	// resource.Quantity formats 1000 as "1k"
	if data.Metrics[0].Target != "1k" {
		t.Errorf("Metric target = %q, want '1k'", data.Metrics[0].Target)
	}
	if data.Metrics[0].Current != "500" {
		t.Errorf("Metric current = %q, want '500'", data.Metrics[0].Current)
	}
}

// ============================================
// HPA with Resource metric using AverageValue (not AverageUtilization)
// ============================================

func TestGetHPA_ResourceMetric_AverageValue(t *testing.T) {
	avgValue := resource.MustParse("500m")
	currentAvgValue := resource.MustParse("300m")
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "resource-avg-hpa",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: "app",
			},
			MinReplicas: int32Ptr(1),
			MaxReplicas: 10,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: corev1.ResourceCPU,
						Target: autoscalingv2.MetricTarget{
							Type:         autoscalingv2.AverageValueMetricType,
							AverageValue: &avgValue,
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 3,
			DesiredReplicas: 4,
			CurrentMetrics: []autoscalingv2.MetricStatus{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricStatus{
						Name: corev1.ResourceCPU,
						Current: autoscalingv2.MetricValueStatus{
							AverageValue: &currentAvgValue,
						},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(hpa)
	ctx := context.Background()

	data, err := GetHPA(ctx, clientset, "default", "resource-avg-hpa")
	if err != nil {
		t.Fatalf("GetHPA() error = %v", err)
	}

	if len(data.Metrics) != 1 {
		t.Fatalf("len(Metrics) = %d, want 1", len(data.Metrics))
	}

	if data.Metrics[0].Type != "Resource" {
		t.Errorf("Metric type = %q, want 'Resource'", data.Metrics[0].Type)
	}
	if data.Metrics[0].Target != "500m" {
		t.Errorf("Metric target = %q, want '500m'", data.Metrics[0].Target)
	}
	if data.Metrics[0].Current != "300m" {
		t.Errorf("Metric current = %q, want '300m'", data.Metrics[0].Current)
	}
}

// ============================================
// HPA with Conditions
// ============================================

func TestGetHPA_WithConditions(t *testing.T) {
	cpu := int32(80)
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hpa-with-conditions",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: "app",
			},
			MinReplicas: int32Ptr(1),
			MaxReplicas: 10,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: corev1.ResourceCPU,
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &cpu,
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 3,
			DesiredReplicas: 3,
			Conditions: []autoscalingv2.HorizontalPodAutoscalerCondition{
				{
					Type:    autoscalingv2.ScalingActive,
					Status:  corev1.ConditionTrue,
					Reason:  "ValidMetricFound",
					Message: "the HPA was able to successfully calculate a replica count from cpu resource utilization",
				},
				{
					Type:    autoscalingv2.AbleToScale,
					Status:  corev1.ConditionTrue,
					Reason:  "ReadyForNewScale",
					Message: "the HPA controller is able to scale the target resource",
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(hpa)
	ctx := context.Background()

	data, err := GetHPA(ctx, clientset, "default", "hpa-with-conditions")
	if err != nil {
		t.Fatalf("GetHPA() error = %v", err)
	}

	if len(data.Conditions) != 2 {
		t.Fatalf("len(Conditions) = %d, want 2", len(data.Conditions))
	}

	found := false
	for _, c := range data.Conditions {
		if c.Type == "ScalingActive" {
			found = true
			if c.Status != "True" {
				t.Errorf("ScalingActive status = %q, want 'True'", c.Status)
			}
			if c.Reason != "ValidMetricFound" {
				t.Errorf("ScalingActive reason = %q, want 'ValidMetricFound'", c.Reason)
			}
		}
	}
	if !found {
		t.Error("Expected to find ScalingActive condition")
	}
}
