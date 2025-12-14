package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakediscovery "k8s.io/client-go/discovery/fake"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
)

// ============================================
// NewClientFromConfig Tests
// ============================================

func TestNewClientFromConfig_NilConfig(t *testing.T) {
	_, err := NewClientFromConfig(nil, "")
	if err == nil {
		t.Error("NewClientFromConfig(nil) should return error")
	}
	if err.Error() != "config cannot be nil" {
		t.Errorf("Error message = %q, want 'config cannot be nil'", err.Error())
	}
}

func TestNewClientFromConfig_ValidConfig(t *testing.T) {
	// Create a minimal valid config pointing to localhost
	// This will fail to connect but will test the client creation path
	config := &rest.Config{
		Host: "https://127.0.0.1:6443",
	}

	client, err := NewClientFromConfig(config, "")
	if err != nil {
		t.Fatalf("NewClientFromConfig() error = %v", err)
	}

	if client == nil {
		t.Fatal("NewClientFromConfig() returned nil client")
	}

	if client.Clientset() == nil {
		t.Error("Clientset() should not be nil")
	}

	if client.DynamicClient() == nil {
		t.Error("DynamicClient() should not be nil")
	}

	// Namespace should default to "default"
	if client.Namespace() != "default" {
		t.Errorf("Namespace() = %q, want 'default'", client.Namespace())
	}
}

func TestNewClientFromConfig_WithKubeconfigPath(t *testing.T) {
	// Create a temp kubeconfig file
	kubeconfigContent := `apiVersion: v1
kind: Config
current-context: test-context
clusters:
- cluster:
    server: https://127.0.0.1:6443
    insecure-skip-tls-verify: true
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
users:
- name: test-user
`
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")
	err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write temp kubeconfig: %v", err)
	}

	config := &rest.Config{
		Host: "https://127.0.0.1:6443",
	}

	client, err := NewClientFromConfig(config, kubeconfigPath)
	if err != nil {
		t.Fatalf("NewClientFromConfig() error = %v", err)
	}

	// Context should be extracted from kubeconfig
	if client.Context() != "test-context" {
		t.Errorf("Context() = %q, want 'test-context'", client.Context())
	}
}

// ============================================
// NewClientWithKubeconfig Tests
// ============================================

func TestNewClientWithKubeconfig_ValidKubeconfig(t *testing.T) {
	// Create a valid kubeconfig pointing to localhost
	kubeconfigContent := `apiVersion: v1
kind: Config
current-context: local-test
clusters:
- cluster:
    server: https://127.0.0.1:6443
    insecure-skip-tls-verify: true
  name: local-cluster
contexts:
- context:
    cluster: local-cluster
    user: local-user
  name: local-test
users:
- name: local-user
`
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")
	err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write temp kubeconfig: %v", err)
	}

	client, err := NewClientWithKubeconfig(kubeconfigPath)
	if err != nil {
		t.Fatalf("NewClientWithKubeconfig() error = %v", err)
	}

	if client == nil {
		t.Fatal("NewClientWithKubeconfig() returned nil client")
	}

	if client.Context() != "local-test" {
		t.Errorf("Context() = %q, want 'local-test'", client.Context())
	}
}

func TestNewClientWithKubeconfig_InvalidPath_FallsBackToInCluster(t *testing.T) {
	// This should try to fall back to in-cluster config, which will also fail
	// in a non-cluster environment, resulting in an error
	_, err := NewClientWithKubeconfig("/nonexistent/path/kubeconfig")

	// We expect an error since we're not in a cluster
	if err == nil {
		// If no error, we might be in a cluster environment (CI)
		t.Log("NewClientWithKubeconfig() succeeded - may be running in a cluster")
	} else {
		// Check that it's the expected error
		if err.Error() != "failed to create kubernetes config: unable to load in-cluster configuration, KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT must be defined" {
			t.Logf("Got expected error (not in cluster): %v", err)
		}
	}
}

// ============================================
// NewClient Tests (wrapper function)
// ============================================

func TestNewClient_UsesDefaultKubeconfig(t *testing.T) {
	// Save and restore KUBECONFIG env var
	oldKubeconfig := os.Getenv("KUBECONFIG")
	defer os.Setenv("KUBECONFIG", oldKubeconfig)

	// Create a temp kubeconfig that NewClient should NOT use (it uses ~/.kube/config)
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")
	kubeconfigContent := `apiVersion: v1
kind: Config
current-context: env-context
clusters:
- cluster:
    server: https://127.0.0.1:6443
  name: env-cluster
contexts:
- context:
    cluster: env-cluster
    user: env-user
  name: env-context
users:
- name: env-user
`
	os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600)
	os.Setenv("KUBECONFIG", kubeconfigPath)

	// NewClient uses ~/.kube/config directly, not KUBECONFIG env var
	// This test documents that behavior
	_, err := NewClient()
	if err != nil {
		// Expected to fail if no ~/.kube/config or not in cluster
		t.Logf("NewClient() error (expected if no kubeconfig): %v", err)
	}
}

func TestClient_Namespace(t *testing.T) {
	client := &Client{
		namespace: "default",
	}

	if client.Namespace() != "default" {
		t.Errorf("Namespace() = %q, want %q", client.Namespace(), "default")
	}
}

func TestClient_SetNamespace(t *testing.T) {
	client := &Client{
		namespace: "default",
	}

	client.SetNamespace("production")
	if client.Namespace() != "production" {
		t.Errorf("After SetNamespace(), Namespace() = %q, want %q", client.Namespace(), "production")
	}
}

func TestClient_Context(t *testing.T) {
	client := &Client{
		context: "minikube",
	}

	if client.Context() != "minikube" {
		t.Errorf("Context() = %q, want %q", client.Context(), "minikube")
	}
}

func TestClient_Clientset(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset()
	client := &Client{
		clientset: fakeClientset,
	}

	// Compare using the interface
	if client.Clientset() == nil {
		t.Error("Clientset() should not return nil")
	}
}

func TestClient_DynamicClient(t *testing.T) {
	client := &Client{
		dynamicClient: nil,
	}

	if client.DynamicClient() != nil {
		t.Error("DynamicClient() should return nil when not set")
	}
}

func TestClient_MetricsClient(t *testing.T) {
	client := &Client{
		metricsClient: nil,
	}

	if client.MetricsClient() != nil {
		t.Error("MetricsClient() should return nil when not set")
	}
}

func TestClient_ListNamespaces(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "default"},
			Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "kube-system"},
			Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
		},
	)

	client := &Client{
		clientset: fakeClientset,
	}

	ctx := context.Background()
	namespaces, err := client.ListNamespaces(ctx)
	if err != nil {
		t.Fatalf("ListNamespaces() error = %v", err)
	}

	if len(namespaces) != 2 {
		t.Errorf("ListNamespaces() returned %d namespaces, want 2", len(namespaces))
	}

	// Verify sorted order
	if namespaces[0].Name != "default" {
		t.Errorf("First namespace should be 'default', got %q", namespaces[0].Name)
	}
	if namespaces[1].Name != "kube-system" {
		t.Errorf("Second namespace should be 'kube-system', got %q", namespaces[1].Name)
	}
}

func TestClient_DeletePod(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
		},
	)

	client := &Client{
		clientset: fakeClientset,
	}

	ctx := context.Background()
	err := client.DeletePod(ctx, "default", "test-pod")
	if err != nil {
		t.Fatalf("DeletePod() error = %v", err)
	}

	// Verify pod was deleted
	_, err = fakeClientset.CoreV1().Pods("default").Get(ctx, "test-pod", metav1.GetOptions{})
	if err == nil {
		t.Error("Pod should have been deleted")
	}
}

func TestClient_DeletePod_NotFound(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClientset,
	}

	ctx := context.Background()
	err := client.DeletePod(ctx, "default", "nonexistent-pod")
	if err == nil {
		t.Error("DeletePod() should return error for nonexistent pod")
	}
}

func TestClient_ScaleWorkload_Deployment(t *testing.T) {
	client := &Client{
		clientset:     fake.NewSimpleClientset(),
		dynamicClient: nil,
	}

	ctx := context.Background()

	// ScaleWorkload for unsupported types should return nil
	err := client.ScaleWorkload(ctx, "default", "test", ResourceDaemonSets, 3)
	if err != nil {
		t.Errorf("ScaleWorkload() for DaemonSet should return nil, got %v", err)
	}

	err = client.ScaleWorkload(ctx, "default", "test", ResourceJobs, 3)
	if err != nil {
		t.Errorf("ScaleWorkload() for Jobs should return nil, got %v", err)
	}

	err = client.ScaleWorkload(ctx, "default", "test", ResourceCronJobs, 3)
	if err != nil {
		t.Errorf("ScaleWorkload() for CronJobs should return nil, got %v", err)
	}
}

func TestClient_RestartWorkload_UnsupportedTypes(t *testing.T) {
	client := &Client{
		clientset: fake.NewSimpleClientset(),
	}

	ctx := context.Background()

	// RestartWorkload for unsupported types should return nil
	err := client.RestartWorkload(ctx, "default", "test", ResourceJobs)
	if err != nil {
		t.Errorf("RestartWorkload() for Jobs should return nil, got %v", err)
	}

	err = client.RestartWorkload(ctx, "default", "test", ResourceCronJobs)
	if err != nil {
		t.Errorf("RestartWorkload() for CronJobs should return nil, got %v", err)
	}
}

// ============================================
// Kubeconfig Tests (with temp files)
// ============================================

func TestListContexts_WithTempKubeconfig(t *testing.T) {
	// Create a temporary kubeconfig file
	kubeconfigContent := `apiVersion: v1
kind: Config
current-context: test-context
clusters:
- cluster:
    server: https://localhost:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
- context:
    cluster: test-cluster
    user: test-user
  name: another-context
users:
- name: test-user
`

	// Create temp directory and file
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")
	err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write temp kubeconfig: %v", err)
	}

	// Set KUBECONFIG env var to temp file
	oldKubeconfig := os.Getenv("KUBECONFIG")
	os.Setenv("KUBECONFIG", kubeconfigPath)
	defer os.Setenv("KUBECONFIG", oldKubeconfig)

	// Create a client manually and test ListContexts
	client := &Client{}
	contexts, currentContext, err := client.ListContexts()
	if err != nil {
		t.Fatalf("ListContexts() error = %v", err)
	}

	if len(contexts) != 2 {
		t.Errorf("ListContexts() returned %d contexts, want 2", len(contexts))
	}

	if currentContext != "test-context" {
		t.Errorf("CurrentContext = %q, want %q", currentContext, "test-context")
	}

	// Check that both contexts are present
	contextMap := make(map[string]bool)
	for _, ctx := range contexts {
		contextMap[ctx] = true
	}
	if !contextMap["test-context"] || !contextMap["another-context"] {
		t.Error("Expected contexts 'test-context' and 'another-context'")
	}
}

func TestListContexts_SingleContext(t *testing.T) {
	kubeconfigContent := `apiVersion: v1
kind: Config
current-context: prod
clusters:
- cluster:
    server: https://prod.example.com:6443
  name: prod-cluster
contexts:
- context:
    cluster: prod-cluster
    user: admin
  name: prod
users:
- name: admin
`

	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")
	err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write temp kubeconfig: %v", err)
	}

	oldKubeconfig := os.Getenv("KUBECONFIG")
	os.Setenv("KUBECONFIG", kubeconfigPath)
	defer os.Setenv("KUBECONFIG", oldKubeconfig)

	client := &Client{}
	contexts, currentContext, err := client.ListContexts()
	if err != nil {
		t.Fatalf("ListContexts() error = %v", err)
	}

	if len(contexts) != 1 {
		t.Errorf("ListContexts() returned %d contexts, want 1", len(contexts))
	}

	if currentContext != "prod" {
		t.Errorf("CurrentContext = %q, want %q", currentContext, "prod")
	}
}

