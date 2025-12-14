package repository

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestIngressReferencesService(t *testing.T) {
	pathTypePrefix := networkingv1.PathTypePrefix

	ing := networkingv1.Ingress{
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathTypePrefix,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{Name: "web-service"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if !ingressReferencesService(ing, "web-service") {
		t.Error("ingressReferencesService() should return true")
	}

	if ingressReferencesService(ing, "other-service") {
		t.Error("ingressReferencesService() should return false for non-matching service")
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		str      string
		expected bool
	}{
		{"found", []string{"a", "b", "c"}, "b", true},
		{"not found", []string{"a", "b", "c"}, "d", false},
		{"empty slice", []string{}, "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.str)
			if result != tt.expected {
				t.Errorf("contains() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseProbe(t *testing.T) {
	if parseProbe(nil) != nil {
		t.Error("parseProbe(nil) should return nil")
	}

	httpProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/health",
				Port:   intstr.FromInt(8080),
				Scheme: corev1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: 10,
	}

	result := parseProbe(httpProbe)
	if result == nil {
		t.Fatal("parseProbe() should not return nil for HTTP probe")
	}
	if result.Type != "HTTP" {
		t.Errorf("Type = %q, want 'HTTP'", result.Type)
	}

	tcpProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt(3306)},
		},
	}
	result = parseProbe(tcpProbe)
	if result.Type != "TCP" {
		t.Errorf("Type = %q, want 'TCP'", result.Type)
	}

	execProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{Command: []string{"cat", "/tmp/healthy"}},
		},
	}
	result = parseProbe(execProbe)
	if result.Type != "Exec" {
		t.Errorf("Type = %q, want 'Exec'", result.Type)
	}
}

func TestLabelsMatch(t *testing.T) {
	tests := []struct {
		name     string
		selector map[string]string
		labels   map[string]string
		expected bool
	}{
		{
			name:     "exact match",
			selector: map[string]string{"app": "nginx"},
			labels:   map[string]string{"app": "nginx"},
			expected: true,
		},
		{
			name:     "selector subset of labels",
			selector: map[string]string{"app": "nginx"},
			labels:   map[string]string{"app": "nginx", "env": "prod", "version": "v1"},
			expected: true,
		},
		{
			name:     "selector not in labels",
			selector: map[string]string{"app": "nginx"},
			labels:   map[string]string{"app": "redis"},
			expected: false,
		},
		{
			name:     "selector key missing from labels",
			selector: map[string]string{"app": "nginx", "env": "prod"},
			labels:   map[string]string{"app": "nginx"},
			expected: false,
		},
		{
			name:     "empty selector matches everything",
			selector: map[string]string{},
			labels:   map[string]string{"app": "nginx"},
			expected: true,
		},
		{
			name:     "empty labels with non-empty selector",
			selector: map[string]string{"app": "nginx"},
			labels:   map[string]string{},
			expected: false,
		},
		{
			name:     "both empty",
			selector: map[string]string{},
			labels:   map[string]string{},
			expected: true,
		},
		{
			name:     "multiple selector labels all match",
			selector: map[string]string{"app": "nginx", "env": "prod"},
			labels:   map[string]string{"app": "nginx", "env": "prod", "version": "v1"},
			expected: true,
		},
		{
			name:     "multiple selector labels partial match",
			selector: map[string]string{"app": "nginx", "env": "prod"},
			labels:   map[string]string{"app": "nginx", "env": "staging"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := labelsMatch(tt.selector, tt.labels)
			if result != tt.expected {
				t.Errorf("labelsMatch(%v, %v) = %v, want %v", tt.selector, tt.labels, result, tt.expected)
			}
		})
	}
}

func TestAllResourceTypes(t *testing.T) {
	// Verify AllResourceTypes contains expected types
	expectedTypes := map[ResourceType]bool{
		ResourceDeployments:  true,
		ResourceStatefulSets: true,
		ResourceDaemonSets:   true,
		ResourceJobs:         true,
		ResourceCronJobs:     true,
		ResourcePods:         true,
	}

	if len(AllResourceTypes) != len(expectedTypes) {
		t.Errorf("AllResourceTypes has %d types, expected %d", len(AllResourceTypes), len(expectedTypes))
	}

	for _, rt := range AllResourceTypes {
		if !expectedTypes[rt] {
			t.Errorf("Unexpected resource type in AllResourceTypes: %s", rt)
		}
	}
}