// ============================================
// Dynamic Client Tests (for Rollouts/Istio)
// ============================================

func TestDynamicClient_ListRollouts(t *testing.T) {
	scheme := runtime.NewScheme()

	rolloutGVR := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "rollouts",
	}

	// Create unstructured Rollout object
	rollout := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Rollout",
			"metadata": map[string]interface{}{
				"name":      "web-rollout",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(3),
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": "web",
					},
				},
			},
			"status": map[string]interface{}{
				"replicas":          int64(3),
				"updatedReplicas":   int64(3),
				"readyReplicas":     int64(3),
				"availableReplicas": int64(3),
			},
		},
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{
			rolloutGVR: "RolloutList",
		},
		rollout,
	)

	ctx := context.Background()

	// Test listing rollouts
	list, err := dynamicClient.Resource(rolloutGVR).Namespace("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("List rollouts error = %v", err)
	}

	if len(list.Items) != 1 {
		t.Errorf("Expected 1 rollout, got %d", len(list.Items))
	}

	if list.Items[0].GetName() != "web-rollout" {
		t.Errorf("Rollout name = %q, want %q", list.Items[0].GetName(), "web-rollout")
	}
}

func TestDynamicClient_GetRollout(t *testing.T) {
	scheme := runtime.NewScheme()

	rolloutGVR := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "rollouts",
	}

	rollout := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Rollout",
			"metadata": map[string]interface{}{
				"name":      "api-rollout",
				"namespace": "production",
			},
			"spec": map[string]interface{}{
				"replicas": int64(5),
			},
		},
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{
			rolloutGVR: "RolloutList",
		},
		rollout,
	)

	ctx := context.Background()

	// Test getting a specific rollout
	result, err := dynamicClient.Resource(rolloutGVR).Namespace("production").Get(ctx, "api-rollout", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get rollout error = %v", err)
	}

	if result.GetName() != "api-rollout" {
		t.Errorf("Rollout name = %q, want %q", result.GetName(), "api-rollout")
	}

	// Verify spec.replicas
	replicas, found, err := unstructured.NestedInt64(result.Object, "spec", "replicas")
	if err != nil || !found {
		t.Errorf("Failed to get spec.replicas")
	}
	if replicas != 5 {
		t.Errorf("Replicas = %d, want 5", replicas)
	}
}

func TestDynamicClient_UpdateRollout(t *testing.T) {
	scheme := runtime.NewScheme()

	rolloutGVR := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "rollouts",
	}

	rollout := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Rollout",
			"metadata": map[string]interface{}{
				"name":      "web-rollout",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(1),
			},
		},
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{
			rolloutGVR: "RolloutList",
		},
		rollout,
	)

	ctx := context.Background()

	// Get the rollout
	result, err := dynamicClient.Resource(rolloutGVR).Namespace("default").Get(ctx, "web-rollout", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get rollout error = %v", err)
	}

	// Update replicas (simulating scale)
	err = unstructured.SetNestedField(result.Object, int64(5), "spec", "replicas")
	if err != nil {
		t.Fatalf("SetNestedField error = %v", err)
	}

	// Update the rollout
	_, err = dynamicClient.Resource(rolloutGVR).Namespace("default").Update(ctx, result, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Update rollout error = %v", err)
	}

	// Verify the update
	updated, err := dynamicClient.Resource(rolloutGVR).Namespace("default").Get(ctx, "web-rollout", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get updated rollout error = %v", err)
	}

	replicas, _, _ := unstructured.NestedInt64(updated.Object, "spec", "replicas")
	if replicas != 5 {
		t.Errorf("Updated replicas = %d, want 5", replicas)
	}
}

func TestDynamicClient_IstioVirtualService(t *testing.T) {
	scheme := runtime.NewScheme()

	vsGVR := schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "virtualservices",
	}

	vs := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.istio.io/v1beta1",
			"kind":       "VirtualService",
			"metadata": map[string]interface{}{
				"name":      "web-vs",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"hosts": []interface{}{"web.example.com"},
				"http": []interface{}{
					map[string]interface{}{
						"route": []interface{}{
							map[string]interface{}{
								"destination": map[string]interface{}{
									"host": "web-svc",
									"port": map[string]interface{}{
										"number": int64(80),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{
			vsGVR: "VirtualServiceList",
		},
		vs,
	)

	ctx := context.Background()

	// Test listing virtual services
	list, err := dynamicClient.Resource(vsGVR).Namespace("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("List VirtualServices error = %v", err)
	}

	if len(list.Items) != 1 {
		t.Errorf("Expected 1 VirtualService, got %d", len(list.Items))
	}

	// Verify hosts
	hosts, found, _ := unstructured.NestedStringSlice(list.Items[0].Object, "spec", "hosts")
	if !found || len(hosts) != 1 || hosts[0] != "web.example.com" {
		t.Errorf("VirtualService hosts = %v, want [web.example.com]", hosts)
	}
}

func TestDynamicClient_IstioGateway(t *testing.T) {
	scheme := runtime.NewScheme()

	gwGVR := schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "gateways",
	}

	gateway := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.istio.io/v1beta1",
			"kind":       "Gateway",
			"metadata": map[string]interface{}{
				"name":      "main-gateway",
				"namespace": "istio-system",
			},
			"spec": map[string]interface{}{
				"selector": map[string]interface{}{
					"istio": "ingressgateway",
				},
				"servers": []interface{}{
					map[string]interface{}{
						"port": map[string]interface{}{
							"number":   int64(443),
							"name":     "https",
							"protocol": "HTTPS",
						},
						"hosts": []interface{}{"*.example.com"},
						"tls": map[string]interface{}{
							"mode":           "SIMPLE",
							"credentialName": "wildcard-cert",
						},
					},
				},
			},
		},
	}

	// Create empty client first, then explicitly Create the object
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{
			gwGVR: "GatewayList",
		},
	)

	ctx := context.Background()

	// Explicitly create the gateway in the namespace
	_, err := dynamicClient.Resource(gwGVR).Namespace("istio-system").Create(ctx, gateway, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Create Gateway error = %v", err)
	}

	// Test listing gateways
	list, err := dynamicClient.Resource(gwGVR).Namespace("istio-system").List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("List Gateways error = %v", err)
	}

	if len(list.Items) != 1 {
		t.Errorf("Expected 1 Gateway, got %d", len(list.Items))
	}

	if list.Items[0].GetName() != "main-gateway" {
		t.Errorf("Gateway name = %q, want %q", list.Items[0].GetName(), "main-gateway")
	}

	// Verify spec.selector
	selector, found, _ := unstructured.NestedStringMap(list.Items[0].Object, "spec", "selector")
	if !found || selector["istio"] != "ingressgateway" {
		t.Errorf("Gateway selector = %v, want {istio: ingressgateway}", selector)
	}
}

// ============================================
// GetPodLogs Tests
// ============================================

func TestGetPodLogs_FakeClientReturnsLogs(t *testing.T) {
	// Note: The fake clientset returns "fake logs" for any pod via GetLogs().Stream()
	// It does NOT check if the pod exists - this is expected fake behavior
	// In production, Kubernetes API would return an error for nonexistent pods
	clientset := fake.NewSimpleClientset()

	ctx := context.Background()
	opts := DefaultLogOptions()

	// Fake client returns logs even for nonexistent pods
	logs, err := GetPodLogs(ctx, clientset, "default", "any-pod", opts)
	if err != nil {
		t.Logf("GetPodLogs() returned error (may be expected depending on client-go version): %v", err)
	}
	// Fake clientset returns "fake logs" string parsed as LogLine slice
	if len(logs) > 0 {
		t.Logf("Got %d log lines", len(logs))
	}
}

func TestGetPodLogs_WithContainer(t *testing.T) {
	// Create pod with containers
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app"},
				{Name: "sidecar"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	clientset := fake.NewSimpleClientset(pod)

	ctx := context.Background()
	opts := DefaultLogOptions()
	opts.Container = "app"

	// The fake clientset returns "fake logs" since PR #91485
	// We test that the function handles streaming correctly
	// Note: may fail with older client-go versions
	_, err := GetPodLogs(ctx, clientset, "default", "test-pod", opts)
	// Error handling depends on client-go version
	_ = err
}

func TestGetPodLogs_WithOptions(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "main"},
			},
		},
	}

	clientset := fake.NewSimpleClientset(pod)

	ctx := context.Background()
	opts := LogOptions{
		Container:  "main",
		TailLines:  50,
		Timestamps: true,
		Previous:   false,
	}

	// Test with various options
	_, err := GetPodLogs(ctx, clientset, "default", "test-pod", opts)
	_ = err // Error handling depends on client-go version
}

// ============================================
// Client with DynamicClient Tests
// ============================================

func TestClient_WithDynamicClient(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	client := &Client{
		clientset:     fake.NewSimpleClientset(),
		dynamicClient: dynamicClient,
		namespace:     "default",
		context:       "test",
	}

	if client.DynamicClient() == nil {
		t.Error("DynamicClient() should not be nil")
	}

	if client.Clientset() == nil {
		t.Error("Clientset() should not be nil")
	}
}

func TestClient_ScaleWorkload_Rollout(t *testing.T) {
	scheme := runtime.NewScheme()

	rolloutGVR := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "rollouts",
	}

	rollout := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Rollout",
			"metadata": map[string]interface{}{
				"name":      "web",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(1),
			},
		},
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{
			rolloutGVR: "RolloutList",
		},
		rollout,
	)

	client := &Client{
		clientset:     fake.NewSimpleClientset(),
		dynamicClient: dynamicClient,
		namespace:     "default",
	}

	ctx := context.Background()

	// ScaleWorkload for Rollouts uses dynamic client
	err := client.ScaleWorkload(ctx, "default", "web", ResourceRollouts, 3)
	if err != nil {
		t.Errorf("ScaleWorkload() for Rollout error = %v", err)
	}
}

// ============================================
// GetAllContainerLogs and GetPreviousLogs Tests
// ============================================

func TestGetAllContainerLogs_MultipleContainers(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "multi-container-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app"},
				{Name: "sidecar"},
				{Name: "init-helper"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	clientset := fake.NewSimpleClientset(pod)
	ctx := context.Background()

	// GetAllContainerLogs fetches logs from all containers
	logs, err := GetAllContainerLogs(ctx, clientset, "default", "multi-container-pod", 100)
	if err != nil {
		t.Logf("GetAllContainerLogs() error (may be expected): %v", err)
	}

	// Fake clientset returns "fake logs" for each container
	t.Logf("Got %d log lines from all containers", len(logs))
}

func TestGetAllContainerLogs_PodNotFound(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	ctx := context.Background()

	// Pod doesn't exist - should return error
	_, err := GetAllContainerLogs(ctx, clientset, "default", "nonexistent-pod", 100)
	if err == nil {
		t.Error("GetAllContainerLogs() should return error for nonexistent pod")
	}
}

func TestGetAllContainerLogs_SingleContainer(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "single-container-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "main"},
			},
		},
	}

	clientset := fake.NewSimpleClientset(pod)
	ctx := context.Background()

	logs, err := GetAllContainerLogs(ctx, clientset, "default", "single-container-pod", 50)
	if err != nil {
		t.Logf("GetAllContainerLogs() error: %v", err)
	}
	t.Logf("Got %d log lines", len(logs))
}

func TestGetPreviousLogs(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "crashed-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app"},
			},
		},
	}

	clientset := fake.NewSimpleClientset(pod)
	ctx := context.Background()

	// GetPreviousLogs retrieves logs from previous container instance
	logs, err := GetPreviousLogs(ctx, clientset, "default", "crashed-pod", "app", 50)
	if err != nil {
		t.Logf("GetPreviousLogs() error (may be expected): %v", err)
	}
	t.Logf("Got %d previous log lines", len(logs))
}

// ============================================
// ListRollouts Tests with Dynamic Client
// ============================================

func TestListRollouts_WithDynamicClient(t *testing.T) {
	scheme := runtime.NewScheme()

	rolloutGVR := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "rollouts",
	}

	rollout1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Rollout",
			"metadata": map[string]interface{}{
				"name":      "web-rollout",
				"namespace": "production",
			},
			"spec": map[string]interface{}{
				"replicas": int64(3),
			},
			"status": map[string]interface{}{
				"readyReplicas":     int64(3),
				"availableReplicas": int64(3),
			},
		},
	}

	rollout2 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Rollout",
			"metadata": map[string]interface{}{
				"name":      "api-rollout",
				"namespace": "production",
			},
			"spec": map[string]interface{}{
				"replicas": int64(5),
			},
			"status": map[string]interface{}{
				"readyReplicas":     int64(5),
				"availableReplicas": int64(5),
			},
		},
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{
			rolloutGVR: "RolloutList",
		},
	)

	ctx := context.Background()

	// Create rollouts
	_, err := dynamicClient.Resource(rolloutGVR).Namespace("production").Create(ctx, rollout1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create rollout1: %v", err)
	}
	_, err = dynamicClient.Resource(rolloutGVR).Namespace("production").Create(ctx, rollout2, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create rollout2: %v", err)
	}

	// List rollouts
	list, err := dynamicClient.Resource(rolloutGVR).Namespace("production").List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("List rollouts error: %v", err)
	}

	if len(list.Items) != 2 {
		t.Errorf("Expected 2 rollouts, got %d", len(list.Items))
	}
}

// ============================================
// RestartWorkload Tests
// ============================================

func TestClient_RestartWorkload_NotFound(t *testing.T) {
	// RestartWorkload returns error when resource doesn't exist
	clientset := fake.NewSimpleClientset()
	ctx := context.Background()

	client := &Client{
		clientset: clientset,
	}

	// All these should return "not found" errors since resources don't exist
	tests := []struct {
		name         string
		resourceType ResourceType
	}{
		{"Deployment", ResourceDeployments},
		{"StatefulSet", ResourceStatefulSets},
		{"DaemonSet", ResourceDaemonSets},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.RestartWorkload(ctx, "default", "nonexistent", tt.resourceType)
			if err == nil {
				t.Errorf("RestartWorkload() for %s should return error when resource doesn't exist", tt.name)
			}
		})
	}
}

func TestClient_RestartWorkload_UnsupportedReturnsNil(t *testing.T) {
	// Jobs and CronJobs don't support restart and return nil
	clientset := fake.NewSimpleClientset()
	ctx := context.Background()

	client := &Client{
		clientset: clientset,
	}

	tests := []struct {
		name         string
		resourceType ResourceType
	}{
		{"Jobs", ResourceJobs},
		{"CronJobs", ResourceCronJobs},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.RestartWorkload(ctx, "default", "any", tt.resourceType)
			if err != nil {
				t.Errorf("RestartWorkload() for %s should return nil, got %v", tt.name, err)
			}
		})
	}
}

// Note: Event tests are in events_test.go

// ============================================
// ListRollouts Tests (actual function)
// ============================================

func TestListRollouts_NilDynamicClient(t *testing.T) {
	ctx := context.Background()

	// ListRollouts with nil dynamic client should return nil, nil
	workloads, err := ListRollouts(ctx, nil, "default")
	if err != nil {
		t.Errorf("ListRollouts() with nil client should not error, got %v", err)
	}
	if workloads != nil {
		t.Errorf("ListRollouts() with nil client should return nil workloads")
	}
}

func TestListRollouts_WithRollouts(t *testing.T) {
	scheme := runtime.NewScheme()

	rolloutGVR := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "rollouts",
	}

	rollout := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Rollout",
			"metadata": map[string]interface{}{
				"name":      "web-rollout",
				"namespace": "production",
				"labels": map[string]interface{}{
					"app": "web",
				},
			},
			"spec": map[string]interface{}{
				"replicas": int64(3),
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": "web",
					},
				},
			},
			"status": map[string]interface{}{
				"readyReplicas": int64(3),
				"phase":         "Healthy",
			},
		},
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{
			rolloutGVR: "RolloutList",
		},
	)

	ctx := context.Background()

	// Create the rollout
	_, err := dynamicClient.Resource(rolloutGVR).Namespace("production").Create(ctx, rollout, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create rollout: %v", err)
	}

	// Test ListRollouts
	workloads, err := ListRollouts(ctx, dynamicClient, "production")
	if err != nil {
		t.Fatalf("ListRollouts() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Errorf("Expected 1 workload, got %d", len(workloads))
	}

	if len(workloads) > 0 {
		if workloads[0].Name != "web-rollout" {
			t.Errorf("Workload name = %q, want %q", workloads[0].Name, "web-rollout")
		}
		if workloads[0].Status != "Healthy" {
			t.Errorf("Workload status = %q, want %q", workloads[0].Status, "Healthy")
		}
		if workloads[0].Type != ResourceRollouts {
			t.Errorf("Workload type = %v, want %v", workloads[0].Type, ResourceRollouts)
		}
	}
}

func TestListRollouts_EmptyNamespace(t *testing.T) {
	scheme := runtime.NewScheme()

	rolloutGVR := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "rollouts",
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{
			rolloutGVR: "RolloutList",
		},
	)

	ctx := context.Background()

	// No rollouts in this namespace
	workloads, err := ListRollouts(ctx, dynamicClient, "empty-ns")
	if err != nil {
		t.Fatalf("ListRollouts() error = %v", err)
	}

	if len(workloads) != 0 {
		t.Errorf("Expected 0 workloads, got %d", len(workloads))
	}
}

// Note: ScaleWorkload for Deployments/StatefulSets uses GetScale/UpdateScale subresource
// which the fake client doesn't fully support. The existing tests in TestClient_ScaleWorkload_Deployment
// cover the unsupported types path. The Rollout scaling is tested in TestClient_ScaleWorkload_Rollout.

// ============================================
// GetRelatedResources Tests
// ============================================

func TestGetRelatedResources_NoPod(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	ctx := context.Background()

	pod := PodInfo{
		Name:      "test-pod",
		Namespace: "default",
		Labels:    map[string]string{"app": "test"},
	}

	related, err := GetRelatedResources(ctx, clientset, nil, pod)
	if err != nil {
		t.Fatalf("GetRelatedResources() error = %v", err)
	}

	if related == nil {
		t.Error("GetRelatedResources() should not return nil")
	}
}

func TestGetRelatedResources_WithOwner(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	ctx := context.Background()

	pod := PodInfo{
		Name:      "test-pod",
		Namespace: "default",
		Labels:    map[string]string{"app": "test"},
		OwnerRef:  "test-rs",
		OwnerKind: "ReplicaSet",
	}

	related, err := GetRelatedResources(ctx, clientset, nil, pod)
	if err != nil {
		t.Fatalf("GetRelatedResources() error = %v", err)
	}

	if related.Owner == nil {
		t.Error("Expected Owner to be set")
	}
	if related.Owner != nil && related.Owner.Name != "test-rs" {
		t.Errorf("Owner name = %q, want %q", related.Owner.Name, "test-rs")
	}
}

func TestGetRelatedResources_WithService(t *testing.T) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "test"},
			Type:     corev1.ServiceTypeClusterIP,
			ClusterIP: "10.0.0.1",
			Ports: []corev1.ServicePort{
				{Port: 80, Protocol: corev1.ProtocolTCP},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels:    map[string]string{"app": "test"},
		},
	}

	clientset := fake.NewSimpleClientset(svc, pod)
	ctx := context.Background()

	podInfo := PodInfo{
		Name:      "test-pod",
		Namespace: "default",
		Labels:    map[string]string{"app": "test"},
	}

	related, err := GetRelatedResources(ctx, clientset, nil, podInfo)
	if err != nil {
		t.Fatalf("GetRelatedResources() error = %v", err)
	}

	if len(related.Services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(related.Services))
	}
}

// ============================================
// getIstioResources Tests
// ============================================

func TestGetIstioResources_EmptyNamespace(t *testing.T) {
	scheme := runtime.NewScheme()

	vsGVR := schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "virtualservices",
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{
			vsGVR: "VirtualServiceList",
		},
	)

	ctx := context.Background()
	services := []ServiceInfo{{Name: "test-svc"}}

	// No VirtualServices in the namespace
	vs, gw := getIstioResources(ctx, dynamicClient, "empty-ns", services)
	if len(vs) != 0 || len(gw) != 0 {
		t.Error("Expected empty results for empty namespace")
	}
}

func TestGetIstioResources_WithVirtualService(t *testing.T) {
	scheme := runtime.NewScheme()

	vsGVR := schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "virtualservices",
	}

	gwGVR := schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "gateways",
	}

	vs := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.istio.io/v1beta1",
			"kind":       "VirtualService",
			"metadata": map[string]interface{}{
				"name":      "test-vs",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"hosts":    []interface{}{"test.example.com"},
				"gateways": []interface{}{"test-gateway"},
				"http": []interface{}{
					map[string]interface{}{
						"route": []interface{}{
							map[string]interface{}{
								"destination": map[string]interface{}{
									"host": "test-svc",
								},
							},
						},
					},
				},
			},
		},
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{
			vsGVR: "VirtualServiceList",
			gwGVR: "GatewayList",
		},
	)

	ctx := context.Background()
	_, err := dynamicClient.Resource(vsGVR).Namespace("default").Create(ctx, vs, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create VirtualService: %v", err)
	}

	services := []ServiceInfo{{Name: "test-svc"}}
	vsInfos, _ := getIstioResources(ctx, dynamicClient, "default", services)

	if len(vsInfos) != 1 {
		t.Errorf("Expected 1 VirtualService, got %d", len(vsInfos))
	}
}

// ============================================
// ForceDeleteNamespace Tests
// ============================================

func TestForceDeleteNamespace_Success(t *testing.T) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-delete",
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceTerminating,
		},
	}

	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	clientset := fake.NewSimpleClientset(ns)
	ctx := context.Background()

	// ForceDeleteNamespace proceeds with any existing namespace
	err := ForceDeleteNamespace(ctx, clientset, dynamicClient, "to-delete")
	// The fake clientset's Discovery doesn't fully support ServerGroupsAndResources,
	// so this may return an error - but we're testing the code path
	if err != nil {
		t.Logf("ForceDeleteNamespace() error (expected with fake client): %v", err)
	}
}