func TestIngressReferencesService_NilHTTPRule(t *testing.T) {
	// Test with a rule that has no HTTP section
	ing := networkingv1.Ingress{
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "example.com",
					// HTTP is nil
				},
			},
		},
	}

	if ingressReferencesService(ing, "any-service") {
		t.Error("ingressReferencesService() should return false when HTTP is nil")
	}
}

func TestParseProbe_gRPCType(t *testing.T) {
	// Test gRPC probe with service name
	port := int32(9000)
	serviceName := "health.v1.HealthService"
	grpcProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			GRPC: &corev1.GRPCAction{
				Port:    port,
				Service: &serviceName,
			},
		},
	}

	result := parseProbe(grpcProbe)
	if result == nil {
		t.Fatal("parseProbe should not return nil for valid gRPC probe")
	}
	if result.Type != "gRPC" {
		t.Errorf("Type = %q, want 'gRPC'", result.Type)
	}
	if result.Port != 9000 {
		t.Errorf("Port = %d, want 9000", result.Port)
	}
}

func TestExtractRolloutReplicas(t *testing.T) {
	tests := []struct {
		name             string
		rolloutObj       map[string]interface{}
		expectedReplicas int32
		expectedReady    int32
	}{
		{
			name:             "empty object",
			rolloutObj:       map[string]interface{}{},
			expectedReplicas: 1,
			expectedReady:    0,
		},
		{
			name: "int64 replicas",
			rolloutObj: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": int64(5),
				},
				"status": map[string]interface{}{
					"readyReplicas": int64(3),
				},
			},
			expectedReplicas: 5,
			expectedReady:    3,
		},
		{
			name: "float64 replicas",
			rolloutObj: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": float64(10),
				},
				"status": map[string]interface{}{
					"readyReplicas": float64(8),
				},
			},
			expectedReplicas: 10,
			expectedReady:    8,
		},
		{
			name: "fallback to availableReplicas",
			rolloutObj: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": int64(3),
				},
				"status": map[string]interface{}{
					"availableReplicas": int64(2),
				},
			},
			expectedReplicas: 3,
			expectedReady:    2,
		},
		{
			name: "availableReplicas float64",
			rolloutObj: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": float64(4),
				},
				"status": map[string]interface{}{
					"availableReplicas": float64(3),
				},
			},
			expectedReplicas: 4,
			expectedReady:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replicas, readyReplicas := extractRolloutReplicas(tt.rolloutObj)
			if replicas != tt.expectedReplicas {
				t.Errorf("extractRolloutReplicas() replicas = %d, want %d", replicas, tt.expectedReplicas)
			}
			if readyReplicas != tt.expectedReady {
				t.Errorf("extractRolloutReplicas() readyReplicas = %d, want %d", readyReplicas, tt.expectedReady)
			}
		})
	}
}

func TestCountReadyEndpoints(t *testing.T) {
	tests := []struct {
		name     string
		epSlices *discoveryv1.EndpointSliceList
		expected int
	}{
		{
			name:     "nil epSlices",
			epSlices: nil,
			expected: 0,
		},
		{
			name:     "empty epSlices",
			epSlices: &discoveryv1.EndpointSliceList{},
			expected: 0,
		},
		{
			name: "mixed ready states",
			epSlices: &discoveryv1.EndpointSliceList{
				Items: []discoveryv1.EndpointSlice{
					{
						Endpoints: []discoveryv1.Endpoint{
							{Conditions: discoveryv1.EndpointConditions{Ready: boolPtr(true)}},
							{Conditions: discoveryv1.EndpointConditions{Ready: boolPtr(false)}},
							{Conditions: discoveryv1.EndpointConditions{Ready: boolPtr(true)}},
						},
					},
					{
						Endpoints: []discoveryv1.Endpoint{
							{Conditions: discoveryv1.EndpointConditions{Ready: boolPtr(true)}},
							{Conditions: discoveryv1.EndpointConditions{Ready: nil}}, // nil ready
						},
					},
				},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countReadyEndpoints(tt.epSlices)
			if result != tt.expected {
				t.Errorf("countReadyEndpoints() = %d, want %d", result, tt.expected)
			}
		})
	}
}