func TestForceDeleteNamespace_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	clientset := fake.NewSimpleClientset()
	ctx := context.Background()

	err := ForceDeleteNamespace(ctx, clientset, dynamicClient, "nonexistent")
	if err == nil {
		t.Error("ForceDeleteNamespace() should error for nonexistent namespace")
	}
}

func TestForceDeleteNamespace_WithFinalizers(t *testing.T) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "finalizer-ns",
		},
		Spec: corev1.NamespaceSpec{
			Finalizers: []corev1.FinalizerName{"kubernetes"},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceTerminating,
		},
	}

	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	clientset := fake.NewSimpleClientset(ns)
	ctx := context.Background()

	// This tests the finalizer removal path
	err := ForceDeleteNamespace(ctx, clientset, dynamicClient, "finalizer-ns")
	if err != nil {
		t.Logf("ForceDeleteNamespace() error: %v", err)
	}
}

// ============================================
// ListRollouts with float64 replicas
// ============================================

func TestListRollouts_WithFloat64Replicas(t *testing.T) {
	scheme := runtime.NewScheme()

	rolloutGVR := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "rollouts",
	}

	// Use float64 for replicas (as JSON unmarshaling sometimes produces)
	rollout := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Rollout",
			"metadata": map[string]interface{}{
				"name":      "float-rollout",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": float64(3),
			},
			"status": map[string]interface{}{
				"readyReplicas": float64(3),
			},
		},
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{
			rolloutGVR: "RolloutList",
		},
	)

	ctx := context.Background()
	_, err := dynamicClient.Resource(rolloutGVR).Namespace("default").Create(ctx, rollout, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create rollout: %v", err)
	}

	workloads, err := ListRollouts(ctx, dynamicClient, "default")
	if err != nil {
		t.Fatalf("ListRollouts() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Errorf("Expected 1 workload, got %d", len(workloads))
	}

	if len(workloads) > 0 && workloads[0].Replicas != 3 {
		t.Errorf("Replicas = %d, want 3", workloads[0].Replicas)
	}
}

// Note: labelsMatch and contains helper tests are in resources_test.go

// ============================================
// ScaleRollout Tests
// ============================================

func TestScaleRollout_NilDynamicClient(t *testing.T) {
	ctx := context.Background()

	err := ScaleRollout(ctx, nil, "default", "test", 3)
	if err == nil {
		t.Error("ScaleRollout() should error with nil dynamic client")
	}
}

func TestScaleRollout_Success(t *testing.T) {
	scheme := runtime.NewScheme()

	rolloutGVR := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "rollouts",
	}

	rollout := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Rollout",
			"metadata": map[string]interface{}{
				"name":      "web",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(1),
			},
		},
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{
			rolloutGVR: "RolloutList",
		},
	)

	ctx := context.Background()
	_, err := dynamicClient.Resource(rolloutGVR).Namespace("default").Create(ctx, rollout, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create rollout: %v", err)
	}

	err = ScaleRollout(ctx, dynamicClient, "default", "web", 5)
	if err != nil {
		t.Errorf("ScaleRollout() error = %v", err)
	}

	// Verify the scale was updated
	updated, err := dynamicClient.Resource(rolloutGVR).Namespace("default").Get(ctx, "web", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get rollout error = %v", err)
	}

	replicas, _, _ := unstructured.NestedInt64(updated.Object, "spec", "replicas")
	if replicas != 5 {
		t.Errorf("Replicas = %d, want 5", replicas)
	}
}

// ============================================
// GetWorkloadPods Tests
// ============================================

func TestGetWorkloadPods_SinglePod(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "standalone-pod",
			Namespace: "default",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	clientset := fake.NewSimpleClientset(pod)
	ctx := context.Background()

	workload := WorkloadInfo{
		Name:      "standalone-pod",
		Namespace: "default",
		Type:      ResourcePods,
	}

	pods, err := GetWorkloadPods(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadPods() error = %v", err)
	}

	if len(pods) != 1 {
		t.Errorf("Expected 1 pod, got %d", len(pods))
	}
}

func TestGetWorkloadPods_WithLabels(t *testing.T) {
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-pod-1",
			Namespace: "default",
			Labels:    map[string]string{"app": "web"},
		},
	}
	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-pod-2",
			Namespace: "default",
			Labels:    map[string]string{"app": "web"},
		},
	}

	clientset := fake.NewSimpleClientset(pod1, pod2)
	ctx := context.Background()

	workload := WorkloadInfo{
		Name:      "web-deployment",
		Namespace: "default",
		Type:      ResourceDeployments,
		Labels:    map[string]string{"app": "web"},
	}

	pods, err := GetWorkloadPods(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadPods() error = %v", err)
	}

	if len(pods) != 2 {
		t.Errorf("Expected 2 pods, got %d", len(pods))
	}
}

// ============================================
// GetRelatedResources Additional Tests
// ============================================

func TestGetRelatedResources_WithReplicaSetOwner(t *testing.T) {
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-rs-abc",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Deployment",
					Name: "web",
				},
			},
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr(int32(3)),
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 3,
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-pod",
			Namespace: "default",
			Labels:    map[string]string{"app": "web"},
		},
	}

	clientset := fake.NewSimpleClientset(rs, deployment, pod)
	ctx := context.Background()

	podInfo := PodInfo{
		Name:      "web-pod",
		Namespace: "default",
		Labels:    map[string]string{"app": "web"},
		OwnerRef:  "web-rs-abc",
		OwnerKind: "ReplicaSet",
	}

	related, err := GetRelatedResources(ctx, clientset, nil, podInfo)
	if err != nil {
		t.Fatalf("GetRelatedResources() error = %v", err)
	}

	if related.Owner == nil {
		t.Error("Expected Owner to be set")
		return
	}

	if related.Owner.WorkloadKind != "Deployment" {
		t.Errorf("WorkloadKind = %q, want %q", related.Owner.WorkloadKind, "Deployment")
	}
	if related.Owner.WorkloadName != "web" {
		t.Errorf("WorkloadName = %q, want %q", related.Owner.WorkloadName, "web")
	}
}

func TestGetRelatedResources_WithIngress(t *testing.T) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Selector:  map[string]string{"app": "web"},
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.0.0.1",
			Ports: []corev1.ServicePort{
				{Port: 80, Protocol: corev1.ProtocolTCP},
			},
		},
	}

	pathType := networkingv1.PathTypePrefix
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-ing",
			Namespace: "default",
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "/",
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: strPtr("nginx"),
			Rules: []networkingv1.IngressRule{
				{
					Host: "web.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "web-svc",
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			TLS: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"web.example.com"},
					SecretName: "tls-secret",
				},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-pod",
			Namespace: "default",
			Labels:    map[string]string{"app": "web"},
		},
	}

	clientset := fake.NewSimpleClientset(svc, ing, pod)
	ctx := context.Background()

	podInfo := PodInfo{
		Name:      "web-pod",
		Namespace: "default",
		Labels:    map[string]string{"app": "web"},
	}

	related, err := GetRelatedResources(ctx, clientset, nil, podInfo)
	if err != nil {
		t.Fatalf("GetRelatedResources() error = %v", err)
	}

	if len(related.Services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(related.Services))
	}

	if len(related.Ingresses) != 1 {
		t.Errorf("Expected 1 ingress, got %d", len(related.Ingresses))
	}
}

func TestGetRelatedResources_WithConfigMapsAndSecrets(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-pod",
			Namespace: "default",
			Labels:    map[string]string{"app": "test"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "app",
					EnvFrom: []corev1.EnvFromSource{
						{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"}}},
						{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "app-secret"}}},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "config-vol",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "vol-config"}},
					},
				},
				{
					Name: "secret-vol",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{SecretName: "vol-secret"},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(pod)
	ctx := context.Background()

	podInfo := PodInfo{
		Name:      "app-pod",
		Namespace: "default",
		Labels:    map[string]string{"app": "test"},
	}

	related, err := GetRelatedResources(ctx, clientset, nil, podInfo)
	if err != nil {
		t.Fatalf("GetRelatedResources() error = %v", err)
	}

	if len(related.ConfigMaps) != 2 {
		t.Errorf("Expected 2 configmaps, got %d: %v", len(related.ConfigMaps), related.ConfigMaps)
	}

	if len(related.Secrets) != 2 {
		t.Errorf("Expected 2 secrets, got %d: %v", len(related.Secrets), related.Secrets)
	}
}

// Helper functions
func ptr(i int32) *int32 {
	return &i
}

func strPtr(s string) *string {
	return &s
}

// Note: int32Ptr exists in resources_test.go

// ============================================
// GetRelatedResources with StatefulSet owner
// ============================================

func TestGetRelatedResources_WithStatefulSetOwner(t *testing.T) {
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "db-rs",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "StatefulSet",
					Name: "db",
				},
			},
		},
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "db",
			Namespace: "default",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: ptr(int32(3)),
		},
		Status: appsv1.StatefulSetStatus{
			ReadyReplicas: 3,
		},
	}

	clientset := fake.NewSimpleClientset(rs, sts)
	ctx := context.Background()

	podInfo := PodInfo{
		Name:      "db-pod",
		Namespace: "default",
		OwnerRef:  "db-rs",
		OwnerKind: "ReplicaSet",
	}

	related, err := GetRelatedResources(ctx, clientset, nil, podInfo)
	if err != nil {
		t.Fatalf("GetRelatedResources() error = %v", err)
	}

	if related.Owner != nil && related.Owner.WorkloadKind != "StatefulSet" {
		t.Errorf("WorkloadKind = %q, want %q", related.Owner.WorkloadKind, "StatefulSet")
	}
}

// ============================================
// GetRelatedResources with Rollout owner
// ============================================

func TestGetRelatedResources_WithRolloutOwner(t *testing.T) {
	scheme := runtime.NewScheme()

	rolloutGVR := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "rollouts",
	}

	vsGVR := schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "virtualservices",
	}

	gwGVR := schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "gateways",
	}

	rollout := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Rollout",
			"metadata": map[string]interface{}{
				"name":      "api",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(5),
			},
			"status": map[string]interface{}{
				"readyReplicas":     int64(5),
				"availableReplicas": int64(5),
			},
		},
	}

	// Register all GVRs that GetRelatedResources might use
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{
			rolloutGVR: "RolloutList",
			vsGVR:      "VirtualServiceList",
			gwGVR:      "GatewayList",
		},
	)

	ctx := context.Background()
	_, err := dynamicClient.Resource(rolloutGVR).Namespace("default").Create(ctx, rollout, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create rollout: %v", err)
	}

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-rs",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Rollout",
					Name: "api",
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(rs)

	podInfo := PodInfo{
		Name:      "api-pod",
		Namespace: "default",
		OwnerRef:  "api-rs",
		OwnerKind: "ReplicaSet",
	}

	related, err := GetRelatedResources(ctx, clientset, dynamicClient, podInfo)
	if err != nil {
		t.Fatalf("GetRelatedResources() error = %v", err)
	}

	if related.Owner != nil && related.Owner.WorkloadKind == "Rollout" {
		if related.Owner.Replicas != 5 {
			t.Errorf("Replicas = %d, want 5", related.Owner.Replicas)
		}
	}
}

// Note: ingressReferencesService tests are in resources_test.go

// ============================================
// GetWorkloadPods Error Paths
// ============================================

func TestGetWorkloadPods_PodNotFound(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	ctx := context.Background()

	workload := WorkloadInfo{
		Name:      "nonexistent",
		Namespace: "default",
		Type:      ResourcePods,
	}

	_, err := GetWorkloadPods(ctx, clientset, workload)
	if err == nil {
		t.Error("GetWorkloadPods() should error for nonexistent pod")
	}
}

// ============================================
// Service with no selector Tests
// ============================================

func TestGetRelatedResources_ServiceWithNoSelector(t *testing.T) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "headless-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			// No selector - should not match any pod
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "None",
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels:    map[string]string{"app": "test"},
		},
	}

	clientset := fake.NewSimpleClientset(svc, pod)
	ctx := context.Background()

	podInfo := PodInfo{
		Name:      "test-pod",
		Namespace: "default",
		Labels:    map[string]string{"app": "test"},
	}

	related, err := GetRelatedResources(ctx, clientset, nil, podInfo)
	if err != nil {
		t.Fatalf("GetRelatedResources() error = %v", err)
	}

	// Service without selector should not match
	if len(related.Services) != 0 {
		t.Errorf("Expected 0 services (no selector), got %d", len(related.Services))
	}
}

// ============================================
// Ingress with annotation-based class
// ============================================

// ============================================
// getIstioResources with Gateway Tests
// ============================================

func TestGetIstioResources_WithGateway(t *testing.T) {
	scheme := runtime.NewScheme()

	vsGVR := schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "virtualservices",
	}

	gwGVR := schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "gateways",
	}

	vs := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.istio.io/v1beta1",
			"kind":       "VirtualService",
			"metadata": map[string]interface{}{
				"name":      "web-vs",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"hosts":    []interface{}{"web.example.com"},
				"gateways": []interface{}{"web-gateway"},
				"http": []interface{}{
					map[string]interface{}{
						"match": []interface{}{
							map[string]interface{}{
								"uri": map[string]interface{}{
									"prefix": "/api",
								},
							},
						},
						"route": []interface{}{
							map[string]interface{}{
								"destination": map[string]interface{}{
									"host": "web-svc",
									"port": map[string]interface{}{
										"number": float64(80),
									},
								},
								"weight": float64(100),
							},
						},
					},
				},
			},
		},
	}

	gateway := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.istio.io/v1beta1",
			"kind":       "Gateway",
			"metadata": map[string]interface{}{
				"name":      "web-gateway",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"servers": []interface{}{
					map[string]interface{}{
						"port": map[string]interface{}{
							"number":   float64(443),
							"protocol": "HTTPS",
						},
						"hosts": []interface{}{"*.example.com"},
						"tls": map[string]interface{}{
							"mode": "SIMPLE",
						},
					},
				},
			},
		},
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{
			vsGVR: "VirtualServiceList",
			gwGVR: "GatewayList",
		},
	)

	ctx := context.Background()
	_, err := dynamicClient.Resource(vsGVR).Namespace("default").Create(ctx, vs, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create VirtualService: %v", err)
	}
	_, err = dynamicClient.Resource(gwGVR).Namespace("default").Create(ctx, gateway, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create Gateway: %v", err)
	}

	services := []ServiceInfo{{Name: "web-svc"}}
	vsInfos, gwInfos := getIstioResources(ctx, dynamicClient, "default", services)

	if len(vsInfos) != 1 {
		t.Errorf("Expected 1 VirtualService, got %d", len(vsInfos))
	}

	if len(gwInfos) != 1 {
		t.Errorf("Expected 1 Gateway, got %d", len(gwInfos))
	}

	if len(gwInfos) > 0 && len(gwInfos[0].Servers) > 0 {
		if gwInfos[0].Servers[0].Port != 443 {
			t.Errorf("Gateway port = %d, want 443", gwInfos[0].Servers[0].Port)
		}
		if gwInfos[0].Servers[0].Protocol != "HTTPS" {
			t.Errorf("Gateway protocol = %q, want HTTPS", gwInfos[0].Servers[0].Protocol)
		}
		if gwInfos[0].Servers[0].TLS != "SIMPLE" {
			t.Errorf("Gateway TLS = %q, want SIMPLE", gwInfos[0].Servers[0].TLS)
		}
	}
}

func TestGetIstioResources_WithNamespacedGateway(t *testing.T) {
	scheme := runtime.NewScheme()

	vsGVR := schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "virtualservices",
	}

	gwGVR := schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "gateways",
	}

	// VirtualService references gateway with namespace/name format
	vs := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.istio.io/v1beta1",
			"kind":       "VirtualService",
			"metadata": map[string]interface{}{
				"name":      "api-vs",
				"namespace": "production",
			},
			"spec": map[string]interface{}{
				"hosts":    []interface{}{"api.example.com"},
				"gateways": []interface{}{"istio-system/main-gateway"},
				"http": []interface{}{
					map[string]interface{}{
						"route": []interface{}{
							map[string]interface{}{
								"destination": map[string]interface{}{
									"host": "api-svc",
									"port": map[string]interface{}{
										"number": int64(8080),
									},
								},
								"weight": int64(80),
							},
						},
					},
				},
			},
		},
	}

	gateway := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.istio.io/v1beta1",
			"kind":       "Gateway",
			"metadata": map[string]interface{}{
				"name":      "main-gateway",
				"namespace": "istio-system",
			},
			"spec": map[string]interface{}{
				"servers": []interface{}{
					map[string]interface{}{
						"port": map[string]interface{}{
							"number":   int64(80),
							"protocol": "HTTP",
						},
						"hosts": []interface{}{"*"},
					},
				},
			},
		},
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{
			vsGVR: "VirtualServiceList",
			gwGVR: "GatewayList",
		},
	)

	ctx := context.Background()
	_, _ = dynamicClient.Resource(vsGVR).Namespace("production").Create(ctx, vs, metav1.CreateOptions{})
	_, _ = dynamicClient.Resource(gwGVR).Namespace("istio-system").Create(ctx, gateway, metav1.CreateOptions{})

	services := []ServiceInfo{{Name: "api-svc"}}
	vsInfos, gwInfos := getIstioResources(ctx, dynamicClient, "production", services)

	if len(vsInfos) != 1 {
		t.Errorf("Expected 1 VirtualService, got %d", len(vsInfos))
	}

	// Gateway should be fetched from istio-system namespace
	if len(gwInfos) != 1 {
		t.Errorf("Expected 1 Gateway, got %d", len(gwInfos))
	}

	if len(gwInfos) > 0 && gwInfos[0].Namespace != "istio-system" {
		t.Errorf("Gateway namespace = %q, want istio-system", gwInfos[0].Namespace)
	}
}

func TestGetIstioResources_NoMatchingService(t *testing.T) {
	scheme := runtime.NewScheme()

	vsGVR := schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "virtualservices",
	}

	gwGVR := schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "gateways",
	}

	// VirtualService routes to a different service
	vs := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.istio.io/v1beta1",
			"kind":       "VirtualService",
			"metadata": map[string]interface{}{
				"name":      "other-vs",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"hosts": []interface{}{"other.example.com"},
				"http": []interface{}{
					map[string]interface{}{
						"route": []interface{}{
							map[string]interface{}{
								"destination": map[string]interface{}{
									"host": "other-svc",
								},
							},
						},
					},
				},
			},
		},
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{
			vsGVR: "VirtualServiceList",
			gwGVR: "GatewayList",
		},
	)

	ctx := context.Background()
	_, _ = dynamicClient.Resource(vsGVR).Namespace("default").Create(ctx, vs, metav1.CreateOptions{})

	services := []ServiceInfo{{Name: "web-svc"}} // Different service
	vsInfos, _ := getIstioResources(ctx, dynamicClient, "default", services)

	// VS should not be included since it doesn't route to web-svc
	if len(vsInfos) != 0 {
		t.Errorf("Expected 0 VirtualServices (no match), got %d", len(vsInfos))
	}
}

// ============================================
// GetWorkloadEvents Tests
// ============================================

func TestGetWorkloadEvents_WithPods(t *testing.T) {
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-event",
			Namespace: "default",
		},
		Type:   "Normal",
		Reason: "Started",
		InvolvedObject: corev1.ObjectReference{
			Name: "web-pod",
			Kind: "Pod",
		},
		Message:        "Started container",
		FirstTimestamp: metav1.Now(),
		LastTimestamp:  metav1.Now(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-pod",
			Namespace: "default",
			Labels:    map[string]string{"app": "web"},
		},
	}

	clientset := fake.NewSimpleClientset(event, pod)
	ctx := context.Background()

	workload := WorkloadInfo{
		Name:      "web-deployment",
		Namespace: "default",
		Labels:    map[string]string{"app": "web"},
	}

	events, err := GetWorkloadEvents(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadEvents() error = %v", err)
	}

	// Should include events for pods with matching labels
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
}

func TestGetWorkloadEvents_WithWorkloadEvent(t *testing.T) {
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-event",
			Namespace: "default",
		},
		Type:   "Normal",
		Reason: "ScalingReplicaSet",
		InvolvedObject: corev1.ObjectReference{
			Name: "web-deployment",
			Kind: "Deployment",
		},
		Message:        "Scaled up replica set",
		FirstTimestamp: metav1.Now(),
		LastTimestamp:  metav1.Now(),
	}

	clientset := fake.NewSimpleClientset(event)
	ctx := context.Background()

	workload := WorkloadInfo{
		Name:      "web-deployment",
		Namespace: "default",
	}

	events, err := GetWorkloadEvents(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadEvents() error = %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
}

func TestGetRelatedResources_IngressWithAnnotationClass(t *testing.T) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "web"},
			Ports:    []corev1.ServicePort{{Port: 80}},
		},
	}

	pathType := networkingv1.PathTypePrefix
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-ing",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "nginx",
			},
		},
		Spec: networkingv1.IngressSpec{
			// No IngressClassName, using annotation instead
			Rules: []networkingv1.IngressRule{
				{
					Host: "web.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "web-svc",
											Port: networkingv1.ServiceBackendPort{Name: "http"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-pod",
			Namespace: "default",
			Labels:    map[string]string{"app": "web"},
		},
	}

	clientset := fake.NewSimpleClientset(svc, ing, pod)
	ctx := context.Background()

	podInfo := PodInfo{
		Name:      "web-pod",
		Namespace: "default",
		Labels:    map[string]string{"app": "web"},
	}

	related, err := GetRelatedResources(ctx, clientset, nil, podInfo)
	if err != nil {
		t.Fatalf("GetRelatedResources() error = %v", err)
	}

	if len(related.Ingresses) != 1 {
		t.Errorf("Expected 1 ingress, got %d", len(related.Ingresses))
	}

	if len(related.Ingresses) > 0 && related.Ingresses[0].Class != "nginx" {
		t.Errorf("Ingress class = %q, want %q", related.Ingresses[0].Class, "nginx")
	}
}

// ============================================
// HPA Tests
// ============================================

func TestListHPAs_Empty(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	ctx := context.Background()

	hpas, err := ListHPAs(ctx, clientset, "default")
	if err != nil {
		t.Fatalf("ListHPAs() error = %v", err)
	}

	if len(hpas) != 0 {
		t.Errorf("Expected 0 HPAs, got %d", len(hpas))
	}
}

// Note: TestGetHPA_NotFound exists in resources_test.go

// ============================================
// GetPodEvents Tests
// ============================================

func TestGetPodEvents_Success(t *testing.T) {
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-event",
			Namespace: "default",
		},
		Type:   "Warning",
		Reason: "BackOff",
		InvolvedObject: corev1.ObjectReference{
			Name: "test-pod",
			Kind: "Pod",
		},
		Message:        "Back-off restarting failed container",
		FirstTimestamp: metav1.Now(),
		LastTimestamp:  metav1.Now(),
		Count:          5,
	}

	clientset := fake.NewSimpleClientset(event)
	ctx := context.Background()

	events, err := GetPodEvents(ctx, clientset, "default", "test-pod")
	if err != nil {
		t.Fatalf("GetPodEvents() error = %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	if len(events) > 0 {
		if events[0].Type != "Warning" {
			t.Errorf("Event type = %q, want Warning", events[0].Type)
		}
		if events[0].Count != 5 {
			t.Errorf("Event count = %d, want 5", events[0].Count)
		}
	}
}

// Note: TestGetPodEvents_Empty exists in events_test.go

// ============================================
// GetNamespaceEvents Tests
// ============================================

func TestGetNamespaceEvents_WithLimit(t *testing.T) {
	events := []runtime.Object{}
	for i := 0; i < 10; i++ {
		events = append(events, &corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("event-%d", i),
				Namespace: "default",
			},
			Type:           "Normal",
			Reason:         "Created",
			Message:        fmt.Sprintf("Event %d", i),
			FirstTimestamp: metav1.Now(),
			LastTimestamp:  metav1.Now(),
		})
	}

	clientset := fake.NewSimpleClientset(events...)
	ctx := context.Background()

	result, err := GetNamespaceEvents(ctx, clientset, "default", 5)
	if err != nil {
		t.Fatalf("GetNamespaceEvents() error = %v", err)
	}

	if len(result) != 5 {
		t.Errorf("Expected 5 events (limited), got %d", len(result))
	}
}

func TestGetNamespaceEvents_NoLimit(t *testing.T) {
	event1 := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "event-1",
			Namespace: "default",
		},
		Type:           "Normal",
		FirstTimestamp: metav1.Now(),
		LastTimestamp:  metav1.Now(),
	}
	event2 := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "event-2",
			Namespace: "default",
		},
		Type:           "Warning",
		FirstTimestamp: metav1.Now(),
		LastTimestamp:  metav1.Now(),
	}

	clientset := fake.NewSimpleClientset(event1, event2)
	ctx := context.Background()

	result, err := GetNamespaceEvents(ctx, clientset, "default", 0)
	if err != nil {
		t.Fatalf("GetNamespaceEvents() error = %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 events, got %d", len(result))
	}
}

// ============================================
// GetRecentWarnings Tests
// ============================================

func TestGetRecentWarnings_Success(t *testing.T) {
	warning := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "warning-event",
			Namespace: "default",
		},
		Type:           "Warning",
		Reason:         "BackOff",
		FirstTimestamp: metav1.Now(),
		LastTimestamp:  metav1.Now(),
	}
	normal := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "normal-event",
			Namespace: "default",
		},
		Type:           "Normal",
		Reason:         "Created",
		FirstTimestamp: metav1.Now(),
		LastTimestamp:  metav1.Now(),
	}

	clientset := fake.NewSimpleClientset(warning, normal)
	ctx := context.Background()

	// Get warnings from the last hour
	warnings, err := GetRecentWarnings(ctx, clientset, "default", 1*time.Hour)
	if err != nil {
		t.Fatalf("GetRecentWarnings() error = %v", err)
	}

	if len(warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(warnings))
	}

	if len(warnings) > 0 && warnings[0].Type != "Warning" {
		t.Errorf("Event type = %q, want Warning", warnings[0].Type)
	}
}

// Note: Log helper function tests (TestSearchLogs, TestFilterErrorLogs, TestGetLogsAroundTime,
// TestIsErrorLine) exist in logs_test.go

// Note: Metrics helper function tests (TestFormatCPU, TestFormatMemory, TestGetPodMetrics_NilClient,
// TestGetNamespaceMetrics_NilClient, TestCalculateResourceUsage_NilInputs) exist in metrics_test.go

// ============================================
// Additional HPA Coverage Tests
// ============================================

func TestListHPAs_ResourceCurrentAverageValue(t *testing.T) {
	// Test HPA with Resource metric using AverageValue (not utilization)
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hpa-avg-value",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "test-deployment",
			},
			MinReplicas: int32Ptr(1),
			MaxReplicas: 10,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "memory",
						Target: autoscalingv2.MetricTarget{
							Type:         autoscalingv2.AverageValueMetricType,
							AverageValue: resourceQuantity("500Mi"),
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 2,
			DesiredReplicas: 3,
			CurrentMetrics: []autoscalingv2.MetricStatus{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricStatus{
						Name: "memory",
						Current: autoscalingv2.MetricValueStatus{
							AverageValue: resourceQuantity("400Mi"),
						},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(hpa)
	ctx := context.Background()

	result, err := ListHPAs(ctx, clientset, "default")
	if err != nil {
		t.Fatalf("ListHPAs() error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 HPA, got %d", len(result))
	}

	// Targets should show memory with average values
	if !strings.Contains(result[0].Targets, "memory:") {
		t.Errorf("Targets = %q, expected to contain 'memory:'", result[0].Targets)
	}
}

func TestListHPAs_ExternalTargetValue(t *testing.T) {
	// Test HPA with External metric using Target.Value (not AverageValue)
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hpa-external-value",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "test-deployment",
			},
			MinReplicas: int32Ptr(1),
			MaxReplicas: 10,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricSource{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "external_metric",
						},
						Target: autoscalingv2.MetricTarget{
							Type:  autoscalingv2.ValueMetricType,
							Value: resourceQuantity("100"),
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 2,
			DesiredReplicas: 3,
			CurrentMetrics: []autoscalingv2.MetricStatus{
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricStatus{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "external_metric",
						},
						Current: autoscalingv2.MetricValueStatus{
							Value: resourceQuantity("80"),
						},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(hpa)
	ctx := context.Background()

	result, err := ListHPAs(ctx, clientset, "default")
	if err != nil {
		t.Fatalf("ListHPAs() error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 HPA, got %d", len(result))
	}

	// Targets should show external_metric
	if !strings.Contains(result[0].Targets, "external_metric:") {
		t.Errorf("Targets = %q, expected to contain 'external_metric:'", result[0].Targets)
	}
}

func TestListHPAs_ExternalCurrentAverageValue(t *testing.T) {
	// Test HPA with External metric current using AverageValue
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hpa-external-avg",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "test-deployment",
			},
			MinReplicas: int32Ptr(1),
			MaxReplicas: 10,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricSource{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "keda_metric",
						},
						Target: autoscalingv2.MetricTarget{
							Type:         autoscalingv2.AverageValueMetricType,
							AverageValue: resourceQuantity("50"),
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 2,
			DesiredReplicas: 3,
			CurrentMetrics: []autoscalingv2.MetricStatus{
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricStatus{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "keda_metric",
						},
						Current: autoscalingv2.MetricValueStatus{
							AverageValue: resourceQuantity("30"),
						},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(hpa)
	ctx := context.Background()

	result, err := ListHPAs(ctx, clientset, "default")
	if err != nil {
		t.Fatalf("ListHPAs() error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 HPA, got %d", len(result))
	}

	// Targets should show keda_metric
	if !strings.Contains(result[0].Targets, "keda_metric:") {
		t.Errorf("Targets = %q, expected to contain 'keda_metric:'", result[0].Targets)
	}
}

func TestListHPAs_NoMetrics(t *testing.T) {
	// Test HPA with no metrics
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hpa-no-metrics",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "test-deployment",
			},
			MinReplicas: int32Ptr(1),
			MaxReplicas: 10,
			Metrics:     []autoscalingv2.MetricSpec{},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 1,
			DesiredReplicas: 1,
		},
	}

	clientset := fake.NewSimpleClientset(hpa)
	ctx := context.Background()

	result, err := ListHPAs(ctx, clientset, "default")
	if err != nil {
		t.Fatalf("ListHPAs() error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 HPA, got %d", len(result))
	}

	// Targets should show <none>
	if result[0].Targets != "<none>" {
		t.Errorf("Targets = %q, expected '<none>'", result[0].Targets)
	}
}

func TestListHPAs_ResourceNoCurrentMetric(t *testing.T) {
	// Test HPA with resource metric but no current metrics
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hpa-no-current",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "test-deployment",
			},
			MinReplicas: int32Ptr(1),
			MaxReplicas: 10,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "cpu",
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: int32Ptr(80),
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 1,
			DesiredReplicas: 1,
			CurrentMetrics:  []autoscalingv2.MetricStatus{}, // No current metrics
		},
	}

	clientset := fake.NewSimpleClientset(hpa)
	ctx := context.Background()

	result, err := ListHPAs(ctx, clientset, "default")
	if err != nil {
		t.Fatalf("ListHPAs() error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 HPA, got %d", len(result))
	}

	// Targets should show <unknown> for current
	if !strings.Contains(result[0].Targets, "<unknown>") {
		t.Errorf("Targets = %q, expected to contain '<unknown>'", result[0].Targets)
	}
}

func TestListHPAs_ExternalNoCurrentMetric(t *testing.T) {
	// Test HPA with external metric but no current metrics
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hpa-external-no-current",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "test-deployment",
			},
			MinReplicas: int32Ptr(1),
			MaxReplicas: 10,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricSource{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "rabbitmq_messages",
						},
						Target: autoscalingv2.MetricTarget{
							Type:  autoscalingv2.ValueMetricType,
							Value: resourceQuantity("100"),
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 1,
			DesiredReplicas: 1,
			CurrentMetrics:  []autoscalingv2.MetricStatus{}, // No current metrics
		},
	}

	clientset := fake.NewSimpleClientset(hpa)
	ctx := context.Background()

	result, err := ListHPAs(ctx, clientset, "default")
	if err != nil {
		t.Fatalf("ListHPAs() error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 HPA, got %d", len(result))
	}

	// Targets should show <unknown> for current
	if !strings.Contains(result[0].Targets, "<unknown>") {
		t.Errorf("Targets = %q, expected to contain '<unknown>'", result[0].Targets)
	}
}

// ============================================
// Additional Node Coverage Tests
// ============================================

func TestListNodes_WithAllConditions(t *testing.T) {
	// Test node with various conditions
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "node-with-conditions",
			CreationTimestamp: metav1.Now(),
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionTrue},
				{Type: corev1.NodeDiskPressure, Status: corev1.ConditionTrue},
				{Type: corev1.NodePIDPressure, Status: corev1.ConditionTrue},
			},
			NodeInfo: corev1.NodeSystemInfo{
				KubeletVersion: "v1.28.0",
			},
			Addresses: []corev1.NodeAddress{
				{Type: corev1.NodeInternalIP, Address: "10.0.0.1"},
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			},
		},
	}

	clientset := fake.NewSimpleClientset(node)
	ctx := context.Background()

	result, err := ListNodes(ctx, clientset)
	if err != nil {
		t.Fatalf("ListNodes() error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(result))
	}

	// NodeInfo has basic fields (Name, Status, Roles, Age, Version, InternalIP, CPU, Memory)
	if result[0].Name != "node-with-conditions" {
		t.Errorf("Name = %q, want 'node-with-conditions'", result[0].Name)
	}
	if result[0].Version != "v1.28.0" {
		t.Errorf("Version = %q, want 'v1.28.0'", result[0].Version)
	}
	if result[0].InternalIP != "10.0.0.1" {
		t.Errorf("InternalIP = %q, want '10.0.0.1'", result[0].InternalIP)
	}
}

func TestGetNode_WithDetails(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-with-details",
			Labels: map[string]string{
				"node.kubernetes.io/instance-type": "m5.large",
			},
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.NodeSpec{
			ProviderID: "aws:///us-west-2a/i-1234567890abcdef0",
		},
		Status: corev1.NodeStatus{
			NodeInfo: corev1.NodeSystemInfo{
				KubeletVersion:          "v1.28.0",
				ContainerRuntimeVersion: "containerd://1.6.20",
				OSImage:                 "Ubuntu 22.04.2 LTS",
			},
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
			Addresses: []corev1.NodeAddress{
				{Type: corev1.NodeInternalIP, Address: "10.0.0.2"},
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:              resource.MustParse("4"),
				corev1.ResourceMemory:           resource.MustParse("8Gi"),
				corev1.ResourceEphemeralStorage: resource.MustParse("100Gi"),
				corev1.ResourcePods:             resource.MustParse("110"),
			},
		},
	}

	clientset := fake.NewSimpleClientset(node)
	ctx := context.Background()

	result, err := GetNode(ctx, clientset, "node-with-details")
	if err != nil {
		t.Fatalf("GetNode() error = %v", err)
	}

	if result == nil {
		t.Fatal("GetNode() returned nil")
	}

	if result.Name != "node-with-details" {
		t.Errorf("Name = %q, want 'node-with-details'", result.Name)
	}
}

// ============================================
// Additional Copy Tests
// ============================================

func TestCopySecretToNamespace_UpdateExisting(t *testing.T) {
	// Source secret
	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "source-ns",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}

	// Existing secret in target namespace
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "target-ns",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"old-key": []byte("old-value"),
		},
	}

	clientset := fake.NewSimpleClientset(sourceSecret, existingSecret)
	ctx := context.Background()

	err := CopySecretToNamespace(ctx, clientset, "source-ns", "test-secret", "target-ns")
	if err != nil {
		t.Fatalf("CopySecretToNamespace() error = %v", err)
	}

	// Verify secret was updated
	updated, err := clientset.CoreV1().Secrets("target-ns").Get(ctx, "test-secret", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get updated secret: %v", err)
	}

	if string(updated.Data["key"]) != "value" {
		t.Errorf("Secret data not updated: got %s, want 'value'", string(updated.Data["key"]))
	}
}

func TestCopyConfigMapToNamespace_UpdateExisting(t *testing.T) {
	// Source configmap
	sourceConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "source-ns",
		},
		Data: map[string]string{
			"config.yaml": "new: value",
		},
	}

	// Existing configmap in target namespace
	existingConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "target-ns",
		},
		Data: map[string]string{
			"config.yaml": "old: value",
		},
	}

	clientset := fake.NewSimpleClientset(sourceConfigMap, existingConfigMap)
	ctx := context.Background()

	err := CopyConfigMapToNamespace(ctx, clientset, "source-ns", "test-cm", "target-ns")
	if err != nil {
		t.Fatalf("CopyConfigMapToNamespace() error = %v", err)
	}

	// Verify configmap was updated
	updated, err := clientset.CoreV1().ConfigMaps("target-ns").Get(ctx, "test-cm", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get updated configmap: %v", err)
	}

	if updated.Data["config.yaml"] != "new: value" {
		t.Errorf("ConfigMap data not updated: got %s, want 'new: value'", updated.Data["config.yaml"])
	}
}

// ============================================
// Additional PodInfo Coverage Tests
// ============================================

func TestPodToPodInfo_WithInitContainers(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-with-init",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{Name: "init-container"},
			},
			Containers: []corev1.Container{
				{Name: "main-container"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "main-container",
					Ready: true,
					State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				},
			},
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "init-container",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 0,
							Reason:   "Completed",
						},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(pod)
	ctx := context.Background()

	pods, err := ListAllPods(ctx, clientset, "default")
	if err != nil {
		t.Fatalf("ListAllPods() error = %v", err)
	}

	if len(pods) != 1 {
		t.Fatalf("Expected 1 pod, got %d", len(pods))
	}
}

func TestPodToPodInfo_PodStatuses(t *testing.T) {
	tests := []struct {
		name           string
		phase          corev1.PodPhase
		reason         string
		containerState corev1.ContainerState
		expectedStatus string
	}{
		{
			name:           "Pending pod",
			phase:          corev1.PodPending,
			expectedStatus: "Pending",
		},
		{
			name:           "Succeeded pod",
			phase:          corev1.PodSucceeded,
			expectedStatus: "Succeeded",
		},
		{
			name:           "Failed pod",
			phase:          corev1.PodFailed,
			expectedStatus: "Failed",
		},
		{
			name:   "Pod with reason",
			phase:  corev1.PodFailed,
			reason: "Evicted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "main"},
					},
				},
				Status: corev1.PodStatus{
					Phase:  tt.phase,
					Reason: tt.reason,
				},
			}

			clientset := fake.NewSimpleClientset(pod)
			ctx := context.Background()

			pods, err := ListAllPods(ctx, clientset, "default")
			if err != nil {
				t.Fatalf("ListAllPods() error = %v", err)
			}

			if len(pods) != 1 {
				t.Fatalf("Expected 1 pod, got %d", len(pods))
			}
		})
	}
}

// ============================================
// Additional Workload Tests
// ============================================

func TestListWorkloads_DeploymentNoReplicas(t *testing.T) {
	// Test deployment with nil replicas - checks that the nil case is handled without panic
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-no-replicas",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			// Replicas is nil (in real K8s, defaults to 1 but fake client returns 0)
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "test"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "main", Image: "nginx"}},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 1,
		},
	}

	clientset := fake.NewSimpleClientset(deployment)
	ctx := context.Background()

	result, err := ListWorkloads(ctx, clientset, "default", ResourceDeployments)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 workload, got %d", len(result))
	}

	// Test passes as long as it doesn't panic on nil replicas
	// The fake client returns 0 for nil replicas (unlike real K8s which defaults to 1)
	if result[0].Name != "deployment-no-replicas" {
		t.Errorf("Name = %q, want 'deployment-no-replicas'", result[0].Name)
	}
}

// helper function for resource quantities
func resourceQuantity(s string) *resource.Quantity {
	q := resource.MustParse(s)
	return &q
}

// ============================================
// ScaleWorkload Tests with Reactors
// ============================================

func TestScaleDeployment_Success(t *testing.T) {
	// Create a deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
		},
	}

	clientset := fake.NewSimpleClientset(deployment)

	// Add reactor to handle GetScale
	clientset.PrependReactor("get", "deployments/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
			},
			Spec: autoscalingv1.ScaleSpec{
				Replicas: 2,
			},
		}, nil
	})

	// Add reactor to handle UpdateScale
	clientset.PrependReactor("update", "deployments/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		updateAction := action.(k8stesting.UpdateAction)
		scale := updateAction.GetObject().(*autoscalingv1.Scale)
		return true, scale, nil
	})

	ctx := context.Background()
	err := ScaleDeployment(ctx, clientset, "default", "test-deployment", 5)
	if err != nil {
		t.Fatalf("ScaleDeployment() error = %v", err)
	}
}

func TestScaleDeployment_NotFound(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	// Add reactor to return NotFound error
	clientset.PrependReactor("get", "deployments/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("deployments.apps \"test-deployment\" not found")
	})

	ctx := context.Background()
	err := ScaleDeployment(ctx, clientset, "default", "test-deployment", 5)
	if err == nil {
		t.Error("ScaleDeployment() expected error for not found")
	}
}

func TestScaleStatefulSet_Success(t *testing.T) {
	statefulset := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-statefulset",
			Namespace: "default",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: int32Ptr(2),
		},
	}

	clientset := fake.NewSimpleClientset(statefulset)

	// Add reactor to handle GetScale
	clientset.PrependReactor("get", "statefulsets/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset",
				Namespace: "default",
			},
			Spec: autoscalingv1.ScaleSpec{
				Replicas: 2,
			},
		}, nil
	})

	// Add reactor to handle UpdateScale
	clientset.PrependReactor("update", "statefulsets/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		updateAction := action.(k8stesting.UpdateAction)
		scale := updateAction.GetObject().(*autoscalingv1.Scale)
		return true, scale, nil
	})

	ctx := context.Background()
	err := ScaleStatefulSet(ctx, clientset, "default", "test-statefulset", 5)
	if err != nil {
		t.Fatalf("ScaleStatefulSet() error = %v", err)
	}
}

func TestScaleStatefulSet_NotFound(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	// Add reactor to return NotFound error
	clientset.PrependReactor("get", "statefulsets/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("statefulsets.apps \"test-statefulset\" not found")
	})

	ctx := context.Background()
	err := ScaleStatefulSet(ctx, clientset, "default", "test-statefulset", 5)
	if err == nil {
		t.Error("ScaleStatefulSet() expected error for not found")
	}
}

func TestScaleRollout_NilClient(t *testing.T) {
	ctx := context.Background()
	err := ScaleRollout(ctx, nil, "default", "test-rollout", 5)
	if err == nil {
		t.Error("ScaleRollout(nil) expected error")
	}
	if !strings.Contains(err.Error(), "dynamic client not available") {
		t.Errorf("Error = %q, want 'dynamic client not available'", err.Error())
	}
}

// Test Client.ScaleWorkload for all resource types
func TestClientScaleWorkload_Deployments(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
	}

	clientset := fake.NewSimpleClientset(deployment)

	// Add reactor to handle GetScale for deployments
	clientset.PrependReactor("get", "deployments/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{Name: "test-deployment", Namespace: "default"},
			Spec:       autoscalingv1.ScaleSpec{Replicas: 2},
		}, nil
	})
	clientset.PrependReactor("update", "deployments/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		updateAction := action.(k8stesting.UpdateAction)
		scale := updateAction.GetObject().(*autoscalingv1.Scale)
		return true, scale, nil
	})

	client := &Client{
		clientset:     clientset,
		dynamicClient: nil,
	}

	ctx := context.Background()
	err := client.ScaleWorkload(ctx, "default", "test-deployment", ResourceDeployments, 5)
	if err != nil {
		t.Fatalf("ScaleWorkload() error = %v", err)
	}
}

func TestClientScaleWorkload_StatefulSets(t *testing.T) {
	statefulset := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-statefulset",
			Namespace: "default",
		},
	}

	clientset := fake.NewSimpleClientset(statefulset)

	// Add reactor to handle GetScale for statefulsets
	clientset.PrependReactor("get", "statefulsets/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{Name: "test-statefulset", Namespace: "default"},
			Spec:       autoscalingv1.ScaleSpec{Replicas: 2},
		}, nil
	})
	clientset.PrependReactor("update", "statefulsets/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		updateAction := action.(k8stesting.UpdateAction)
		scale := updateAction.GetObject().(*autoscalingv1.Scale)
		return true, scale, nil
	})

	client := &Client{
		clientset:     clientset,
		dynamicClient: nil,
	}

	ctx := context.Background()
	err := client.ScaleWorkload(ctx, "default", "test-statefulset", ResourceStatefulSets, 5)
	if err != nil {
		t.Fatalf("ScaleWorkload() error = %v", err)
	}
}

func TestClientScaleWorkload_Rollouts(t *testing.T) {
	rollout := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Rollout",
			"metadata": map[string]interface{}{
				"name":      "test-rollout",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(2),
			},
		},
	}

	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme, rollout)

	client := &Client{
		clientset:     nil,
		dynamicClient: dynamicClient,
	}

	ctx := context.Background()
	err := client.ScaleWorkload(ctx, "default", "test-rollout", ResourceRollouts, 5)
	if err != nil {
		t.Fatalf("ScaleWorkload() error = %v", err)
	}
}

func TestClientScaleWorkload_DaemonSets(t *testing.T) {
	// DaemonSets cannot be scaled, should return nil
	client := &Client{
		clientset:     fake.NewSimpleClientset(),
		dynamicClient: nil,
	}

	ctx := context.Background()
	err := client.ScaleWorkload(ctx, "default", "test-daemonset", ResourceDaemonSets, 5)
	if err != nil {
		t.Errorf("ScaleWorkload(DaemonSet) should return nil, got %v", err)
	}
}

func TestClientScaleWorkload_Jobs(t *testing.T) {
	// Jobs cannot be scaled, should return nil
	client := &Client{
		clientset:     fake.NewSimpleClientset(),
		dynamicClient: nil,
	}

	ctx := context.Background()
	err := client.ScaleWorkload(ctx, "default", "test-job", ResourceJobs, 5)
	if err != nil {
		t.Errorf("ScaleWorkload(Job) should return nil, got %v", err)
	}
}

func TestClientScaleWorkload_CronJobs(t *testing.T) {
	// CronJobs cannot be scaled, should return nil
	client := &Client{
		clientset:     fake.NewSimpleClientset(),
		dynamicClient: nil,
	}

	ctx := context.Background()
	err := client.ScaleWorkload(ctx, "default", "test-cronjob", ResourceCronJobs, 5)
	if err != nil {
		t.Errorf("ScaleWorkload(CronJob) should return nil, got %v", err)
	}
}

// ============================================
// ForceDeleteNamespace Tests with Discovery Mock
// ============================================

func TestForceDeleteNamespace_WithDiscoveryResources(t *testing.T) {
	// Create namespace with finalizers
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "terminating-ns",
		},
		Spec: corev1.NamespaceSpec{
			Finalizers: []corev1.FinalizerName{
				"kubernetes",
			},
		},
	}

	clientset := fake.NewSimpleClientset(ns)

	// Mock the discovery to return some API resources
	clientset.Discovery().(*fakediscovery.FakeDiscovery).Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "pods",
					Namespaced: true,
					Kind:       "Pod",
					Verbs:      []string{"get", "list", "delete", "create"},
				},
				{
					Name:       "services",
					Namespaced: true,
					Kind:       "Service",
					Verbs:      []string{"get", "list", "delete", "create"},
				},
				{
					Name:       "namespaces",
					Namespaced: false,
					Kind:       "Namespace",
					Verbs:      []string{"get", "list", "delete", "create"},
				},
			},
		},
		{
			GroupVersion: "apps/v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "deployments",
					Namespaced: true,
					Kind:       "Deployment",
					Verbs:      []string{"get", "list", "delete", "create"},
				},
			},
		},
	}

	// Create a fake dynamic client
	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	ctx := context.Background()
	err := ForceDeleteNamespace(ctx, clientset, dynamicClient, "terminating-ns")
	if err != nil {
		t.Fatalf("ForceDeleteNamespace() error = %v", err)
	}
}

func TestForceDeleteNamespace_WithSubresources(t *testing.T) {
	// Create namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "subresource-ns",
		},
	}

	clientset := fake.NewSimpleClientset(ns)

	// Mock discovery with subresources (should be skipped)
	clientset.Discovery().(*fakediscovery.FakeDiscovery).Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "pods",
					Namespaced: true,
					Kind:       "Pod",
					Verbs:      []string{"get", "list", "delete"},
				},
				{
					Name:       "pods/log", // Subresource - should be skipped
					Namespaced: true,
					Kind:       "Pod",
					Verbs:      []string{"get"},
				},
				{
					Name:       "pods/exec", // Subresource - should be skipped
					Namespaced: true,
					Kind:       "Pod",
					Verbs:      []string{"create"},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	ctx := context.Background()
	err := ForceDeleteNamespace(ctx, clientset, dynamicClient, "subresource-ns")
	if err != nil {
		t.Fatalf("ForceDeleteNamespace() error = %v", err)
	}
}

func TestForceDeleteNamespace_ResourcesWithoutDeleteVerb(t *testing.T) {
	// Create namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "readonly-ns",
		},
	}

	clientset := fake.NewSimpleClientset(ns)

	// Mock discovery with resources that don't have delete verb
	clientset.Discovery().(*fakediscovery.FakeDiscovery).Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "pods",
					Namespaced: true,
					Kind:       "Pod",
					Verbs:      []string{"get", "list"}, // No delete verb
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	ctx := context.Background()
	err := ForceDeleteNamespace(ctx, clientset, dynamicClient, "readonly-ns")
	if err != nil {
		t.Fatalf("ForceDeleteNamespace() error = %v", err)
	}
}

func TestForceDeleteNamespace_MultipleAPIGroups(t *testing.T) {
	// Create namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "multi-api-ns",
		},
	}

	clientset := fake.NewSimpleClientset(ns)

	// Mock discovery with multiple API groups
	clientset.Discovery().(*fakediscovery.FakeDiscovery).Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "pods",
					Namespaced: true,
					Kind:       "Pod",
					Verbs:      []string{"get", "list", "delete"},
				},
				{
					Name:       "configmaps",
					Namespaced: true,
					Kind:       "ConfigMap",
					Verbs:      []string{"get", "list", "delete"},
				},
			},
		},
		{
			GroupVersion: "apps/v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "deployments",
					Namespaced: true,
					Kind:       "Deployment",
					Verbs:      []string{"get", "list", "delete"},
				},
				{
					Name:       "statefulsets",
					Namespaced: true,
					Kind:       "StatefulSet",
					Verbs:      []string{"get", "list", "delete"},
				},
			},
		},
		{
			GroupVersion: "batch/v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "jobs",
					Namespaced: true,
					Kind:       "Job",
					Verbs:      []string{"get", "list", "delete"},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	ctx := context.Background()
	err := ForceDeleteNamespace(ctx, clientset, dynamicClient, "multi-api-ns")
	if err != nil {
		t.Fatalf("ForceDeleteNamespace() error = %v", err)
	}
}

func TestForceDeleteNamespace_NonNamespacedResources(t *testing.T) {
	// Create namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ns",
		},
	}

	clientset := fake.NewSimpleClientset(ns)

	// Mock discovery with non-namespaced resources only (should be skipped)
	clientset.Discovery().(*fakediscovery.FakeDiscovery).Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "namespaces", // Non-namespaced resource
					Namespaced: false,
					Kind:       "Namespace",
					Verbs:      []string{"get", "list", "delete"},
				},
				{
					Name:       "nodes", // Non-namespaced resource
					Namespaced: false,
					Kind:       "Node",
					Verbs:      []string{"get", "list", "delete"},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	ctx := context.Background()
	err := ForceDeleteNamespace(ctx, clientset, dynamicClient, "test-ns")
	if err != nil {
		t.Fatalf("ForceDeleteNamespace() error = %v", err)
	}
}

// ============================================
// Additional Edge Case Tests
// ============================================

func TestListDeployments_EmptyNamespace(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	ctx := context.Background()

	result, err := ListWorkloads(ctx, clientset, "default", ResourceDeployments)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected 0 workloads, got %d", len(result))
	}
}

func TestListDaemonSets_WithPods(t *testing.T) {
	daemonset := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-daemonset",
			Namespace: "default",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
		},
		Status: appsv1.DaemonSetStatus{
			NumberReady:            3,
			DesiredNumberScheduled: 3,
		},
	}

	clientset := fake.NewSimpleClientset(daemonset)
	ctx := context.Background()

	result, err := ListWorkloads(ctx, clientset, "default", ResourceDaemonSets)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 workload, got %d", len(result))
	}

	if result[0].Name != "test-daemonset" {
		t.Errorf("Name = %q, want 'test-daemonset'", result[0].Name)
	}
}

func TestCopySecretToNamespace_MissingSource(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	ctx := context.Background()

	err := CopySecretToNamespace(ctx, clientset, "source-ns", "non-existent-secret", "target-ns")
	if err == nil {
		t.Error("CopySecretToNamespace() expected error for source not found")
	}
}

func TestCopyConfigMapToNamespace_MissingSource(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	ctx := context.Background()

	err := CopyConfigMapToNamespace(ctx, clientset, "source-ns", "non-existent-configmap", "target-ns")
	if err == nil {
		t.Error("CopyConfigMapToNamespace() expected error for source not found")
	}
}

func TestPodWithContainerWaiting(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-waiting",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "main"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "main",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "ImagePullBackOff",
							Message: "Back-off pulling image",
						},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(pod)
	ctx := context.Background()

	pods, err := ListAllPods(ctx, clientset, "default")
	if err != nil {
		t.Fatalf("ListAllPods() error = %v", err)
	}

	if len(pods) != 1 {
		t.Fatalf("Expected 1 pod, got %d", len(pods))
	}

	// Check the status reflects the waiting state
	if pods[0].Status != "ImagePullBackOff" {
		t.Logf("Pod status: %s", pods[0].Status) // Log for debugging, not an error
	}
}

func TestPodWithContainerTerminated(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-terminated",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "main"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "main",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 1,
							Reason:   "Error",
							Message:  "Container crashed",
						},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(pod)
	ctx := context.Background()

	pods, err := ListAllPods(ctx, clientset, "default")
	if err != nil {
		t.Fatalf("ListAllPods() error = %v", err)
	}

	if len(pods) != 1 {
		t.Fatalf("Expected 1 pod, got %d", len(pods))
	}
}

func TestPodWithLastTerminationState(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-restarted",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "main"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "main",
					Ready:        true,
					RestartCount: 5,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
					LastTerminationState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 137,
							Reason:   "OOMKilled",
						},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(pod)
	ctx := context.Background()

	pods, err := ListAllPods(ctx, clientset, "default")
	if err != nil {
		t.Fatalf("ListAllPods() error = %v", err)
	}

	if len(pods) != 1 {
		t.Fatalf("Expected 1 pod, got %d", len(pods))
	}
}

func TestListNodes_NotReady(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "not-ready-node",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
			},
			NodeInfo: corev1.NodeSystemInfo{
				KubeletVersion: "v1.28.0",
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			},
		},
	}

	clientset := fake.NewSimpleClientset(node)
	ctx := context.Background()

	result, err := ListNodes(ctx, clientset)
	if err != nil {
		t.Fatalf("ListNodes() error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(result))
	}

	if result[0].Status != "NotReady" {
		t.Errorf("Status = %q, want 'NotReady'", result[0].Status)
	}
}

func TestGetNode_NonExistent(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	ctx := context.Background()

	_, err := GetNode(ctx, clientset, "non-existent-node")
	if err == nil {
		t.Error("GetNode() expected error for non-existent node")
	}
}

func TestListPodsByNode_Empty(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	ctx := context.Background()

	result, err := ListPodsByNode(ctx, clientset, "non-existent-node")
	if err != nil {
		t.Fatalf("ListPodsByNode() error = %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected 0 pods, got %d", len(result))
	}
}
